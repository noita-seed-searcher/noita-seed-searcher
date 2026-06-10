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
	ID                       string   `json:"id"`
	Stackable                bool     `json:"stackable"`
	MaxInPerkPool            int      `json:"max_in_perk_pool"`
	StackableMaximum         int      `json:"stackable_maximum"`
	StackableIsRare          bool     `json:"stackable_is_rare"`
	StackableHowOftenReappears int    `json:"stackable_how_often_reappears"`
	NotInDefaultPerkPool     bool     `json:"not_in_default_perk_pool"`
	RemoveOtherPerks         []string `json:"remove_other_perks"`
}

type TempleLocation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

var perksArr []PerkData
var templeLocations []TempleLocation

func init() {
	// Perks are stored as an ordered array to preserve JS Object.values() order.
	if err := json.Unmarshal(perksJSON, &perksArr); err != nil {
		panic("failed to load perks.json: " + err.Error())
	}
	if err := json.Unmarshal(templeLocationsJSON, &templeLocations); err != nil {
		panic("failed to load temple_locations.json: " + err.Error())
	}
}

const (
	minDistanceBetweenDuplicatePerks = 4
	defaultMaxStackablePerkCount     = 128
)

type perkGlobal struct {
	values map[string]int
	flags  map[string]bool
}

func newPerkGlobal() *perkGlobal {
	return &perkGlobal{
		values: make(map[string]int),
		flags:  make(map[string]bool),
	}
}

func (g *perkGlobal) GetValue(key string, defaultVal int) int {
	if v, ok := g.values[key]; ok {
		return v
	}
	return defaultVal
}

func (g *perkGlobal) SetValue(key string, val int) {
	g.values[key] = val
}

func (g *perkGlobal) GameAddFlagRun(flag string) {
	g.flags[flag] = true
}

func (g *perkGlobal) HasFlagRun(flag string) bool {
	return g.flags[flag]
}

func getPerkPickedFlagName(perkID string) string {
	return "PERK_PICKED_" + perkID
}

func getPerkFlagName(perkID string) string {
	return "PERK_" + perkID
}

// PerkState holds per-run perk computation state.
type PerkState struct {
	rng    *RNG
	global *perkGlobal
}

func newPerkState(rng *RNG) *PerkState {
	return &PerkState{rng: rng, global: newPerkGlobal()}
}

// perkGetSpawnOrder generates the global perk deck for the current world seed.
// Returns (perkDeck, stackableCounts).
func (ps *PerkState) perkGetSpawnOrder() ([]string, map[string]int) {
	ps.rng.SetRandomSeed(1, 2)

	perkDeck := []string{}
	stackableDistances := map[string]int{}
	stackableCount := map[string]int{}

	for _, perkData := range perksArr {
		if perkData.NotInDefaultPerkPool {
			continue
		}

		perkName := perkData.ID
		howManyTimes := 1
		stackableDistances[perkName] = -1
		stackableCount[perkName] = -1

		if perkData.Stackable {
			maxPerks := ps.rng.RandomInt(1, 2)
			if perkData.MaxInPerkPool > 0 {
				maxPerks = ps.rng.RandomInt(1, int32(perkData.MaxInPerkPool))
			}

			if perkData.StackableMaximum > 0 {
				stackableCount[perkName] = perkData.StackableMaximum
			} else {
				stackableCount[perkName] = defaultMaxStackablePerkCount
			}

			if perkData.StackableIsRare {
				maxPerks = 1
			}

			dist := minDistanceBetweenDuplicatePerks
			if perkData.StackableHowOftenReappears > 0 {
				dist = perkData.StackableHowOftenReappears
			}
			stackableDistances[perkName] = dist

			howManyTimes = int(ps.rng.RandomInt(1, maxPerks))
		}

		for j := 0; j < howManyTimes; j++ {
			perkDeck = append(perkDeck, perkName)
		}
	}

	// Shuffle
	ps.shuffleTable(perkDeck)

	// Remove duplicates that are too close
	for i := len(perkDeck) - 1; i >= 0; i-- {
		perk := perkDeck[i]
		if stackableDistances[perk] != -1 {
			minDist := stackableDistances[perk]
			removeMe := false
			for ri := i - minDist; ri < i; ri++ {
				if ri >= 0 && perkDeck[ri] == perk {
					removeMe = true
					break
				}
			}
			if removeMe {
				perkDeck = append(perkDeck[:i], perkDeck[i+1:]...)
			}
		}
	}

	return perkDeck, stackableCount
}

func (ps *PerkState) shuffleTable(t []string) {
	for i := len(t) - 1; i >= 1; i-- {
		j := ps.rng.RandomInt(0, int32(i))
		t[i], t[j] = t[j], t[i]
	}
}

func (ps *PerkState) getPerkDeck() []string {
	deck, stackableCount := ps.perkGetSpawnOrder()

	// Remove picked perks (in fresh search context, nothing is picked)
	for i := range deck {
		perkName := deck[i]
		flagName := getPerkPickedFlagName(perkName)
		pickupCount := ps.global.GetValue(flagName+"_PICKUP_COUNT", 0)
		if pickupCount > 0 {
			stackCount := stackableCount[perkName]
			if stackCount == -1 || stackCount == 0 || pickupCount >= stackCount {
				deck[i] = ""
			}
		}
	}
	return deck
}

func (ps *PerkState) getNextPerk(perkDeck []string) string {
	nextIdx := ps.global.GetValue("TEMPLE_NEXT_PERK_INDEX", 0)
	perkID := ""
	if nextIdx < len(perkDeck) {
		perkID = perkDeck[nextIdx]
	}
	for perkID == "" {
		if nextIdx < len(perkDeck) {
			perkDeck[nextIdx] = "LEGGY_FEET"
		}
		nextIdx++
		if nextIdx >= len(perkDeck) {
			nextIdx = 0
		}
		if nextIdx < len(perkDeck) {
			perkID = perkDeck[nextIdx]
		}
	}
	nextIdx++
	if nextIdx >= len(perkDeck) {
		nextIdx = 0
	}
	ps.global.SetValue("TEMPLE_NEXT_PERK_INDEX", nextIdx)
	ps.global.GameAddFlagRun(getPerkFlagName(perkID))
	return perkID
}

func (ps *PerkState) getNextReroll(perkDeck []string) string {
	nextIdx := ps.global.GetValue("TEMPLE_REROLL_PERK_INDEX", len(perkDeck)-1)
	perkID := ""
	if nextIdx >= 0 && nextIdx < len(perkDeck) {
		perkID = perkDeck[nextIdx]
	}
	for perkID == "" {
		if nextIdx >= 0 && nextIdx < len(perkDeck) {
			perkDeck[nextIdx] = "LEGGY_FEET"
		}
		nextIdx--
		if nextIdx < 0 {
			nextIdx = len(perkDeck) - 1
		}
		if nextIdx >= 0 && nextIdx < len(perkDeck) {
			perkID = perkDeck[nextIdx]
		}
	}
	nextIdx--
	if nextIdx < 0 {
		nextIdx = len(perkDeck) - 1
	}
	ps.global.SetValue("TEMPLE_REROLL_PERK_INDEX", nextIdx)
	ps.global.GameAddFlagRun(getPerkFlagName(perkID))
	return perkID
}

func (ps *PerkState) generateRowFromDeck(deck []string, count int) []string {
	result := make([]string, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, ps.getNextPerk(deck))
	}
	return result
}

func (ps *PerkState) handlePerkPickup(perk string) {
	flagName := getPerkPickedFlagName(perk)
	ps.global.GameAddFlagRun(flagName)
	count := ps.global.GetValue(flagName+"_PICKUP_COUNT", 0)
	ps.global.SetValue(flagName+"_PICKUP_COUNT", count+1)

	if perk == "EXTRA_PERK" {
		ps.global.SetValue("TEMPLE_PERK_COUNT", ps.global.GetValue("TEMPLE_PERK_COUNT", 3)+1)
	}
}

// PerkIterator generates perk rows one at a time from a pre-built deck.
// After NewPerkIterator, calling NextRow() repeatedly yields rows without
// re-running the expensive deck shuffle — and callers can stop early.
type PerkIterator struct {
	ps   *PerkState
	deck []string
	done int // rows yielded so far
}

func NewPerkIterator(rng *RNG) *PerkIterator {
	ps := newPerkState(rng)
	ps.global.SetValue("TEMPLE_PERK_COUNT", 3)
	return &PerkIterator{ps: ps, deck: ps.getPerkDeck()}
}

// NextRow returns the next perk row, or nil when all temple rows are exhausted.
func (it *PerkIterator) NextRow() []string {
	if it.done >= len(templeLocations) {
		return nil
	}
	perkCount := it.ps.global.GetValue("TEMPLE_PERK_COUNT", 3)
	row := it.ps.generateRowFromDeck(it.deck, perkCount)
	it.done++
	return row
}

// GetPerks returns perk rows for the current world seed (up to all temple levels).
// The deck is generated once and reused across rows.
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
	ps := newPerkState(rng)
	return ps.getPerkDeck()
}
