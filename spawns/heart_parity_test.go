package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestHeartParity verifies spawnHeart's routing decision matches the reference
// JS across a coordinate sweep (heart / chest / great_chest / mimic /
// chest_leggy / none). Chest contents come from the existing chest generators
// and are not compared here (the JS reference stubs them).
func TestHeartParity(t *testing.T) {
	data, err := os.ReadFile("heart_vectors.json")
	if err != nil {
		t.Fatalf("read heart_vectors.json (run `node heart_parity_gen.mjs`): %v", err)
	}
	var vecs []struct {
		Seed uint32  `json:"seed"`
		NG   int     `json:"ng"`
		X    float64 `json:"x"`
		Y    float64 `json:"y"`
		Kind string  `json:"kind"`
	}
	if err := json.Unmarshal(data, &vecs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}

	for _, v := range vecs {
		got := "none"
		if s := spawnHeart(v.Seed, v.NG, v.X, v.Y, "coalmine"); s != nil {
			got = s.Kind
		}
		if got != v.Kind {
			t.Errorf("spawnHeart(seed=%d, %g, %g) = %q, want %q", v.Seed, v.X, v.Y, got, v.Kind)
		}
	}
}

// TestListNaturalSpawnsSmoke runs the full chain end-to-end and checks it yields
// item spawns of known kinds. (Item-content parity for altars/potions/wands
// rests on the pre-existing generators; the upstream buffer + scan are verified
// by the other parity tests.)
func TestListNaturalSpawnsSmoke(t *testing.T) {
	knownKinds := map[string]bool{
		"heart": true, "mimic": true, "chest_leggy": true, "jar": true,
		"chest": true, "great_chest": true, "wand": true, "potion": true,
		"item": true, "potion_altar": true, "wand_altar": true, "pixel_scene": true,
	}
	for _, seed := range []uint32{123456789, 42, 786433} {
		spawns, err := listNaturalSpawns(seed, 0, 0, 0)
		if err != nil {
			t.Fatalf("seed=%d: %v", seed, err)
		}
		if len(spawns) == 0 {
			t.Errorf("seed=%d: no spawns", seed)
		}
		counts := map[string]int{}
		for _, s := range spawns {
			if !knownKinds[s.Kind] {
				t.Errorf("seed=%d: unexpected kind %q from %s", seed, s.Kind, s.FuncName)
			}
			counts[s.Kind]++
		}
		t.Logf("seed=%d: %d item spawns %v", seed, len(spawns), counts)
	}
}
