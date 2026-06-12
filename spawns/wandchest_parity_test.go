package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

func scaled(x float64) int64 { return int64(math.Round(x * 1e6)) }

// wandSig matches wandSig() in wandchest_parity_gen.mjs.
func wandSig(w *Wand) string {
	if w == nil {
		return "none"
	}
	rare := 0
	if w.IsRare == 1 {
		rare = 1
	}
	parts := []string{
		w.Name, w.Sprite, strconv.Itoa(rare), strconv.Itoa(w.ShuffleDeckWhenEmpty),
		strconv.FormatInt(scaled(w.DeckCapacity), 10), strconv.FormatInt(scaled(w.ActionsPerRound), 10),
		strconv.FormatInt(scaled(w.ReloadTime), 10), strconv.FormatInt(scaled(w.FireRateWait), 10),
		strconv.FormatInt(scaled(w.SpreadDegrees), 10), strconv.FormatInt(scaled(w.SpeedMultiplier), 10),
		strconv.FormatInt(scaled(w.ManaMax), 10), strconv.FormatInt(scaled(w.ManaChargeSpeed), 10),
		strings.Join(w.AlwaysCasts, ","), strings.Join(w.Cards, ","),
	}
	return strings.Join(parts, "|")
}

func chestItemSig(it *Item) string {
	if it.Wand != nil {
		return "W:" + wandSig(it.Wand)
	}
	return fmt.Sprintf("%s|%s|%s|%d", it.ItemType, it.Material, it.Spell, it.Amount)
}

func chestSig(res *ChestResult) string {
	if res == nil || res.Items == nil {
		return "none"
	}
	sigs := make([]string, len(res.Items))
	for i, it := range res.Items {
		sigs[i] = chestItemSig(it)
	}
	return strings.Join(sigs, ";")
}

// wandShellSig drops actions_per_round (idx 5), always_casts (idx 12) and cards
// (idx 13) from a wandSig — the fields that diverge — leaving the wand identity
// and core stats (name, sprite, rare, shuffle, capacity, reload, fire rate,
// spread, speed, mana).
func wandShellSig(sig string) string {
	if sig == "none" {
		return sig
	}
	p := strings.Split(sig, "|")
	if len(p) < 14 {
		return sig
	}
	keep := []string{p[0], p[1], p[2], p[3], p[4], p[6], p[7], p[8], p[9], p[10], p[11]}
	return strings.Join(keep, "|")
}

// TestWandChestContentParity verifies the wand-stat shell and the non-wand chest
// loot (potions/spells/gold/markers) match the reference JS. NOTE: the wand
// spell deck (cards + always_casts) and actions_per_round currently DIVERGE from
// this telescope reference — the spell data and GetRandomActionWithType are
// byte-identical (verified), so the divergence is localized to the deck-fill
// logic in gun_generation; it's logged here as a known gap, not asserted.
func TestWandChestContentParity(t *testing.T) {
	data, err := os.ReadFile("wandchest_vectors.json")
	if err != nil {
		t.Fatalf("read wandchest_vectors.json (run `node wandchest_parity_gen.mjs`): %v", err)
	}
	var vecs []struct {
		Seed   uint32  `json:"seed"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		WandP5 string  `json:"wand_p5"`
		WandL1 string  `json:"wand_l1"`
		Chest  string  `json:"chest"`
		Great  string  `json:"great"`
	}
	if err := json.Unmarshal(data, &vecs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}

	var shellDiverge, deckDiverge, chestDiverge, total int
	for _, v := range vecs {
		// Premade wands (generateLevel1Wand) must match EXACTLY, including the
		// spell deck — this verifies the deck mechanism works for premades.
		if got := wandSig(generateWandByType(v.Seed, 0, v.X, v.Y, "premade_5", false)); got != v.WandP5 {
			t.Errorf("premade_5 (seed=%d,%g,%g) =\n  %q\nwant\n  %q", v.Seed, v.X, v.Y, got, v.WandP5)
		}

		// Measure (don't assert) the randomized generateGun divergence. It also
		// cascades into chests: a chest's randomized wand desyncs the main PRNG,
		// shifting later chest items — so chest contents can't be cleanly
		// separated from it here. (Potions/pouches/items/altars are verified
		// independently by TestLootContentParity.)
		l1 := wandSig(generateWandByType(v.Seed, 0, v.X, v.Y, "wand_level_01", false))
		total++
		if wandShellSig(l1) != wandShellSig(v.WandL1) {
			shellDiverge++
		}
		if l1 != v.WandL1 {
			deckDiverge++
		}
		if chestSig(GenerateChest(v.Seed, 0, v.X, v.Y, false, false)) != v.Chest {
			chestDiverge++
		}
	}
	t.Logf("VERIFIED: premade wands (full, incl. deck). Potions/items verified in TestLootContentParity.")
	if shellDiverge+deckDiverge+chestDiverge > 0 {
		t.Errorf("wand_level_01 diverges from telescope ref — shell %d/%d, deck %d/%d, chests %d/%d", shellDiverge, total, deckDiverge, total, chestDiverge, total)
	} else {
		t.Logf("VERIFIED: randomized wand_level_01 — shell %d/%d, deck %d/%d, chests %d/%d", shellDiverge, total, deckDiverge, total, chestDiverge, total)
	}
}
