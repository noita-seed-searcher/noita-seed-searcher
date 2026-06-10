package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
)

//go:embed data/spells.json
var spellsJSON []byte

// SpellData holds the search-relevant fields of a spell.
type SpellData struct {
	ID                 string             `json:"id"`
	Type               int                `json:"type"` // ACTION_TYPE
	SpawnProbabilities map[string]float64 `json:"spawn_probabilities"`
}

var spellsArr []SpellData

func init() {
	if err := json.Unmarshal(spellsJSON, &spellsArr); err != nil {
		panic("failed to load spells.json: " + err.Error())
	}
}

// ACTION_TYPE constants matching Wand.ts
const (
	ACTION_PROJECTILE        = 0
	ACTION_STATIC_PROJECTILE = 1
	ACTION_MODIFIER          = 2
	ACTION_DRAW_MANY         = 3
	ACTION_MATERIAL          = 4
	ACTION_OTHER             = 5
	ACTION_UTILITY           = 6
	ACTION_PASSIVE           = 7
)

func spawnProbability(spell *SpellData, level int) float64 {
	key := fmt.Sprintf("%d", level)
	if p, ok := spell.SpawnProbabilities[key]; ok {
		return p
	}
	return 0
}

// fround converts x,y to float32 then back, matching TS Math.fround used in seededRandom wrapper.
func fround(v float64) float64 { return float64(float32(v)) }

// GetRandomAction mirrors GetRandomAction from random.ts.
func GetRandomAction(rng *RNG, x, y float64, level int, offset int) string {
	seed := uint32(int(rng.worldSeed) + offset)
	fx, fy := fround(x), fround(y)

	var sum float64
	for i := range spellsArr {
		sum += spawnProbability(&spellsArr[i], level)
	}

	accumulated := sum * rng.SeededRandom(seed, fx, fy)
	for i := range spellsArr {
		p := spawnProbability(&spellsArr[i], level)
		if p == 0 {
			continue
		}
		if p >= accumulated {
			return spellsArr[i].ID
		}
		accumulated -= p
	}
	if len(spellsArr) > 0 {
		return spellsArr[0].ID
	}
	return ""
}

// GetRandomActionWithType mirrors GetRandomActionWithType from random.ts.
func GetRandomActionWithType(rng *RNG, x, y float64, level, actionType int, offset int) string {
	seed := uint32(int(rng.worldSeed) + offset)
	fx, fy := fround(x), fround(y)

	var sum float64
	for i := range spellsArr {
		if spellsArr[i].Type != actionType {
			continue
		}
		sum += spawnProbability(&spellsArr[i], level)
	}

	accumulated := sum * rng.SeededRandom(seed, fx, fy)
	for i := range spellsArr {
		spell := &spellsArr[i]
		if spell.Type != actionType {
			continue
		}
		p := spawnProbability(spell, level)
		if p > 0 && p >= accumulated {
			return spell.ID
		}
		accumulated -= p
	}

	// Fallback: find any spell of the right type
	rand := int(math.Trunc(rng.SeededRandom(seed, fx, fy) * float64(len(spellsArr))))
	for j := 0; j < len(spellsArr); j++ {
		spell := &spellsArr[(j+rand)%len(spellsArr)]
		if spell.Type == actionType && spawnProbability(spell, level) > 0 {
			return spell.ID
		}
	}

	if len(spellsArr) > 0 {
		return spellsArr[0].ID
	}
	return ""
}
