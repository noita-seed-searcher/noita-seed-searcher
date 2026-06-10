package main

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestAlchemySeed123(t *testing.T) {
	r := GetAlchemyResult(123)
	expected := AlchemyResult{
		LC: [3]string{"lava", "magic_liquid_invisibility", "diamond"},
		AP: [3]string{"swamp", "magic_liquid_movement_faster", "wax"},
	}
	if r != expected {
		t.Fatalf("alchemy mismatch:\ngot:      %v\nexpected: %v", r, expected)
	}
}

func TestPerkSeed123(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	rows := GetPerks(rng)

	expected := [][]string{
		{"LOW_HP_DAMAGE_BOOST", "PROJECTILE_HOMING", "MOLD"},
		{"BLEED_GAS", "RADAR_ENEMY", "FASTER_LEVITATION"},
		{"GLASS_CANNON", "PROJECTILE_REPULSION_SECTOR", "ORBIT"},
		{"TELEPORTITIS", "REMOVE_FOG_OF_WAR", "SAVING_GRACE"},
		{"RESPAWN", "PERSONAL_LASER", "TELEKINESIS"},
		{"FAST_PROJECTILES", "BOUNCE", "BLEED_OIL"},
		{"HOMUNCULUS", "PEACE_WITH_GODS", "PROJECTILE_EATER_SECTOR"},
	}

	if !reflect.DeepEqual(rows, expected) {
		t.Fatalf("perk mismatch for seed 123:\ngot:      %v\nexpected: %v", rows, expected)
	}
}

func TestPerkSeed806(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(806)
	rows := GetPerks(rng)

	expectedRows0and1 := [][]string{
		{"BLEED_GAS", "PROTECTION_ELECTRICITY", "VAMPIRISM"},
		{"BLEED_OIL", "EXTRA_PERK", "PROTECTION_RADIOACTIVITY"},
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least 2 rows, got %d", len(rows))
	}
	for i, expected := range expectedRows0and1 {
		if !reflect.DeepEqual(rows[i], expected) {
			t.Fatalf("perk row %d mismatch for seed 806:\ngot:      %v\nexpected: %v", i, rows[i], expected)
		}
	}
}

func TestFungalShiftSeed123(t *testing.T) {
	shifts := PickFungal(123, 20)
	if len(shifts) < 6 {
		t.Fatalf("expected at least 6 shifts, got %d", len(shifts))
	}
	expected0 := FungalShift{
		FlaskTo: false, FlaskFrom: true,
		From: []string{"lava"}, To: "sand",
		GoldToX: "gold", GrassToX: "grass_holy",
	}
	expected5 := FungalShift{
		FlaskTo: true, FlaskFrom: false,
		From:    []string{"radioactive_liquid", "poison", "material_darkness"},
		To:      "acid",
		GoldToX: "mammi", GrassToX: "grass",
	}
	if !reflect.DeepEqual(shifts[0], expected0) {
		t.Fatalf("fungal shift 0 mismatch:\ngot:      %+v\nexpected: %+v", shifts[0], expected0)
	}
	if !reflect.DeepEqual(shifts[5], expected5) {
		t.Fatalf("fungal shift 5 mismatch:\ngot:      %+v\nexpected: %+v", shifts[5], expected5)
	}
}

func TestStartingFlask(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	flask := GetStartingFlask(rng)
	fmt.Printf("Seed 123 starting flask: %s\n", flask)
	if flask == "" || flask == "unknown" {
		t.Fatalf("unexpected flask: %s", flask)
	}
}

// --- Wand tests (reference values from Wand.spec.ts) ---

func floatEq(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestWandSeed123Level1(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	wand := ProvideWand(rng, 10, 10, 20, 1, false, false)
	g := wand.Gun

	checks := map[string][2]float64{
		"actions_per_round":      {g.ActionsPerRound, 1},
		"deck_capacity":          {g.DeckCapacity, 6},
		"fire_rate_wait":         {g.FireRateWait, 18},
		"force_unshuffle":        {g.ForceUnshuffle, 0},
		"is_rare":                {g.IsRare, 0},
		"mana_charge_speed":      {g.ManaChargeSpeed, 245},
		"mana_max":               {g.ManaMax, 80},
		"prob_draw_many":         {g.ProbDrawMany, 0.15},
		"prob_unshuffle":         {g.ProbUnshuffle, 0.1},
		"reload_time":            {g.ReloadTime, 28},
		"shuffle_deck_when_empty": {g.ShuffleDeckWhenEmpty, 1},
		"speed_multiplier":       {g.SpeedMultiplier, 1.08233642578125},
		"spread_degrees":         {g.SpreadDegrees, 2},
	}
	for name, pair := range checks {
		if !floatEq(pair[0], pair[1], 1e-9) {
			t.Errorf("%s: got %.10f want %.10f", name, pair[0], pair[1])
		}
	}

	wantCards := []string{"BURST_2", "HEAVY_SPREAD", "DYNAMITE", "DYNAMITE", "DYNAMITE"}
	if !reflect.DeepEqual(wand.Cards.Cards, wantCards) {
		t.Errorf("cards:\ngot:  %v\nwant: %v", wand.Cards.Cards, wantCards)
	}
}

func TestWandSeed123Level6(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	wand := ProvideWand(rng, 155, 6161, 200, 6, false, false)
	g := wand.Gun

	checks := map[string][2]float64{
		"actions_per_round":      {g.ActionsPerRound, 4},
		"cost":                   {g.Cost, 0},
		"fire_rate_wait":         {g.FireRateWait, 13},
		"force_unshuffle":        {g.ForceUnshuffle, 0},
		"is_rare":                {g.IsRare, 0},
		"mana_charge_speed":      {g.ManaChargeSpeed, 304},
		"mana_max":               {g.ManaMax, 980},
		"prob_draw_many":         {g.ProbDrawMany, 0.15},
		"prob_unshuffle":         {g.ProbUnshuffle, 0.1},
		"reload_time":            {g.ReloadTime, 38},
		"shuffle_deck_when_empty": {g.ShuffleDeckWhenEmpty, 1},
		"speed_multiplier":       {g.SpeedMultiplier, 1.09279203414917},
		"spread_degrees":         {g.SpreadDegrees, 4},
	}
	for name, pair := range checks {
		if !floatEq(pair[0], pair[1], 1e-6) {
			t.Errorf("%s: got %.10f want %.10f", name, pair[0], pair[1])
		}
	}

	wantCards := []string{
		"EXPLODING_DEER", "GRENADE_LARGE", "ROCKET_TIER_3", "GRENADE_TIER_3",
		"MAGIC_SHIELD", "PIPE_BOMB_DEATH_TRIGGER", "SLOW_BULLET_TRIGGER",
		"BLACK_HOLE_DEATH_TRIGGER", "GRENADE_TRIGGER", "GRENADE_TRIGGER",
		"HITFX_CRITICAL_BLOOD", "EXPLOSION_REMOVE", "NUKE", "SUMMON_ROCK",
		"TNTBOX_BIG", "METEOR",
	}
	if !reflect.DeepEqual(wand.Cards.Cards, wantCards) {
		t.Errorf("cards:\ngot:  %v\nwant: %v", wand.Cards.Cards, wantCards)
	}
}

func TestWandSeed123Level6ForceUnshuffle(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	wand := ProvideWand(rng, 155, 6161, 200, 6, true, false)
	if wand.Gun.ShuffleDeckWhenEmpty != 0 {
		t.Errorf("shuffle_deck_when_empty: got %.0f want 0", wand.Gun.ShuffleDeckWhenEmpty)
	}
	// Cards must be identical to the non-unshuffle case
	wantCards := []string{
		"EXPLODING_DEER", "GRENADE_LARGE", "ROCKET_TIER_3", "GRENADE_TIER_3",
		"MAGIC_SHIELD", "PIPE_BOMB_DEATH_TRIGGER", "SLOW_BULLET_TRIGGER",
		"BLACK_HOLE_DEATH_TRIGGER", "GRENADE_TRIGGER", "GRENADE_TRIGGER",
		"HITFX_CRITICAL_BLOOD", "EXPLOSION_REMOVE", "NUKE", "SUMMON_ROCK",
		"TNTBOX_BIG", "METEOR",
	}
	if !reflect.DeepEqual(wand.Cards.Cards, wantCards) {
		t.Errorf("cards:\ngot:  %v\nwant: %v", wand.Cards.Cards, wantCards)
	}
}

// --- Shop tests (reference values from Shop.spec.ts) ---

func TestShopSeed123Row0Type(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	shop := ProvideShopLevel(rng, 0, false)
	if shop.Type != ShopTypeWand {
		t.Errorf("row 0 type: got %v want ShopTypeWand (%v)", shop.Type, ShopTypeWand)
	}
}

func TestShopSeed123Row2Items(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	shop := ProvideShopLevel(rng, 2, false)
	if shop.Type != ShopTypeItem {
		t.Fatalf("row 2 type: got %v want ShopTypeItem", shop.Type)
	}
	want := []string{
		"RECHARGE", "CLOUD_ACID", "TELEPORT_PROJECTILE_CLOSER", "MANA_REDUCE",
		"EXPLOSION_TINY", "ROCKET_OCTAGON", "PROPANE_TANK", "GRENADE_TRIGGER",
		"BULLET_TIMER", "LIGHTNING",
	}
	got := make([]string, len(shop.Items))
	for i, item := range shop.Items {
		got[i] = item.SpellID
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row 2 items:\ngot:  %v\nwant: %v", got, want)
	}
}

func TestShopSeed123Row6Items(t *testing.T) {
	rng := newRNG()
	rng.SetWorldSeed(123)
	shop := ProvideShopLevel(rng, 6, false)
	if shop.Type != ShopTypeItem {
		t.Fatalf("row 6 type: got %v want ShopTypeItem", shop.Type)
	}
	want := []string{
		"RANDOM_EXPLOSION", "ROCKET_TIER_3", "HOMING_CURSOR", "CURSE_WITHER_MELEE",
		"RECHARGE", "RECOIL_DAMPER", "BLACK_HOLE_BIG", "HEAVY_SPREAD",
		"BULLET_TIMER", "MINE",
	}
	got := make([]string, len(shop.Items))
	for i, item := range shop.Items {
		got[i] = item.SpellID
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row 6 items:\ngot:  %v\nwant: %v", got, want)
	}
}
