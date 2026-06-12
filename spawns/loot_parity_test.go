package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// itemSig normalizes a generated Item to the same signature as sig() in
// loot_parity_gen.mjs: "<item>|<material>|<active>".
func itemSig(it *Item) string {
	if it == nil {
		return "none"
	}
	active := ""
	if it.Active {
		active = "A"
	}
	return fmt.Sprintf("%s|%s|%s", it.ItemType, it.Material, active)
}

// TestLootContentParity verifies the leaf loot generators (createPotion,
// createPowderPouch, SpawnItem, SpawnPotionAltar) produce the same contents as
// the reference JS — closing the trust gap on the items the dispatch emits.
func TestLootContentParity(t *testing.T) {
	data, err := os.ReadFile("loot_vectors.json")
	if err != nil {
		t.Fatalf("read loot_vectors.json (run `node loot_parity_gen.mjs`): %v", err)
	}
	var vecs []struct {
		Seed   uint32  `json:"seed"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Potion string  `json:"potion"`
		Pouch  string  `json:"pouch"`
		Item   string  `json:"item"`
		Altar  string  `json:"altar"`
	}
	if err := json.Unmarshal(data, &vecs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}

	for _, v := range vecs {
		if got := itemSig(createPotion(v.Seed, 0, v.X, v.Y, "normal", "normal")); got != v.Potion {
			t.Errorf("createPotion(seed=%d,%g,%g) = %q, want %q", v.Seed, v.X, v.Y, got, v.Potion)
		}
		if got := itemSig(createPowderPouch(v.Seed, 0, v.X, v.Y)); got != v.Pouch {
			t.Errorf("createPowderPouch(seed=%d,%g,%g) = %q, want %q", v.Seed, v.X, v.Y, got, v.Pouch)
		}
		if got := itemSig(SpawnItem(v.Seed, 0, v.X, v.Y, "coalmine", false)); got != v.Item {
			t.Errorf("SpawnItem(seed=%d,%g,%g) = %q, want %q", v.Seed, v.X, v.Y, got, v.Item)
		}
		if got := itemSig(SpawnPotionAltar(v.Seed, 0, v.X, v.Y, "coalmine", "normal", false)); got != v.Altar {
			t.Errorf("SpawnPotionAltar(seed=%d,%g,%g) = %q, want %q", v.Seed, v.X, v.Y, got, v.Altar)
		}
	}
}
