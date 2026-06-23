package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// A true_orb only ever drops from the "very special (~impossible)" branch at the
// top of GenerateGreatChest (see chest.go): random(0,100000)>=100000 then
// random(0,1000)==999 -> true_orb, else sampo. That gate is ~1/100.1M per great
// chest, so this search is the rarest one in the tool. Because the gate is keyed
// on the chest's world (seed, x, y), every parallel world is an independent extra
// roll — which is exactly why this searcher takes a parallel-world range.
//
// In the main world the orb-bearing great chests live in the Mines (coalmine), so
// the search defaults to that biome; -biomes can widen it if you want.

// orbChestSourceFuncs are the spawn functions that directly produce a chest the
// orb roll can fire in: the three great-chest sources plus the two coalmine
// puzzles (which call spawnChest at an offset). Pixel-scene loaders are handled
// separately — they expand to one of these via coalmineSceneChestSpawns.
var orbChestSourceFuncs = map[string]bool{
	"spawn_chest":               true,
	"spawn_heart":               true,
	"spawn_bbqbox":              true,
	"spawn_swing_puzzle_target": true,
	"spawn_oiltank_puzzle":      true,
}

// chestHasTrueOrb reports whether a generated chest holds a true_orb. A great
// chest holds at most one (the very-special branch sets count=0 and appends a
// single item), so presence is all we check.
func chestHasTrueOrb(c *ChestResult) bool {
	if c == nil {
		return false
	}
	for _, it := range c.Items {
		if it.ItemType == "true_orb" {
			return true
		}
	}
	return false
}

// trueOrbsInRegions tiles each precomputed biome region once, then replays every
// great-chest spawn point across the requested parallel-world range, shifting the
// world coordinates per world so each (pw, pwV) is an independent orb roll. Only
// reads shared state, so it is safe to call from multiple goroutines.
func trueOrbsInRegions(seed uint32, ng, pwMax, pwMaxV int, t chestTarget, regions [][]point, bboxes [][4]int) []*Spawn {
	worldSize := 70 * 512
	if ng > 0 {
		worldSize = 64*512 - 8
	}
	const pwVSize = 24570

	var out []*Spawn
	for i := range bboxes {
		layer := generateTileLayer(bboxes[i], regions[i], t.ts, seed, ng, t.entry.biomeName, "normal", t.entry.randomColors)
		if layer == nil {
			continue
		}
		// Tiling + the spawn-point scan are parallel-world independent (the biome
		// map is shared), so resolve the chest-source points once and only shift
		// coordinates per world below. Pixel-scene loaders are kept too: in
		// coalmine they place chests inside the scene they load.
		isCoalmine := t.entry.biomeName == "coalmine"
		var dets []detectedSpawn
		for _, d := range prescanSpawnFunctions(layer, ng > 0, "normal") {
			if orbChestSourceFuncs[d.funcName] || (isCoalmine && pixelSceneLoaders[d.funcName]) {
				dets = append(dets, d)
			}
		}
		if len(dets) == 0 {
			continue
		}
		for pwV := -pwMaxV; pwV <= pwMaxV; pwV++ {
			for pw := -pwMax; pw <= pwMax; pw++ {
				dx := float64(pw * worldSize)
				dy := float64(pwV * pwVSize)
				for _, d := range dets {
					sx, sy := float64(d.x)+dx, float64(d.y)+dy
					// A pixel-scene loader resolves to the chest(s) baked into the
					// scene it selects; everything else is a chest spawn itself.
					var cands []chestCand
					if isCoalmine && pixelSceneLoaders[d.funcName] {
						cands = coalmineSceneChestSpawns(d.funcName, seed, ng, sx, sy)
					} else {
						cands = []chestCand{{d.funcName, sx, sy}}
					}
					for _, c := range cands {
						s := spawnSwitchItem(c.funcName, seed, ng, c.x, c.y, t.entry.biomeName, "normal")
						if s == nil || s.Kind != "great_chest" || !chestHasTrueOrb(s.Chest) {
							continue
						}
						s.Biome = t.entry.biomeName
						s.PW = pw
						s.PWV = pwV
						out = append(out, s)
					}
				}
			}
		}
	}
	return out
}

// seedTrueOrbs collects every true_orb great chest for a single seed across all
// target biomes and parallel worlds. Mirrors seedGreatChests: when static
// (ng==0) the seed-independent regions held on each target are reused, otherwise
// the seeded biome map is generated and scanned per seed.
func seedTrueOrbs(seed uint32, ng, pwMax, pwMaxV int, targets []chestTarget, static bool) []*Spawn {
	var bm *BiomeMap
	if !static {
		var err error
		bm, err = generateBiomeData(seed, ng, "normal")
		if err != nil {
			fmt.Fprintf(os.Stderr, "seed %d: %v\n", seed, err)
			return nil
		}
	}
	var out []*Spawn
	for _, t := range targets {
		regions, bboxes := t.regions, t.bboxes
		if !static {
			regions, bboxes = findBiomeRegions(bm.Pixels, bm.W, bm.H, t.entry.color)
		}
		out = append(out, trueOrbsInRegions(seed, ng, pwMax, pwMaxV, t, regions, bboxes)...)
	}
	return out
}

// searchTrueOrb scans the inclusive seed range [start, end] for seeds whose given
// biomes contain at least one true_orb great chest, optionally across a range of
// parallel worlds (±pwMax horizontal, ±pwMaxV vertical). Structure (ordered
// parallel batches, deterministic output, throttled progress line) matches
// searchGreatChest; see that function for the rationale.
func searchTrueOrb(ng int, start, end uint32, biomeNames []string, limit, pwMax, pwMaxV int, progress io.Writer) error {
	if len(biomeNames) == 0 {
		return fmt.Errorf("no biomes specified")
	}
	var targets []chestTarget
	for _, name := range biomeNames {
		entry := findBiomeEntry(name)
		if entry == nil {
			return fmt.Errorf("unknown biome %q", name)
		}
		if entry.wangFile == "" {
			return fmt.Errorf("biome %q has no tile generation (cannot contain chests)", name)
		}
		ts, err := buildBiomeTileset(entry.wangFile)
		if err != nil {
			return err
		}
		targets = append(targets, chestTarget{entry: entry, ts: ts})
	}

	// ng==0: the base biome map (and each biome's regions) is identical for every
	// seed, so compute it once instead of per seed in the hot loop.
	static := ng == 0
	if static {
		bm, err := generateBiomeData(start, ng, "normal")
		if err != nil {
			return err
		}
		for i := range targets {
			targets[i].regions, targets[i].bboxes = findBiomeRegions(bm.Pixels, bm.W, bm.H, targets[i].entry.color)
		}
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	biomeList := strings.Join(biomeNames, ", ")
	pwNote := ""
	if pwMax > 0 || pwMaxV > 0 {
		pwNote = fmt.Sprintf(", pw ±%d/±%d", pwMax, pwMaxV)
	}
	fmt.Printf("Searching seeds %d..%d for true_orb great_chest in %s%s (%d workers)...\n",
		start, end, biomeList, pwNote, workers)

	found := 0
	total := uint64(end) - uint64(start) + 1
	var scanned uint64
	startTime := time.Now()
	var lastDraw time.Time

	clearLine := func() {
		if progress != nil {
			fmt.Fprint(progress, "\r\033[K")
		}
	}
	drawProgress := func(curSeed uint32, force bool) {
		if progress == nil {
			return
		}
		now := time.Now()
		if !force && now.Sub(lastDraw) < 200*time.Millisecond {
			return
		}
		lastDraw = now
		elapsed := now.Sub(startTime).Seconds()
		rate := 0.0
		if elapsed > 0 {
			rate = float64(scanned) / elapsed
		}
		eta := math.Inf(1)
		if rate > 0 {
			eta = float64(total-scanned) / rate
		}
		fmt.Fprintf(progress, "\r\033[K  %d/%d seeds (%.0f%%)  %.0f seeds/s  seed=%d  found=%d  ETA %s",
			scanned, total, 100*float64(scanned)/float64(total), rate, curSeed, found, fmtETA(eta))
	}

	printSeed := func(seed uint32, spawns []*Spawn) bool {
		if len(spawns) == 0 {
			return false
		}
		clearLine()
		found++
		for _, sp := range spawns {
			fmt.Printf("seed %d: true_orb great_chest in %s%s @ (%.0f, %.0f) — %d item(s)\n",
				seed, sp.Biome, pwSuffix(sp), sp.X, sp.Y, len(sp.Chest.Items))
			for _, it := range sp.Chest.Items {
				if it.Count > 1 {
					fmt.Printf("    - x%d ", it.Count)
				} else {
					fmt.Printf("    - ")
				}
				printItem(it)
			}
		}
		return limit > 0 && found >= limit
	}

	processBatch := func(b0, b1 uint32) bool {
		n := int(b1 - b0 + 1)
		res := make([][]*Spawn, n)
		var idx int64
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					i := int(atomic.AddInt64(&idx, 1)) - 1
					if i >= n {
						return
					}
					res[i] = seedTrueOrbs(b0+uint32(i), ng, pwMax, pwMaxV, targets, static)
				}
			}()
		}
		wg.Wait()
		for i := 0; i < n; i++ {
			if printSeed(b0+uint32(i), res[i]) {
				scanned += uint64(i + 1)
				return true
			}
		}
		scanned += uint64(n)
		drawProgress(b1, false)
		return false
	}

	// Warm up lazy global caches by scanning the first seed alone before any
	// concurrent access touches them.
	done := processBatch(start, start)

	const batchPerWorker = 128
	batchSize := uint32(workers * batchPerWorker)
	for b0 := start + 1; !done && start < end && b0 <= end; {
		b1 := b0 + batchSize - 1
		if b1 > end || b1 < b0 {
			b1 = end
		}
		done = processBatch(b0, b1)
		if b1 == end {
			break
		}
		b0 = b1 + 1
	}

	clearLine()
	fmt.Printf("Done. %d seed(s) with true_orb great_chest in %s.\n", found, biomeList)
	return nil
}
