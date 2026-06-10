package main

import (
	"fmt"
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
