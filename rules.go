package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// ruleCosts maps leaf rule types to relative evaluation cost.
// These are calibrated from observed throughput:
// alchemy ~4.8M/s, fungal ~400k/s, perk ~90k/s, wand ~30k/s, shop ~5k/s.
var ruleCosts = map[string]int{
	"alchemy":           1,
	"startingFlask":     1,
	"startingSpell":     1,
	"startingBombSpell": 1,
	"weather":           2,
	"biomeModifier":     3,
	"fungalShift":       10,
	"perk":              50,
	"lottery":           55, // depends on perk
	"wand":              150,
	"shop":              800,
}

func leafCost(ruleType string) int {
	if c, ok := ruleCosts[ruleType]; ok {
		return c
	}
	return 0 // unimplemented rules are free (always true, filtered out immediately)
}

func nodeCost(rule *RuleNode) int {
	switch rule.Type {
	case RuleTypeAND, RuleTypeOR, RuleTypeRules:
		total := 0
		for _, r := range rule.Rules {
			total += nodeCost(r)
		}
		return total
	case RuleTypeNOT:
		if len(rule.Rules) > 0 {
			return nodeCost(rule.Rules[0])
		}
		return 0
	default:
		return leafCost(rule.Type)
	}
}

// sortRulesByCost recursively sorts AND/OR children cheapest-first so that
// short-circuit evaluation skips expensive providers whenever a cheap one fails.
func sortRulesByCost(rule *RuleNode) {
	switch rule.Type {
	case RuleTypeAND, RuleTypeOR, RuleTypeRules:
		sort.SliceStable(rule.Rules, func(i, j int) bool {
			return nodeCost(rule.Rules[i]) < nodeCost(rule.Rules[j])
		})
	}
	for _, r := range rule.Rules {
		sortRulesByCost(r)
	}
}

// Rule types match the TypeScript IRule / ILogicRules.
const (
	RuleTypeAND   = "and"
	RuleTypeOR    = "or"
	RuleTypeNOT   = "not"
	RuleTypeRules = "rules"
)

// RuleNode is a single node in the rule tree.
type RuleNode struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Rules  []*RuleNode     `json:"rules,omitempty"`
	Val    json.RawMessage `json:"val,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Checker evaluates rules against a seed.
type Checker struct {
	rng      *RNG
	perkRows [][]string // cached for current seed; nil = not yet computed
}

func newChecker() *Checker {
	return &Checker{rng: newRNG()}
}

func (c *Checker) SetSeed(seed uint32) {
	c.rng.SetWorldSeed(seed)
	c.perkRows = nil
}

func (c *Checker) getPerks() [][]string {
	if c.perkRows == nil {
		c.perkRows = GetPerks(c.rng)
	}
	return c.perkRows
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
		return c.checkWand(rule)
	case "shop":
		return c.checkShop(rule)
	case "spells":
		return true
	case "lottery":
		return c.checkLottery(rule)
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

	rows := c.getPerks()

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

// --- Wand ---

type WandRuleParams struct {
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Cost          float64 `json:"cost"`
	Level         int     `json:"level"`
	ForceUnshuffle bool   `json:"force_unshuffle"`
	UnshufflePerk bool    `json:"unshufflePerk"`
}

type WandRuleVal struct {
	Gun          map[string][2]float64 `json:"gun"`
	Cards        []string              `json:"cards"`
	CardsStrict  bool                  `json:"cardsStrict"`
	PermanentCard json.RawMessage      `json:"permanentCard"`
}

func (c *Checker) testGun(target WandRuleVal, wand WandResult) bool {
	gun := wand.Gun
	for stat, rng := range target.Gun {
		var val float64
		switch stat {
		case "cost":
			val = gun.Cost
		case "deck_capacity":
			val = gun.DeckCapacity
		case "actions_per_round":
			val = gun.ActionsPerRound
		case "reload_time":
			val = gun.ReloadTime
		case "shuffle_deck_when_empty":
			val = gun.ShuffleDeckWhenEmpty
		case "fire_rate_wait":
			val = gun.FireRateWait
		case "spread_degrees":
			val = gun.SpreadDegrees
		case "speed_multiplier":
			val = gun.SpeedMultiplier
		case "mana_charge_speed":
			val = gun.ManaChargeSpeed
		case "mana_max":
			val = gun.ManaMax
		case "force_unshuffle":
			val = gun.ForceUnshuffle
		case "prob_unshuffle":
			val = gun.ProbUnshuffle
		case "prob_draw_many":
			val = gun.ProbDrawMany
		case "is_rare":
			val = gun.IsRare
		default:
			continue
		}
		if val < rng[0] || val > rng[1] {
			return false
		}
	}

	if len(target.Cards) > 0 {
		if target.CardsStrict {
			if !includesAll(wand.Cards.Cards, target.Cards) {
				return false
			}
		} else {
			if !includesSome(wand.Cards.Cards, target.Cards) {
				return false
			}
		}
	}

	// permanentCard: null → must have no permanent card
	//                true → must have any permanent card
	//                ["X","Y"] → permanent card must be one of those
	if len(target.PermanentCard) > 0 {
		raw := string(target.PermanentCard)
		if raw == "null" {
			if wand.Cards.PermanentCard != "" {
				return false
			}
		} else if raw == "true" {
			if wand.Cards.PermanentCard == "" {
				return false
			}
		} else {
			var options []string
			if err := json.Unmarshal(target.PermanentCard, &options); err == nil {
				if wand.Cards.PermanentCard == "" {
					return false
				}
				found := false
				for _, opt := range options {
					if opt == wand.Cards.PermanentCard {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		}
	}

	return true
}

func (c *Checker) checkWand(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var params WandRuleParams
	if len(rule.Params) > 0 {
		if err := json.Unmarshal(rule.Params, &params); err != nil {
			return true
		}
	}
	var val WandRuleVal
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	wand := ProvideWand(c.rng, params.X, params.Y, params.Cost, params.Level, params.ForceUnshuffle, params.UnshufflePerk)
	return c.testGun(val, wand)
}

// --- Shop ---

type ShopRuleItem struct {
	Spell string          `json:"spell"`
	Wand  *WandRuleVal    `json:"wand"`
}

type ShopLevelRule struct {
	Type   ShopType       `json:"type"`
	Items  []ShopRuleItem `json:"items"`
	Strict bool           `json:"strict"`
}

func (c *Checker) checkShop(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	// rule.val is a sparse array indexed by temple level (0-6)
	var shopRules []*ShopLevelRule
	if err := json.Unmarshal(rule.Val, &shopRules); err != nil {
		return true
	}

	for j, shopRule := range shopRules {
		if shopRule == nil {
			continue
		}
		if shopRule.Type == 0 {
			continue
		}

		info := ProvideShopLevel(c.rng, j, false)
		if shopRule.Type != info.Type {
			return false
		}

		if len(shopRule.Items) == 0 {
			return true
		}

		if info.Type == ShopTypeWand {
			matched := false
			for _, sw := range info.Wands {
				if shopRule.Items[0].Wand != nil && c.testGun(*shopRule.Items[0].Wand, sw.Wand) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		} else {
			needles := make([]string, 0, len(shopRule.Items))
			for _, item := range shopRule.Items {
				if item.Spell != "" {
					needles = append(needles, strings.ToUpper(item.Spell))
				}
			}
			haystack := make([]string, 0, len(info.Items))
			for _, item := range info.Items {
				haystack = append(haystack, strings.ToUpper(item.SpellID))
			}
			if shopRule.Strict {
				if !includesAll(haystack, needles) {
					return false
				}
			} else {
				if !includesSome(haystack, needles) {
					return false
				}
			}
		}
	}
	return true
}

// --- Lottery ---

// lotteryIsRerolledFn is the standalone implementation — returns true if perk IS rerolled.
func lotteryIsRerolledFn(rng *RNG, level, perkNumber, perksOnLevel, lotteries int) bool {
	if level >= len(templeLocations) {
		return false
	}
	temple := templeLocations[level]
	perkY := temple.Y
	rawX := temple.X + (float64(perkNumber)+0.5)*(60.0/float64(perksOnLevel))
	perkX := float64(roundHalfToEvenI32(rawX))
	probability := 100.0 * math.Pow(0.5, float64(lotteries))
	rng.SetRandomSeed(perkX, perkY)
	return float64(rng.RandomInt(1, 100)) <= probability
}

func (c *Checker) lotteryIsRerolled(level, perkNumber, perksOnLevel, lotteries int) bool {
	return lotteryIsRerolledFn(c.rng, level, perkNumber, perksOnLevel, lotteries)
}

// LotteryRuleVal is the val for the "lottery" rule type.
// Perks: which perks to check.
// Lotteries: how many PERKS_LOTTERY were picked (default 1).
// Count: how many of the listed perks must NOT be rerolled (default 1 = at least one).
type LotteryRuleVal struct {
	Perks     []string `json:"perks"`
	Lotteries int      `json:"lotteries"`
	Count     int      `json:"count"`
}

func (c *Checker) checkLottery(rule *RuleNode) bool {
	if len(rule.Val) == 0 {
		return true
	}
	var val LotteryRuleVal
	if err := json.Unmarshal(rule.Val, &val); err != nil {
		return true
	}
	if len(val.Perks) == 0 {
		return true
	}
	lotteries := val.Lotteries
	if lotteries == 0 {
		lotteries = 1
	}
	minCount := val.Count
	if minCount == 0 {
		minCount = 1
	}

	rows := c.getPerks()
	const perksOnLevel = 3

	notRerolled := 0
	for _, needle := range val.Perks {
		// Find which row (level 1+) this perk appears in and at what position.
		for level := 1; level < len(rows); level++ {
			for perkNum, perk := range rows[level] {
				if perk == needle {
					if !c.lotteryIsRerolled(level, perkNum, perksOnLevel, lotteries) {
						notRerolled++
					}
					goto nextPerk
				}
			}
		}
	nextPerk:
	}
	return notRerolled >= minCount
}
