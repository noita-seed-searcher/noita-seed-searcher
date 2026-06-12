package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestAllWorldsParity verifies the spawn-point enumeration across all biomes
// and seeds against the reference JS prescanSpawnFunctions: total count,
// ordered hash of (index,x,y), and a sample of detected points.
func TestAllWorldsParity(t *testing.T) {
	data, err := os.ReadFile("allworlds_vectors.json")
	if err != nil {
		t.Fatalf("read allworlds_vectors.json (run `node biome_allworlds_parity_gen.mjs`): %v", err)
	}
	var root struct {
		Cases []struct {
			Seed   uint32 `json:"seed"`
			Biome  string `json:"biome"`
			Count  int    `json:"count"`
			Hash   uint32 `json:"hash"`
			Sample []struct {
				FuncName string `json:"funcName"`
				Index    int    `json:"index"`
				X        int    `json:"x"`
				Y        int    `json:"y"`
			} `json:"sample"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	vecs := root.Cases
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}

	bm, err := generateBiomeData(0, 0, "normal")
	if err != nil {
		t.Fatalf("generateBiomeData: %v", err)
	}
	tilesetCache := map[string]*stbhwTileset{}

	// Group vectors by (seed, biome) for aggregation check
	type key struct {
		seed  uint32
		biome string
	}
	seenKeys := map[key]int{}

	for idx, v := range vecs {
		var entry biomeEntry
		for _, e := range biomeConfig {
			if e.biomeName == v.Biome {
				entry = e
				break
			}
		}
		if entry.wangFile == "" {
			t.Errorf("biome %q not found in biomeConfig", v.Biome)
			continue
		}

		ts, ok := tilesetCache[entry.wangFile]
		if !ok {
			ts, err = buildBiomeTileset(entry.wangFile)
			if err != nil {
				t.Fatalf("buildBiomeTileset(%q): %v", entry.wangFile, err)
			}
			tilesetCache[entry.wangFile] = ts
		}

		regions, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, entry.color)

		// Determine which region index this vector corresponds to by region count
		k := key{v.Seed, v.Biome}
		regionIdx := seenKeys[k]
		seenKeys[k]++

		if regionIdx >= len(bboxes) {
			t.Errorf("seed=%d biome=%s: region index %d >= total regions %d", v.Seed, v.Biome, regionIdx, len(bboxes))
			continue
		}

		layer := generateTileLayer(bboxes[regionIdx], regions[regionIdx], ts, v.Seed, 0, v.Biome, "normal", entry.randomColors)
		if layer == nil {
			t.Errorf("seed=%d biome=%s region %d: nil layer", v.Seed, v.Biome, regionIdx)
			continue
		}

		detected := prescanSpawnFunctions(layer, false, "normal")

		if len(detected) != v.Count {
			t.Errorf("vec[%d] seed=%d biome=%s region %d: count=%d want=%d", idx, v.Seed, v.Biome, regionIdx, len(detected), v.Count)
		}
		if h := hashScanList(detected); h != v.Hash {
			t.Errorf("vec[%d] seed=%d biome=%s region %d: hash=%d want=%d", idx, v.Seed, v.Biome, regionIdx, h, v.Hash)
		}
	}

	biomeSeen := map[string]bool{}
	for _, v := range vecs {
		biomeSeen[v.Biome] = true
	}
	t.Logf("VERIFIED: %d biome-seed cases across %d biomes", len(vecs), len(biomeSeen))
}
