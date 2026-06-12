package main

import (
	"encoding/json"
	"os"
	"strings"
)

// WeightConfig maps item identifiers to score values.
// Keys match (case-insensitively) against: spell IDs in wand cards/always_cast,
// Item.Spell, Item.Material, and Item.ItemType.
type WeightConfig map[string]float64

func loadWeights(path string) (WeightConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]float64
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	wc := make(WeightConfig, len(raw))
	for k, v := range raw {
		wc[strings.ToUpper(k)] = v
	}
	return wc, nil
}

func (wc WeightConfig) get(key string) float64 {
	return wc[strings.ToUpper(key)]
}

func (wc WeightConfig) scoreItem(it *Item) float64 {
	if it == nil {
		return 0
	}
	var score float64
	if it.Spell != "" {
		score += wc.get(it.Spell)
	}
	if it.Material != "" {
		score += wc.get(it.Material)
	}
	score += wc.get(it.ItemType)
	if it.Wand != nil {
		for _, c := range it.Wand.Cards {
			score += wc.get(c)
		}
		for _, c := range it.Wand.AlwaysCasts {
			score += wc.get(c)
		}
	}
	return score
}

func (wc WeightConfig) scoreSpawn(s *Spawn) float64 {
	var score float64
	if s.Item != nil {
		score += wc.scoreItem(s.Item)
	}
	if s.Chest != nil {
		for _, it := range s.Chest.Items {
			score += wc.scoreItem(it)
		}
	}
	return score
}
