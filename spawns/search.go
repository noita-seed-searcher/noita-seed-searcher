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

// fmtETA renders a remaining-seconds estimate as a compact duration.
func fmtETA(secs float64) string {
	if secs <= 0 || math.IsInf(secs, 0) || math.IsNaN(secs) {
		return "?"
	}
	d := time.Duration(secs * float64(time.Second))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// findBiomeEntry returns the biomeConfig entry for a biome name, or nil.
func findBiomeEntry(biomeName string) *biomeEntry {
	for i := range biomeConfig {
		if biomeConfig[i].biomeName == biomeName {
			return &biomeConfig[i]
		}
	}
	return nil
}

// greatChestSpawnFuncs are the spawn functions that can yield a great_chest:
// spawn_chest (via SpawnChest's rare-roll) and spawn_heart / spawn_bbqbox (both
// route through spawnHeart, which has its own great_chest branch). Restricting
// dispatch to these keeps the search cheap while still covering every source —
// e.g. excavationsite has no spawn_chest, only spawn_heart.
var greatChestSpawnFuncs = map[string]bool{
	"spawn_chest":  true,
	"spawn_heart":  true,
	"spawn_bbqbox": true,
}

// chestTarget pairs a tile-generating biome with its (shared, read-only)
// tileset, decoded once up front and reused across every seed and worker.
type chestTarget struct {
	entry *biomeEntry
	ts    *stbhwTileset
}

// chestHearts sums the heart drops in a generated chest. Counts are post-dedup,
// so it.Count is authoritative. heart and heart_bigger are the two HP-raising
// drops (full_heal only heals and is not counted here).
func chestHearts(c *ChestResult) (hearts, biggerHearts int) {
	for _, it := range c.Items {
		n := it.Count
		if n == 0 {
			n = 1
		}
		switch it.ItemType {
		case "heart":
			hearts += n
		case "heart_bigger":
			biggerHearts += n
		}
	}
	return hearts, biggerHearts
}

// greatChestsInBiomeMap returns every great_chest for a seed inside one biome,
// reusing a biome map that the caller generated once for the seed. It skips
// every spawn function that cannot produce a great_chest, so it is cheap enough
// to run across a large seed range. It only reads shared state (the tileset and
// package config tables), so it is safe to call from multiple goroutines.
func greatChestsInBiomeMap(bm *BiomeMap, seed uint32, ng, minHearts int, t chestTarget) []*Spawn {
	var out []*Spawn
	regions, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, t.entry.color)
	for i := range bboxes {
		layer := generateTileLayer(bboxes[i], regions[i], t.ts, seed, ng, t.entry.biomeName, "normal", t.entry.randomColors)
		if layer == nil {
			continue
		}
		for _, d := range prescanSpawnFunctions(layer, ng > 0, "normal") {
			if !greatChestSpawnFuncs[d.funcName] {
				continue
			}
			s := spawnSwitchItem(d.funcName, seed, ng, float64(d.x), float64(d.y), t.entry.biomeName, "normal")
			if s == nil || s.Kind != "great_chest" {
				continue
			}
			if minHearts > 0 {
				h, b := chestHearts(s.Chest)
				if h+b < minHearts {
					continue
				}
			}
			s.Biome = t.entry.biomeName
			out = append(out, s)
		}
	}
	return out
}

// seedGreatChests collects every great_chest for a single seed across all target
// biomes. The biome map is generated once and shared across targets (a free win
// when searching several biomes). Errors are seed-local and logged to stderr.
func seedGreatChests(seed uint32, ng, minHearts int, targets []chestTarget) []*Spawn {
	bm, err := generateBiomeData(seed, ng, "normal")
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed %d: %v\n", seed, err)
		return nil
	}
	var out []*Spawn
	for _, t := range targets {
		out = append(out, greatChestsInBiomeMap(bm, seed, ng, minHearts, t)...)
	}
	return out
}

// searchGreatChest scans the inclusive seed range [start, end] for seeds whose
// given biomes contain at least one great_chest, printing each hit. A seed
// counts once toward limit even if it has hits in several biomes. limit stops
// the search after that many matching seeds (0 = no limit).
//
// Seeds are scanned in parallel across GOMAXPROCS workers in ordered batches,
// so output (and limit behaviour) stays deterministic regardless of worker
// scheduling.
//
// progress receives a live, in-place status line (seeds/s + ETA). It is meant
// for the console and is kept separate from the result stream so that, when
// results are redirected to a file via -out, the progress never lands in it.
func searchGreatChest(ng int, start, end uint32, biomeNames []string, limit, minHearts int, progress io.Writer) error {
	if len(biomeNames) == 0 {
		return fmt.Errorf("no biomes specified")
	}
	// Resolve every biome and build its tileset once, up front.
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
		targets = append(targets, chestTarget{entry, ts})
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	biomeList := strings.Join(biomeNames, ", ")
	heartFilter := ""
	if minHearts > 0 {
		heartFilter = fmt.Sprintf(", >=%d hearts", minHearts)
	}
	fmt.Printf("Searching seeds %d..%d for great_chest in %s%s (%d workers)...\n",
		start, end, biomeList, heartFilter, workers)

	found := 0
	total := uint64(end) - uint64(start) + 1
	var scanned uint64
	startTime := time.Now()
	var lastDraw time.Time

	// clearLine wipes the in-place progress line so a result (or the final
	// summary) prints cleanly. No-op when progress is nil.
	clearLine := func() {
		if progress != nil {
			fmt.Fprint(progress, "\r\033[K")
		}
	}
	// drawProgress refreshes the live status line, throttled to ~5/s.
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

	// printSeed emits a seed's hits in order and reports whether limit is reached.
	printSeed := func(seed uint32, spawns []*Spawn) bool {
		if len(spawns) == 0 {
			return false
		}
		clearLine()
		found++
		for _, sp := range spawns {
			h, b := chestHearts(sp.Chest)
			fmt.Printf("seed %d: great_chest in %s @ (%.0f, %.0f) — %d item(s)  [%d heart, %d heart_bigger]\n",
				seed, sp.Biome, sp.X, sp.Y, len(sp.Chest.Items), h, b)
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

	// processBatch scans the inclusive seed range [b0, b1] with a worker pool,
	// then prints results in seed order. Returns true if limit was reached.
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
					res[i] = seedGreatChests(b0+uint32(i), ng, minHearts, targets)
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

	// Warm up lazy global caches (e.g. coalmineOverlayCache) by scanning the
	// first seed alone before any concurrent access touches them.
	done := processBatch(start, start)

	// Scan the remainder in parallel, in ordered batches bounded in size so
	// memory stays flat and -limit stops promptly.
	const batchPerWorker = 128
	batchSize := uint32(workers * batchPerWorker)
	for b0 := start + 1; !done && start < end && b0 <= end; {
		b1 := b0 + batchSize - 1
		if b1 > end || b1 < b0 { // clamp and guard uint32 overflow
			b1 = end
		}
		done = processBatch(b0, b1)
		if b1 == end {
			break
		}
		b0 = b1 + 1
	}

	clearLine()
	fmt.Printf("Done. %d seed(s) with great_chest in %s.\n", found, biomeList)
	return nil
}
