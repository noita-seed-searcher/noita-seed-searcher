package main

import (
	"encoding/json"
	"os"
	"testing"
)

func hashBytes(arr []byte) uint32 {
	h := uint32(2166136261)
	for _, v := range arr {
		h ^= uint32(v)
		h *= 16777619
	}
	return h
}

func hashI32(arr []int32) uint32 {
	h := uint32(2166136261)
	for _, v := range arr {
		h ^= uint32(v)
		h *= 16777619
	}
	return h
}

// TestWangDecodeParity verifies Go's loadWangRGB matches the independent PIL
// decode in wang_base.json.
func TestWangDecodeParity(t *testing.T) {
	data, err := os.ReadFile("wang_base.json")
	if err != nil {
		t.Fatalf("read wang_base.json (run `python3 wang_base_gen.py`): %v", err)
	}
	var bases map[string]struct {
		W   int   `json:"w"`
		H   int   `json:"h"`
		RGB []int `json:"rgb"`
	}
	if err := json.Unmarshal(data, &bases); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := bases["coalmine"]
	got, w, h, err := loadWangRGB("data/wang_tiles/coalmine.png")
	if err != nil {
		t.Fatalf("loadWangRGB: %v", err)
	}
	if w != want.W || h != want.H {
		t.Fatalf("dims = %dx%d, want %dx%d", w, h, want.W, want.H)
	}
	if len(got) != len(want.RGB) {
		t.Fatalf("len = %d, want %d", len(got), len(want.RGB))
	}
	for i := range got {
		if int(got[i]) != want.RGB[i] {
			t.Fatalf("byte %d = %d, want %d", i, got[i], want.RGB[i])
		}
	}
}

type tileVectors struct {
	Tileset struct {
		IsCorner     bool    `json:"isCorner"`
		NumColor     []int32 `json:"numColor"`
		ShortSideLen int     `json:"shortSideLen"`
		NumVaryX     int     `json:"numVaryX"`
		NumVaryY     int     `json:"numVaryY"`
		NumHTiles    int     `json:"numHTiles"`
		NumVTiles    int     `json:"numVTiles"`
	} `json:"tileset"`
	Regions []struct {
		Bbox      [4]int `json:"bbox"`
		NumPoints int    `json:"numPoints"`
	} `json:"regions"`
	Cases []struct {
		Seed            uint32 `json:"seed"`
		NG              int    `json:"ng"`
		RegionIdx       int    `json:"regionIdx"`
		Bbox            [4]int `json:"bbox"`
		Width           int    `json:"width"`
		Height          int    `json:"height"`
		Xmax            int    `json:"xmax"`
		Ymax            int    `json:"ymax"`
		BufferHash      uint32 `json:"bufferHash"`
		TileIndicesHash uint32 `json:"tileIndicesHash"`
	} `json:"cases"`
}

const coalmineColor uint32 = 0xffd57917

// TestTileGenParity verifies the Wang tileset build, coalmine region detection,
// and the pre-hack raw tile buffer (stbhw_generate_image) all match the JS.
func TestTileGenParity(t *testing.T) {
	data, err := os.ReadFile("tile_vectors.json")
	if err != nil {
		t.Fatalf("read tile_vectors.json (run `node biome_tile_parity_gen.mjs`): %v", err)
	}
	var v tileVectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// 1. Tileset build parity.
	ts, err := buildBiomeTileset("data/wang_tiles/coalmine.png")
	if err != nil {
		t.Fatalf("buildBiomeTileset: %v", err)
	}
	if ts.isCorner != v.Tileset.IsCorner || ts.shortSideLen != v.Tileset.ShortSideLen ||
		ts.numVaryX != v.Tileset.NumVaryX || ts.numVaryY != v.Tileset.NumVaryY ||
		ts.numHTiles != v.Tileset.NumHTiles || ts.numVTiles != v.Tileset.NumVTiles {
		t.Errorf("tileset meta = {corner:%v ssl:%d vary:%dx%d h:%d v:%d}, want {corner:%v ssl:%d vary:%dx%d h:%d v:%d}",
			ts.isCorner, ts.shortSideLen, ts.numVaryX, ts.numVaryY, ts.numHTiles, ts.numVTiles,
			v.Tileset.IsCorner, v.Tileset.ShortSideLen, v.Tileset.NumVaryX, v.Tileset.NumVaryY, v.Tileset.NumHTiles, v.Tileset.NumVTiles)
	}
	for i := 0; i < 4; i++ {
		if ts.numColor[i] != v.Tileset.NumColor[i] {
			t.Errorf("numColor[%d] = %d, want %d", i, ts.numColor[i], v.Tileset.NumColor[i])
		}
	}

	// 2. Region detection parity (on Go's embedded NG0 map).
	bm, err := generateBiomeData(0, 0, "normal")
	if err != nil {
		t.Fatalf("generateBiomeData: %v", err)
	}
	_, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, coalmineColor)
	if len(bboxes) != len(v.Regions) {
		t.Fatalf("region count = %d, want %d", len(bboxes), len(v.Regions))
	}
	regions, _ := findBiomeRegions(bm.Pixels, bm.W, bm.H, coalmineColor)
	for i := range bboxes {
		if bboxes[i] != v.Regions[i].Bbox {
			t.Errorf("region[%d] bbox = %v, want %v", i, bboxes[i], v.Regions[i].Bbox)
		}
		if len(regions[i]) != v.Regions[i].NumPoints {
			t.Errorf("region[%d] numPoints = %d, want %d", i, len(regions[i]), v.Regions[i].NumPoints)
		}
	}

	// 3. Raw tile buffer parity per case.
	for _, c := range v.Cases {
		raw := generateRawTileBuffer(c.Bbox, ts, c.Seed, c.NG, 0)
		if raw == nil {
			t.Errorf("seed=%d region=%d: nil raw buffer", c.Seed, c.RegionIdx)
			continue
		}
		if raw.width != c.Width || raw.height != c.Height || raw.xmax != c.Xmax || raw.ymax != c.Ymax {
			t.Errorf("seed=%d region=%d: dims {w:%d h:%d xmax:%d ymax:%d}, want {w:%d h:%d xmax:%d ymax:%d}",
				c.Seed, c.RegionIdx, raw.width, raw.height, raw.xmax, raw.ymax, c.Width, c.Height, c.Xmax, c.Ymax)
			continue
		}
		if h := hashBytes(raw.buffer); h != c.BufferHash {
			t.Errorf("seed=%d region=%d: bufferHash = %d, want %d", c.Seed, c.RegionIdx, h, c.BufferHash)
		}
		if h := hashI32(raw.tileIndices); h != c.TileIndicesHash {
			t.Errorf("seed=%d region=%d: tileIndicesHash = %d, want %d", c.Seed, c.RegionIdx, h, c.TileIndicesHash)
		}
	}
}
