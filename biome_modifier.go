package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed data/biome_modifiers.json
var biomeModifiersJSON []byte

type BiomeModifierData struct {
	ID                  string   `json:"id"`
	Probability         float64  `json:"probability"`
	DoesNotApplyToBiome []string `json:"does_not_apply_to_biome"`
	ApplyOnlyToBiome    []string `json:"apply_only_to_biome"`
	RequiresFlag        string   `json:"requires_flag"`
}

var biomeModifiersMap map[string]BiomeModifierData
var biomeModifiersList []BiomeModifierData

func init() {
	if err := json.Unmarshal(biomeModifiersJSON, &biomeModifiersMap); err != nil {
		panic("failed to load biome_modifiers.json: " + err.Error())
	}
	for _, m := range biomeModifiersMap {
		biomeModifiersList = append(biomeModifiersList, m)
	}
}

var biomeLayers = [][]string{
	{"coalmine", "mountain_hall"},
	{"coalmine_alt"},
	{"excavationsite"},
	{"fungicave"},
	{"snowcave"},
	{"snowcastle"},
	{"rainforest", "rainforest_open"},
	{"vault"},
	{"crypt"},
}

const (
	chanceOfModifierPerBiome   = 0.1
	chanceOfModifierCoalmine   = 0.2
	chanceOfModifierExcavation = 0.15
	chanceOfMoistFungicave     = 0.5
	chanceOfMoistLake          = 0.75
)

func biomeModifierAppliesToBiome(mod BiomeModifierData, biomeName string) bool {
	for _, skip := range mod.DoesNotApplyToBiome {
		if skip == biomeName {
			return false
		}
	}
	if len(mod.ApplyOnlyToBiome) > 0 {
		for _, required := range mod.ApplyOnlyToBiome {
			if required == biomeName {
				return true
			}
		}
		return false
	}
	return true
}

// GetBiomeModifiers computes the biome modifiers for a world seed.
// Returns a map of biome name -> modifier ID (empty string if none).
func GetBiomeModifiers(rng *RNG) map[string]string {
	result := map[string]string{}

	rnd := randomCreate(347893, 90734)

	probabilities := make([]float64, len(biomeModifiersList))
	for i, m := range biomeModifiersList {
		probabilities[i] = m.Probability
	}

	for _, biomeNames := range biomeLayers {
		biome0 := biomeNames[0]
		chance := chanceOfModifierPerBiome
		if biome0 == "coalmine" {
			chance = chanceOfModifierCoalmine
		} else if biome0 == "excavationsite" {
			chance = chanceOfModifierExcavation
		}

		hasModifier := rng.randomNext(&rnd, 0.0, 1.0) <= chance
		var modIdx int = -1
		if hasModifier {
			modIdx = pickRandomFromTableWeightedIdx(probabilities, &rnd, rng.worldSeed)
		}

		for _, biomeName := range biomeNames {
			if modIdx >= 0 && biomeModifierAppliesToBiome(biomeModifiersList[modIdx], biomeName) {
				result[biomeName] = biomeModifiersList[modIdx].ID
			}
		}
	}

	// Fungicave moist chance
	if rng.randomNext(&rnd, 0.0, 1.0) < chanceOfMoistFungicave {
		if _, exists := result["fungicave"]; !exists {
			result["fungicave"] = "MOIST"
		}
	}

	// Fixed modifiers
	result["wandcave"] = "FOG_OF_WAR_CLEAR_AT_PLAYER"
	result["wizardcave"] = "FOG_OF_WAR_CLEAR_AT_PLAYER"
	result["alchemist_secret"] = "FOG_OF_WAR_CLEAR_AT_PLAYER"

	setIfNone := func(biome, mod string) {
		if _, exists := result[biome]; !exists {
			result[biome] = mod
		}
	}

	setIfNone("mountain_top", "FREEZING")
	setIfNone("mountain_floating_island", "FREEZING")
	setIfNone("winter", "FREEZING")
	result["winter_caves"] = "FREEZING_COSMETIC"

	setIfNone("lavalake", "HOT")
	setIfNone("desert", "HOT")
	setIfNone("pyramid_entrance", "HOT")
	setIfNone("pyramid_left", "HOT")
	setIfNone("pyramid_top", "HOT")
	setIfNone("pyramid_right", "HOT")

	setIfNone("watercave", "MOIST")

	if rng.randomNext(&rnd, 0.0, 1.0) < chanceOfMoistLake {
		setIfNone("lake_statue", "MOIST")
	}

	return result
}
