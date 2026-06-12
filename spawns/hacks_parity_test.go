package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestOverlayDecodeParity verifies Go's loadOverlay matches the PIL decode.
func TestOverlayDecodeParity(t *testing.T) {
	data, err := os.ReadFile("overlay_base.json")
	if err != nil {
		t.Fatalf("read overlay_base.json (run `python3 overlay_base_gen.py`): %v", err)
	}
	var bases map[string]struct {
		W    int   `json:"w"`
		H    int   `json:"h"`
		RGBA []int `json:"rgba"`
	}
	if err := json.Unmarshal(data, &bases); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := bases["coalmine"]
	ov, err := loadOverlay("data/wang_tiles/extra_layers/coalmine.png")
	if err != nil {
		t.Fatalf("loadOverlay: %v", err)
	}
	if ov.w != want.W || ov.h != want.H {
		t.Fatalf("dims = %dx%d, want %dx%d", ov.w, ov.h, want.W, want.H)
	}
	if len(ov.data) != len(want.RGBA) {
		t.Fatalf("len = %d, want %d", len(ov.data), len(want.RGBA))
	}
	for i := range ov.data {
		if int(ov.data[i]) != want.RGBA[i] {
			t.Fatalf("byte %d = %d, want %d", i, ov.data[i], want.RGBA[i])
		}
	}
}

type layerVectors struct {
	Regions [][4]int `json:"regions"`
	Cases   []struct {
		Seed       uint32 `json:"seed"`
		NG         int    `json:"ng"`
		RegionIdx  int    `json:"regionIdx"`
		Bbox       [4]int `json:"bbox"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		MapH       int    `json:"mapH"`
		Attempts   int    `json:"attempts"`
		PathLen    int    `json:"pathLen"`
		BufferHash uint32 `json:"bufferHash"`
	} `json:"cases"`
}

// TestLayerGenParity verifies the full coalmine pipeline (block-out rooms, main
// biome hack, coalmine hack, pathfinding reroll loop, room restore, undo hack,
// postprocessing, masking) produces the same final layer buffer as the JS.
func TestLayerGenParity(t *testing.T) {
	data, err := os.ReadFile("layer_vectors.json")
	if err != nil {
		t.Fatalf("read layer_vectors.json (run `node biome_layer_parity_gen.mjs`): %v", err)
	}
	var v layerVectors
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
	regions, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, biomeConfig[0].color)

	for _, c := range v.Cases {
		layer := generateTileLayer(bboxes[c.RegionIdx], regions[c.RegionIdx], ts, c.Seed, c.NG, "coalmine", "normal", nil)
		if layer == nil {
			t.Errorf("seed=%d region=%d: nil layer", c.Seed, c.RegionIdx)
			continue
		}
		if layer.width != c.Width || layer.height != c.Height || layer.mapH != c.MapH {
			t.Errorf("seed=%d: dims {w:%d h:%d mapH:%d}, want {w:%d h:%d mapH:%d}",
				c.Seed, layer.width, layer.height, layer.mapH, c.Width, c.Height, c.MapH)
			continue
		}
		if layer.attempts != c.Attempts {
			t.Errorf("seed=%d: attempts = %d, want %d", c.Seed, layer.attempts, c.Attempts)
		}
		if len(layer.path) != c.PathLen {
			t.Errorf("seed=%d: pathLen = %d, want %d", c.Seed, len(layer.path), c.PathLen)
		}
		if h := hashBytes(layer.buffer); h != c.BufferHash {
			t.Errorf("seed=%d: bufferHash = %d, want %d", c.Seed, h, c.BufferHash)
		}
	}
}
