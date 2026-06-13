package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// WeightConfig maps item identifiers to score values.
// Keys match (case-insensitively) against: spell IDs in wand cards/always_cast,
// Item.Spell, Item.Material, and Item.ItemType.
type WeightConfig map[string]float64

func loadWeights(path string) (WeightConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comment = '#'
	r.FieldsPerRecord = 2
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	wc := make(WeightConfig, len(records))
	for i, rec := range records {
		v, err := strconv.ParseFloat(strings.TrimSpace(rec[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid weight %q: %w", i+1, rec[1], err)
		}
		wc[strings.ToUpper(strings.TrimSpace(rec[0]))] = v
	}
	return wc, nil
}

func (wc WeightConfig) get(key string) float64 {
	return wc[strings.ToUpper(key)]
}

// SpawnMatch is a single weighted hit within a spawn.
type SpawnMatch struct {
	Key    string  // matched identifier (spell ID, material, item type)
	Source string  // "cards", "always_cast", "spell", "material", "type", or prefixed with "chest › "
	Score  float64
}

func (wc WeightConfig) itemMatches(it *Item, prefix string) []SpawnMatch {
	if it == nil {
		return nil
	}
	var out []SpawnMatch
	add := func(key, source string) {
		if v := wc.get(key); v != 0 {
			out = append(out, SpawnMatch{strings.ToUpper(key), prefix + source, v})
		}
	}
	if it.Spell != "" {
		add(it.Spell, "spell")
	}
	if it.Material != "" {
		add(it.Material, "material")
	}
	add(it.ItemType, "type")
	if it.Wand != nil {
		for _, c := range it.Wand.Cards {
			add(c, "cards")
		}
		for _, c := range it.Wand.AlwaysCasts {
			add(c, "always_cast")
		}
	}
	return out
}

func (wc WeightConfig) SpawnMatches(s *Spawn) []SpawnMatch {
	var out []SpawnMatch
	out = append(out, wc.itemMatches(s.Item, "")...)
	if s.Chest != nil {
		for _, it := range s.Chest.Items {
			out = append(out, wc.itemMatches(it, "chest › ")...)
		}
	}
	return out
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
