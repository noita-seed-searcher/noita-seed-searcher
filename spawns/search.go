package main

import (
	"fmt"
	"os"
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

// greatChestsInBiome returns every great_chest that spawns for the given seed
// inside a single biome. It mirrors the per-biome slice of listNaturalSpawns
// but skips every other biome and every spawn function except spawn_chest, so
// it is cheap enough to run across a large seed range. The tileset is passed in
// (built once by the caller) to avoid re-decoding the wang PNG per seed.
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
			if d.funcName != "spawn_chest" {
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
// given biome contains at least one great_chest, printing each hit. limit stops
// the search after that many matching seeds (0 = no limit).
func searchGreatChest(ng int, start, end uint32, biomeName string, limit int) error {
	entry := findBiomeEntry(biomeName)
	if entry == nil {
		return fmt.Errorf("unknown biome %q", biomeName)
	}
	if entry.wangFile == "" {
		return fmt.Errorf("biome %q has no tile generation (cannot contain chests)", biomeName)
	}
	ts, err := buildBiomeTileset(entry.wangFile)
	if err != nil {
		return err
	}

	fmt.Printf("Searching seeds %d..%d for great_chest in %s...\n", start, end, biomeName)
	found := 0
	// Loop on s so it terminates correctly even when end == math.MaxUint32.
	for s := start; ; s++ {
		spawns, err := greatChestsInBiome(s, ng, entry, ts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "seed %d: %v\n", s, err)
		} else if len(spawns) > 0 {
			found++
			for _, sp := range spawns {
				fmt.Printf("seed %d: great_chest in %s @ (%.0f, %.0f) — %d item(s)\n",
					s, sp.Biome, sp.X, sp.Y, len(sp.Chest.Items))
				for _, it := range sp.Chest.Items {
					fmt.Printf("    - ")
					printItem(it)
				}
			}
			if limit > 0 && found >= limit {
				break
			}
		}
		if s == end {
			break
		}
	}
	fmt.Printf("Done. %d seed(s) with great_chest in %s.\n", found, biomeName)
	return nil
}
