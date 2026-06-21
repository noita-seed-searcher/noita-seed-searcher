package main

import (
	"fmt"
	"os"
	"strings"
)

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

// greatChestsInBiome returns every great_chest that spawns for the given seed
// inside a single biome. It mirrors the per-biome slice of listNaturalSpawns
// but skips every other biome and every spawn function that cannot produce a
// great_chest, so it is cheap enough to run across a large seed range. The
// tileset is passed in (built once by the caller) to avoid re-decoding the wang
// PNG per seed.
func greatChestsInBiome(seed uint32, ng int, entry *biomeEntry, ts *stbhwTileset) ([]*Spawn, error) {
	bm, err := generateBiomeData(seed, ng, "normal")
	if err != nil {
		return nil, err
	}
	var out []*Spawn
	regions, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, entry.color)
	for i := range bboxes {
		layer := generateTileLayer(bboxes[i], regions[i], ts, seed, ng, entry.biomeName, "normal", entry.randomColors)
		if layer == nil {
			continue
		}
		for _, d := range prescanSpawnFunctions(layer, ng > 0, "normal") {
			if !greatChestSpawnFuncs[d.funcName] {
				continue
			}
			s := spawnSwitchItem(d.funcName, seed, ng, float64(d.x), float64(d.y), entry.biomeName, "normal")
			if s != nil && s.Kind == "great_chest" {
				s.Biome = entry.biomeName
				out = append(out, s)
			}
		}
	}
	return out, nil
}

// searchGreatChest scans the inclusive seed range [start, end] for seeds whose
// given biomes contain at least one great_chest, printing each hit. A seed
// counts once toward limit even if it has hits in several biomes. limit stops
// the search after that many matching seeds (0 = no limit).
func searchGreatChest(ng int, start, end uint32, biomeNames []string, limit int) error {
	if len(biomeNames) == 0 {
		return fmt.Errorf("no biomes specified")
	}
	// Resolve every biome and build its tileset once, up front.
	type target struct {
		entry *biomeEntry
		ts    *stbhwTileset
	}
	var targets []target
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
		targets = append(targets, target{entry, ts})
	}

	fmt.Printf("Searching seeds %d..%d for great_chest in %s...\n",
		start, end, strings.Join(biomeNames, ", "))
	found := 0
	// Loop on s so it terminates correctly even when end == math.MaxUint32.
	for s := start; ; s++ {
		hit := false
		for _, t := range targets {
			spawns, err := greatChestsInBiome(s, ng, t.entry, t.ts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "seed %d (%s): %v\n", s, t.entry.biomeName, err)
				continue
			}
			for _, sp := range spawns {
				hit = true
				fmt.Printf("seed %d: great_chest in %s @ (%.0f, %.0f) — %d item(s)\n",
					s, sp.Biome, sp.X, sp.Y, len(sp.Chest.Items))
				for _, it := range sp.Chest.Items {
					fmt.Printf("    - ")
					printItem(it)
				}
			}
		}
		if hit {
			found++
			if limit > 0 && found >= limit {
				break
			}
		}
		if s == end {
			break
		}
	}
	fmt.Printf("Done. %d seed(s) with great_chest in %s.\n", found, strings.Join(biomeNames, ", "))
	return nil
}
