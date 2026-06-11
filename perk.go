package main

import (
	_ "embed"
	"encoding/json"
)

//go:embed data/perks.json
var perksJSON []byte

//go:embed data/temple_locations.json
var templeLocationsJSON []byte

type PerkData struct {
	ID                         string   `json:"id"`
	Stackable                  bool     `json:"stackable"`
	MaxInPerkPool              int      `json:"max_in_perk_pool"`
	StackableMaximum           int      `json:"stackable_maximum"`
	StackableIsRare            bool     `json:"stackable_is_rare"`
	StackableHowOftenReappears int      `json:"stackable_how_often_reappears"`
	NotInDefaultPerkPool       bool     `json:"not_in_default_perk_pool"`
	RemoveOtherPerks           []string `json:"remove_other_perks"`
}

type TempleLocation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

var perksArr []PerkData
var templeLocations []TempleLocation

// stackableDistMap is precomputed from perksArr: perk ID → min distance between copies (-1 = non-stackable).
// Avoids a per-seed map allocation in buildPerkDeck.
var stackableDistMap map[string]int

// row0PerkX/Y are precomputed coordinates for the 3 perk slots at temple level 0.
// Used in checkPerkShopSeed's cheap lottery pre-filter to avoid per-seed float arithmetic.
var row0PerkX [3]float64
var row0PerkY float64

func init() {
	if err := json.Unmarshal(perksJSON, &perksArr); err != nil {
		panic("failed to load perks.json: " + err.Error())
	}
	if err := json.Unmarshal(templeLocationsJSON, &templeLocations); err != nil {
		panic("failed to load temple_locations.json: " + err.Error())
	}

	stackableDistMap = make(map[string]int, len(perksArr))
	for _, p := range perksArr {
		dist := -1
		if p.Stackable {
			dist = minDistanceBetweenDuplicatePerks
			if p.StackableHowOftenReappears > 0 {
				dist = p.StackableHowOftenReappears
			}
		}
		stackableDistMap[p.ID] = dist
	}

	if len(templeLocations) > 0 {
		t := templeLocations[0]
		row0PerkY = t.Y
		for i := 0; i < 3; i++ {
			rawX := t.X + (float64(i)+0.5)*(60.0/3.0)
			row0PerkX[i] = float64(roundHalfToEvenI32(rawX))
		}
	}
}

const (
	minDistanceBetweenDuplicatePerks = 4
	defaultMaxStackablePerkCount     = 128
)

// perkState holds per-seed perk generation state.
type perkState struct {
	rng         *RNG
	nextPerkIdx int
	perkCount   int
}

func newPerkState(rng *RNG) *perkState {
	return &perkState{rng: rng, perkCount: 3}
}

// buildPerkDeck builds and shuffles the perk deck for the current world seed.
func (ps *perkState) buildPerkDeck() []string {
	ps.rng.SetRandomSeed(1, 2)

	deck := make([]string, 0, 120)
	for _, p := range perksArr {
		if p.NotInDefaultPerkPool {
			continue
		}
		howManyTimes := 1
		if p.Stackable {
			maxPerks := ps.rng.RandomInt(1, 2)
			if p.MaxInPerkPool > 0 {
				maxPerks = ps.rng.RandomInt(1, int32(p.MaxInPerkPool))
			}
			if p.StackableIsRare {
				maxPerks = 1
			}
			howManyTimes = int(ps.rng.RandomInt(1, maxPerks))
		}
		for j := 0; j < howManyTimes; j++ {
			deck = append(deck, p.ID)
		}
	}

	for i := len(deck) - 1; i >= 1; i-- {
		j := ps.rng.RandomInt(0, int32(i))
		deck[i], deck[j] = deck[j], deck[i]
	}

	// Remove stackable duplicates that are too close together.
	for i := len(deck) - 1; i >= 0; i-- {
		dist := stackableDistMap[deck[i]]
		if dist == -1 {
			continue
		}
		for ri := i - dist; ri < i; ri++ {
			if ri >= 0 && deck[ri] == deck[i] {
				deck = append(deck[:i], deck[i+1:]...)
				break
			}
		}
	}

	return deck
}

func (ps *perkState) nextPerk(deck []string) string {
	idx := ps.nextPerkIdx
	if idx >= len(deck) {
		idx = 0
	}
	ps.nextPerkIdx = idx + 1
	return deck[idx]
}

func (ps *perkState) generateRow(deck []string) []string {
	row := make([]string, ps.perkCount)
	for i := range row {
		row[i] = ps.nextPerk(deck)
	}
	return row
}

// PerkIterator generates perk rows one at a time from a pre-built deck.
// After NewPerkIterator, each NextRow() call is cheap — no re-shuffling.
type PerkIterator struct {
	ps   *perkState
	deck []string
	done int
}

func NewPerkIterator(rng *RNG) *PerkIterator {
	ps := newPerkState(rng)
	return &PerkIterator{ps: ps, deck: ps.buildPerkDeck()}
}

// NextRow returns the next perk row, or nil when all temple rows are exhausted.
func (it *PerkIterator) NextRow() []string {
	if it.done >= len(templeLocations) {
		return nil
	}
	row := it.ps.generateRow(it.deck)
	it.done++
	return row
}

// GetPerks returns all perk rows for the current world seed.
func GetPerks(rng *RNG) [][]string {
	it := NewPerkIterator(rng)
	result := make([][]string, 0, len(templeLocations))
	for {
		row := it.NextRow()
		if row == nil {
			break
		}
		result = append(result, row)
	}
	return result
}

// GetPerkDeck returns the full shuffled perk deck for the current world seed.
func GetPerkDeck(rng *RNG) []string {
	return newPerkState(rng).buildPerkDeck()
}
