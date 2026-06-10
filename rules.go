package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Rule types match the TypeScript IRule / ILogicRules.
const (
	RuleTypeAND   = "and"
	RuleTypeOR    = "or"
	RuleTypeNOT   = "not"
	RuleTypeRules = "rules"
)

// RuleNode is a single node in the rule tree.
type RuleNode struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Rules []*RuleNode     `json:"rules,omitempty"`
	Val   json.RawMessage `json:"val,omitempty"`
}

// Checker evaluates rules against a seed.
type Checker struct {
	rng *RNG
}

func newChecker() *Checker {
	return &Checker{rng: newRNG()}
}

func (c *Checker) SetSeed(seed uint32) {
	c.rng.SetWorldSeed(seed)
}

func (c *Checker) Check(rule *RuleNode) bool {
	switch rule.Type {
	case RuleTypeAND:
		for _, r := range rule.Rules {
			if !c.Check(r) {
				return false
			}
		}
		return true
	case RuleTypeOR:
		for _, r := range rule.Rules {
			if c.Check(r) {
				return true
			}
		}
		return false
	case RuleTypeNOT:
		if len(rule.Rules) == 0 {
			return true
		}
		return !c.Check(rule.Rules[0])
	default:
		return c.checkLeaf(rule)
	}
}

func (c *Checker) checkLeaf(rule *RuleNode) bool {
	switch rule.Type {
	case "alchemy":
		return c.checkAlchemy(rule)
	case "fungalShift":
		return c.checkFungalShift(rule)
	case "perk":
		return c.checkPerk(rule)
	case "startingFlask":
		return c.checkStartingFlask(rule)
	case "startingSpell":
		return c.checkStartingSpell(rule)
	case "startingBombSpell":
		return c.checkStartingBombSpell(rule)
	case "biomeModifier":
		return c.checkBiomeModifier(rule)
	case "weather":
		return c.checkWeather(rule)
	case "waterCave":
		return true // not yet filtered
	case "map":
		return true // map is too complex for now
	case "wand":
		return true // wand is too complex for now
	case "shop":
		return true // shop is too complex for now
	case "spells":
		return true
	case "lottery":
		return true
	case "biome":
		return true
	case "material":
		return true
	case "potion":
		return true
	case "potionSecret":
		return true
	case "potionRandomMaterial":
		return true
	case "powderStash":
		return true
	case "chestRandom":
		return true
	case "pacifistChest":
		return true
	case "alwaysCast":
		return true
	case "excavationsiteCubeChamber":
		return true
	case "snowcaveSecretChamber":
		return true
	case "snowcastleSecretChamber":
		return true
	default:
		fmt.Printf("Warning: unknown rule type %q, skipping\n", rule.Type)
		return true
	}
}

// --- Alchemy ---

type AlchemyRuleVal struct {
	LC []string `json:"LC"`
	AP []string `json:"AP"`
}

func includesAll(haystack []string, needles []string) bool {
	for _, n := range needles {
		found := false
		for _, h := range haystack {
			if h == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (c *Checker) checkAlchemy(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val AlchemyRuleVal
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	result := GetAlchemyResult(c.rng.worldSeed)
	if !includesAll(result.LC[:], val.LC) {
		return false
	}
	if !includesAll(result.AP[:], val.AP) {
		return false
	}
	return true
}

// --- Fungal Shift ---

type FungalRuleEntry struct {
	From      []string `json:"from"`
	To        []string `json:"to"`
	FlaskFrom *bool    `json:"flaskFrom"`
	FlaskTo   *bool    `json:"flaskTo"`
	GoldToX   string   `json:"gold_to_x"`
	GrassToX  string   `json:"grass_to_x"`
}

func includesSome(haystack []string, needles []string) bool {
	for _, n := range needles {
		for _, h := range haystack {
			if h == n {
				return true
			}
		}
	}
	return false
}

func (c *Checker) checkFungalShift(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val []FungalRuleEntry
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}

	shifts := PickFungal(c.rng.worldSeed, 20)

	for i, entry := range val {
		if entry.FlaskTo == nil && entry.FlaskFrom == nil &&
			len(entry.From) == 0 && len(entry.To) == 0 &&
			entry.GoldToX == "" && entry.GrassToX == "" {
			continue
		}
		if i >= len(shifts) {
			return false
		}
		shift := shifts[i]

		if entry.FlaskTo != nil && *entry.FlaskTo != shift.FlaskTo {
			return false
		}
		if entry.FlaskFrom != nil && *entry.FlaskFrom != shift.FlaskFrom {
			return false
		}
		if entry.GoldToX != "" && entry.GoldToX != shift.GoldToX {
			return false
		}
		if entry.GrassToX != "" && entry.GrassToX != shift.GrassToX {
			return false
		}

		if len(entry.From) > 0 {
			var expandedNeedles []string
			for _, f := range entry.From {
				expandedNeedles = append(expandedNeedles, strings.Split(f, ",")...)
			}
			if !includesSome(shift.From, expandedNeedles) {
				return false
			}
		}

		if len(entry.To) > 0 {
			found := false
			for _, t := range entry.To {
				if t == shift.To {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

// --- Perk ---

type PerkRuleVal struct {
	Some [][]string `json:"some"`
	All  [][]string `json:"all"`
	Deck [][]string `json:"deck"`
}

func (c *Checker) checkPerk(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val PerkRuleVal
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}

	if len(val.Deck) > 0 && len(val.Deck[0]) > 0 {
		deck := GetPerkDeck(c.rng)
		if !includesAll(deck, val.Deck[0]) {
			return false
		}
	}

	rows := GetPerks(c.rng)

	for i, somePerks := range val.Some {
		if len(somePerks) == 0 {
			continue
		}
		if i >= len(rows) {
			return false
		}
		if !includesSome(rows[i], somePerks) {
			return false
		}
	}
	for i, allPerks := range val.All {
		if len(allPerks) == 0 {
			continue
		}
		if i >= len(rows) {
			return false
		}
		if !includesAll(rows[i], allPerks) {
			return false
		}
	}
	return true
}

// --- Starting Flask ---

func (c *Checker) checkStartingFlask(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val string
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	return val == GetStartingFlask(c.rng)
}

// --- Starting Spell ---

func (c *Checker) checkStartingSpell(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val string
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	return val == GetStartingSpell(c.rng)
}

// --- Starting Bomb Spell ---

func (c *Checker) checkStartingBombSpell(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val string
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	return val == GetStartingBombSpell(c.rng)
}

// --- Biome Modifier ---

type BiomeModifierRuleVal map[string][]string // biome -> list of acceptable modifier IDs

func (c *Checker) checkBiomeModifier(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val BiomeModifierRuleVal
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	modifiers := GetBiomeModifiers(c.rng)
	for biome, acceptableIDs := range val {
		found := false
		for _, id := range acceptableIDs {
			if modifiers[biome] == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// --- Weather ---

type WeatherRuleVal struct {
	RainMaterial *string    `json:"rain_material"`
	RainType     *int       `json:"rain_type"`
	Fog          *[2]float64 `json:"fog"`
	Clouds       *[2]float64 `json:"clouds"`
}

func (c *Checker) checkWeather(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val WeatherRuleVal
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	weather := GetWeather(c.rng)
	if val.RainMaterial != nil && *val.RainMaterial != weather.RainMaterial {
		return false
	}
	if val.RainType != nil && *val.RainType != weather.RainType {
		return false
	}
	if val.Fog != nil && (weather.Fog < val.Fog[0] || weather.Fog > val.Fog[1]) {
		return false
	}
	if val.Clouds != nil && (weather.Clouds < val.Clouds[0] || weather.Clouds > val.Clouds[1]) {
		return false
	}
	return true
}
