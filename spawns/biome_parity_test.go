package main

import (
	"encoding/json"
	"os"
	"testing"
)

// hash32 replicates the FNV-1a-ish rolling hash in biome_parity_gen.mjs.
func hash32(arr []uint32) uint32 {
	h := uint32(2166136261)
	for _, v := range arr {
		h ^= v
		h *= 16777619
	}
	return h
}

type biomeBaseEntry struct {
	W    int      `json:"w"`
	H    int      `json:"h"`
	ARGB []uint32 `json:"argb"`
}

// TestBiomeDecodeParity verifies Go's embedded-PNG decode (loadBiomeBase)
// matches the independent PIL decode in biome_base.json.
func TestBiomeDecodeParity(t *testing.T) {
	data, err := os.ReadFile("biome_base.json")
	if err != nil {
		t.Fatalf("read biome_base.json (run `python3 biome_base_gen.py`): %v", err)
	}
	var bases map[string]biomeBaseEntry
	if err := json.Unmarshal(data, &bases); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, tc := range []struct {
		key, file string
		w, h      int
	}{
		{"normal", "data/biome_maps/biome_map.png", biomeWNG0, biomeH},
		{"ngp", "data/biome_maps/biome_map_newgame_plus.png", biomeWNGP, biomeH},
		{"nightmare", "data/biome_maps/biome_map_nightmare.png", biomeWNGP, biomeH},
	} {
		want, ok := bases[tc.key]
		if !ok {
			t.Fatalf("missing base %q", tc.key)
		}
		got, err := loadBiomeBase(tc.file, tc.w, tc.h)
		if err != nil {
			t.Fatalf("%s: loadBiomeBase: %v", tc.key, err)
		}
		if len(got) != len(want.ARGB) {
			t.Fatalf("%s: len = %d, want %d", tc.key, len(got), len(want.ARGB))
		}
		for i := range got {
			if got[i] != want.ARGB[i] {
				t.Fatalf("%s: pixel %d = %#08x, want %#08x", tc.key, i, got[i], want.ARGB[i])
			}
		}
	}
}

type biomeVector struct {
	Seed     uint32 `json:"seed"`
	NG       int    `json:"ng"`
	GameMode string `json:"gameMode"`
	W        int    `json:"w"`
	H        int    `json:"h"`
	Pixels   []uint32
	Orbs     []struct {
		X    int    `json:"x"`
		Y    int    `json:"y"`
		Name string `json:"name"`
	} `json:"orbs"`
	HeavenHash uint32 `json:"heavenHash"`
	HellHash   uint32 `json:"hellHash"`
}

// TestBiomeGenParity verifies generateBiomeData matches the reference JS
// across the cases in biome_vectors.json (static + procedural NG+/nightmare).
func TestBiomeGenParity(t *testing.T) {
	data, err := os.ReadFile("biome_vectors.json")
	if err != nil {
		t.Fatalf("read biome_vectors.json (run `node biome_parity_gen.mjs`): %v", err)
	}
	var vecs []biomeVector
	if err := json.Unmarshal(data, &vecs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}

	for _, v := range vecs {
		bm, err := generateBiomeData(v.Seed, v.NG, v.GameMode)
		if err != nil {
			t.Fatalf("seed=%d ng=%d %s: %v", v.Seed, v.NG, v.GameMode, err)
		}
		tag := func() string {
			return jsonTag(v.Seed, v.NG, v.GameMode)
		}
		if bm.W != v.W || bm.H != v.H {
			t.Errorf("%s: dims = %dx%d, want %dx%d", tag(), bm.W, bm.H, v.W, v.H)
			continue
		}
		if len(bm.Pixels) != len(v.Pixels) {
			t.Errorf("%s: pixel len = %d, want %d", tag(), len(bm.Pixels), len(v.Pixels))
			continue
		}
		mismatch := 0
		for i := range bm.Pixels {
			if bm.Pixels[i] != v.Pixels[i] {
				if mismatch < 3 {
					t.Errorf("%s: pixel %d (x=%d,y=%d) = %#08x, want %#08x",
						tag(), i, i%v.W, i/v.W, bm.Pixels[i], v.Pixels[i])
				}
				mismatch++
			}
		}
		if mismatch > 0 {
			t.Errorf("%s: %d pixel mismatches total", tag(), mismatch)
		}

		if len(bm.Orbs) != len(v.Orbs) {
			t.Errorf("%s: orb count = %d, want %d", tag(), len(bm.Orbs), len(v.Orbs))
		} else {
			for i, o := range bm.Orbs {
				w := v.Orbs[i]
				if o.X != w.X || o.Y != w.Y || o.Name != w.Name {
					t.Errorf("%s: orb[%d] = (%d,%d,%q), want (%d,%d,%q)",
						tag(), i, o.X, o.Y, o.Name, w.X, w.Y, w.Name)
				}
			}
		}

		if h := hash32(bm.Heaven); h != v.HeavenHash {
			t.Errorf("%s: heavenHash = %d, want %d", tag(), h, v.HeavenHash)
		}
		if h := hash32(bm.Hell); h != v.HellHash {
			t.Errorf("%s: hellHash = %d, want %d", tag(), h, v.HellHash)
		}
	}
}

func jsonTag(seed uint32, ng int, mode string) string {
	b, _ := json.Marshal(struct {
		Seed uint32 `json:"seed"`
		NG   int    `json:"ng"`
		Mode string `json:"mode"`
	}{seed, ng, mode})
	return string(b)
}
