package main

import (
	"encoding/json"
	"os"
	"testing"
)

// hashScanList replicates hashScan in biome_layer_parity_gen.mjs.
func hashScanList(list []detectedSpawn) uint32 {
	h := uint32(2166136261)
	for _, s := range list {
		for _, v := range []int{s.index, s.x, s.y} {
			h ^= uint32(int32(v))
			h *= 16777619
		}
	}
	return h
}

type scanVectors struct {
	Cases []struct {
		Seed      uint32         `json:"seed"`
		NG        int            `json:"ng"`
		RegionIdx int            `json:"regionIdx"`
		Count     int            `json:"count"`
		ScanHash  uint32         `json:"scanHash"`
		Counts    map[string]int `json:"counts"`
		Sample    []struct {
			FuncName string `json:"funcName"`
			Index    int    `json:"index"`
			X        int    `json:"x"`
			Y        int    `json:"y"`
		} `json:"sample"`
	} `json:"cases"`
}

// TestScanParity verifies the natural-spawn-point enumeration over the final
// coalmine layer buffer matches the reference JS prescanSpawnFunctions: total
// count, per-funcName counts, an ordered hash of (index,x,y), and a sample.
func TestScanParity(t *testing.T) {
	data, err := os.ReadFile("scan_vectors.json")
	if err != nil {
		t.Fatalf("read scan_vectors.json (run `node biome_layer_parity_gen.mjs`): %v", err)
	}
	var v scanVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	ts, err := buildBiomeTileset("data/wang_tiles/coalmine.png")
	if err != nil {
		t.Fatalf("buildBiomeTileset: %v", err)
	}
	bm, err := generateBiomeData(0, 0, "normal")
	if err != nil {
		t.Fatalf("generateBiomeData: %v", err)
	}
	regions, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, coalmineColor)

	for _, c := range v.Cases {
		layer := generateTileLayer(bboxes[c.RegionIdx], regions[c.RegionIdx], ts, c.Seed, c.NG, "coalmine", "normal", nil)
		if layer == nil {
			t.Errorf("seed=%d: nil layer", c.Seed)
			continue
		}
		detected := prescanSpawnFunctions(layer, false, "normal")

		if len(detected) != c.Count {
			t.Errorf("seed=%d: spawn count = %d, want %d", c.Seed, len(detected), c.Count)
		}
		if h := hashScanList(detected); h != c.ScanHash {
			t.Errorf("seed=%d: scanHash = %d, want %d", c.Seed, h, c.ScanHash)
		}

		// Per-funcName counts.
		got := map[string]int{}
		for _, s := range detected {
			got[s.funcName]++
		}
		for name, want := range c.Counts {
			if got[name] != want {
				t.Errorf("seed=%d: count[%s] = %d, want %d", c.Seed, name, got[name], want)
			}
		}
		if len(got) != len(c.Counts) {
			t.Errorf("seed=%d: %d distinct funcNames, want %d", c.Seed, len(got), len(c.Counts))
		}

		// Sample of first entries.
		for i, s := range c.Sample {
			if i >= len(detected) {
				break
			}
			d := detected[i]
			if d.funcName != s.FuncName || d.index != s.Index || d.x != s.X || d.y != s.Y {
				t.Errorf("seed=%d: spawn[%d] = {%s idx=%d (%d,%d)}, want {%s idx=%d (%d,%d)}",
					c.Seed, i, d.funcName, d.index, d.x, d.y, s.FuncName, s.Index, s.X, s.Y)
			}
		}
	}
}
