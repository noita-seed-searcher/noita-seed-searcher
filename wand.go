package main

import (
	_ "embed"
	"encoding/json"
	"math"
)

//go:embed data/wands.json
var wandsJSON []byte

// WandTemplate is a wand sprite/stats template from wands.json.
type WandTemplate struct {
	FireRateWait         int `json:"fire_rate_wait"`
	ActionsPerRound      int `json:"actions_per_round"`
	ShuffleDeckWhenEmpty int `json:"shuffle_deck_when_empty"`
	DeckCapacity         int `json:"deck_capacity"`
	SpreadDegrees        int `json:"spread_degrees"`
	ReloadTime           int `json:"reload_time"`
}

var wandTemplates []WandTemplate

func init() {
	if err := json.Unmarshal(wandsJSON, &wandTemplates); err != nil {
		panic("failed to load wands.json: " + err.Error())
	}
}

// Gun holds wand stats, matching the TS Gun class fields.
type Gun struct {
	Cost                 float64
	DeckCapacity         float64
	ActionsPerRound      float64
	ReloadTime           float64
	ShuffleDeckWhenEmpty float64
	FireRateWait         float64
	SpreadDegrees        float64
	SpeedMultiplier      float64
	ManaChargeSpeed      float64
	ManaMax              float64
	ForceUnshuffle       float64
	IsRare               float64
	ProbUnshuffle        float64
	ProbDrawMany         float64
}

// GunCards holds the spells loaded into a wand.
type GunCards struct {
	Cards         []string
	PermanentCard string
}

// WandResult holds a generated wand.
type WandResult struct {
	Gun   Gun
	UI    *WandTemplate
	Cards GunCards
}

type probEntry struct {
	prob, min, max, mean float32
	sharpness            int32
}

var gunProbsDecapacity = []probEntry{
	{1, 3, 10, 6, 2},
	{0.1, 2, 7, 4, 4},
	{0.05, 1, 5, 3, 4},
	{0.15, 5, 11, 8, 2},
	{0.12, 2, 20, 8, 4},
	{0.15, 3, 12, 6, 6},
	{1, 1, 20, 6, 0},
}

var gunProbsReloadTime = []probEntry{
	{1, 5, 60, 30, 2},
	{0.5, 1, 100, 40, 2},
	{0.02, 1, 100, 40, 0},
	{0.35, 1, 240, 40, 0},
}

var gunProbsFireRateWait = []probEntry{
	{1, 1, 30, 5, 2},
	{0.1, 1, 50, 15, 3},
	{0.1, -15, 15, 0, 3},
	{0.45, 0, 35, 12, 0},
}

var gunProbsSpreadDegrees = []probEntry{
	{1, -5, 10, 0, 3},
	{0.1, -35, 35, 0, 0},
}

var gunProbsSpeedMultiplier = []probEntry{
	{1, 0.8, 1.2, 1, 6},
	{0.05, 1, 2, 1.1, 3},
	{0.05, 0.5, 1, 0.9, 3},
	{1, 0.8, 1.2, 1, 0},
	{0.001, 1, 10, 5, 2},
}

var gunProbsActionsPerRound = []probEntry{
	{1, 1, 3, 1, 3},
	{0.2, 2, 4, 2, 8},
	{0.05, 1, 5, 2, 2},
	{1, 1, 5, 2, 0},
}

func totalProb(probs []probEntry) float32 {
	sum := float32(0)
	for _, p := range probs {
		sum += p.prob
	}
	return sum
}

func clamp(val, lower, upper float64) float64 {
	if lower > upper {
		return clamp(val, upper, lower)
	}
	return math.Max(lower, math.Min(upper, val))
}

func (rng *RNG) getGunProbs(probs []probEntry) probEntry {
	r := float32(rng.Randomf()) * totalProb(probs)
	for _, v := range probs {
		if r <= v.prob {
			return v
		}
		r -= v.prob
	}
	return probs[len(probs)-1]
}

func wandShuffleTable(rng *RNG, t []int) {
	for i := len(t) - 1; i >= 1; i-- {
		j := rng.RandomInt(0, int32(i))
		t[i], t[j] = t[j], t[i]
	}
}

// wandGetUI mirrors GetWandUI — includes the Random(0,100) early-return when score==0.
func wandGetUI(rng *RNG, gun *Gun) *WandTemplate {
	firerate := clamp((gun.FireRateWait+5)/7-1, 0, 4)
	apr := clamp(gun.ActionsPerRound-1, 0, 2)
	shuffle := clamp(gun.ShuffleDeckWhenEmpty, 0, 1)
	deck := clamp((gun.DeckCapacity-3)/3, 0, 7)
	spread := clamp((gun.SpreadDegrees+5)/5-1, 0, 2)
	reload := clamp((gun.ReloadTime+5)/25-1, 0, 2)

	bestScore := 1000.0
	var best *WandTemplate
	for i := range wandTemplates {
		w := &wandTemplates[i]
		score := math.Abs(firerate-float64(w.FireRateWait))*2 +
			math.Abs(apr-float64(w.ActionsPerRound))*20 +
			math.Abs(shuffle-float64(w.ShuffleDeckWhenEmpty))*30 +
			math.Abs(deck-float64(w.DeckCapacity))*5 +
			math.Abs(spread-float64(w.SpreadDegrees)) +
			math.Abs(reload-float64(w.ReloadTime))
		if score <= bestScore {
			best = w
			bestScore = score
			if score == 0 && rng.RandomInt(0, 100) < 33 {
				return best
			}
		}
	}
	return best
}

func getGunData(rng *RNG, cost float64, level int, forceUnshuffle bool, unshufflePerk bool) Gun {
	if level == 1 {
		if rng.RandomInt(0, 100) < 50 {
			cost += 5
		}
	}
	cost += float64(rng.RandomInt(-3, 3))

	// Gun constructor — matches TS `new Gun(randoms, level, cost)`
	gun := Gun{
		Cost:                 cost,
		ShuffleDeckWhenEmpty: 1, // TS default is 1
		ProbUnshuffle:        0.1,
		ProbDrawMany:         0.15,
	}
	gun.ManaChargeSpeed = float64(50*level) + float64(rng.RandomInt(-5, int32(5*level)))
	gun.ManaMax = float64(50+150*level) + float64(rng.RandomInt(-5, 5))*10

	p := rng.RandomInt(0, 100)
	if p < 20 {
		gun.ManaChargeSpeed = (float64(50*level) + float64(rng.RandomInt(-5, int32(5*level)))) / 5
		gun.ManaMax = (float64(50+150*level) + float64(rng.RandomInt(-5, 5))*10) * 3
	}

	p = rng.RandomInt(0, 100)
	if p < 15 {
		gun.ManaChargeSpeed = (float64(50*level) + float64(rng.RandomInt(-5, int32(5*level)))) * 5
		gun.ManaMax = (float64(50+150*level) + float64(rng.RandomInt(-5, 5))*10) / 3
	}

	if gun.ManaMax < 50 {
		gun.ManaMax = 50
	}
	if gun.ManaChargeSpeed < 10 {
		gun.ManaChargeSpeed = 10
	}

	p = rng.RandomInt(0, 100)
	if p < int32(15+level*6) {
		gun.ForceUnshuffle = 1
	}

	p = rng.RandomInt(0, 100)
	if p < 5 {
		gun.IsRare = 1
		gun.Cost += 65
	}

	vars01 := []int{0, 1, 2, 3} // reload_time, fire_rate_wait, spread_degrees, speed_multiplier
	vars02 := []int{4}           // deck_capacity
	vars03 := []int{5, 6}        // shuffle_deck_when_empty, actions_per_round

	wandShuffleTable(rng, vars01)
	if gun.ForceUnshuffle != 1 {
		wandShuffleTable(rng, vars03)
	}

	for _, v := range vars01 {
		applyRandomVariable(rng, &gun, v)
	}
	for _, v := range vars02 {
		applyRandomVariable(rng, &gun, v)
	}
	for _, v := range vars03 {
		applyRandomVariable(rng, &gun, v)
	}

	if gun.Cost > 5 && rng.RandomInt(0, 1000) < 995 {
		if gun.ShuffleDeckWhenEmpty == 1 {
			gun.DeckCapacity += gun.Cost / 5
			gun.Cost = 0
		} else {
			gun.DeckCapacity += gun.Cost / 10
			gun.Cost = 0
		}
	}

	if forceUnshuffle || unshufflePerk {
		gun.ShuffleDeckWhenEmpty = 0
	}

	if rng.RandomInt(0, 10000) <= 9999 {
		gun.DeckCapacity = clamp(gun.DeckCapacity, 2, 26)
	}

	if gun.DeckCapacity <= 1 {
		gun.DeckCapacity = 2
	}

	if gun.ReloadTime >= 60 {
		var randomAddAPR func()
		randomAddAPR = func() {
			gun.ActionsPerRound++
			if rng.RandomInt(0, 100) < 70 {
				randomAddAPR()
			}
		}
		randomAddAPR()

		if rng.RandomInt(0, 100) < 50 {
			newAPR := gun.DeckCapacity
			for i := 0; i < 6; i++ {
				tmp := float64(rng.RandomInt(int32(gun.ActionsPerRound), int32(gun.DeckCapacity)))
				if tmp < newAPR {
					newAPR = tmp
				}
			}
			gun.ActionsPerRound = newAPR
		}
	}

	gun.ActionsPerRound = clamp(gun.ActionsPerRound, 1, gun.DeckCapacity)
	return gun
}

// applyRandomVariable applies one variable's gun_probs, mirroring apply_random_variable.
// variable: 0=reload_time, 1=fire_rate_wait, 2=spread_degrees, 3=speed_multiplier,
//
//	4=deck_capacity, 5=shuffle_deck_when_empty, 6=actions_per_round
func applyRandomVariable(rng *RNG, gun *Gun, variable int) {
	cost := gun.Cost

	switch variable {
	case 0: // reload_time
		probs := rng.getGunProbs(gunProbsReloadTime)
		minV := clamp(60-cost*5, 1, 240)
		val := clamp(float64(rng.RandomDistribution(int32(probs.min), int32(probs.max), int32(probs.mean), probs.sharpness)), minV, 1024)
		gun.ReloadTime = val
		gun.Cost -= (60 - val) / 5

	case 1: // fire_rate_wait
		probs := rng.getGunProbs(gunProbsFireRateWait)
		minV := clamp(16-cost, -50, 50)
		val := clamp(float64(rng.RandomDistribution(int32(probs.min), int32(probs.max), int32(probs.mean), probs.sharpness)), minV, 50)
		gun.FireRateWait = val
		gun.Cost -= (16 - val)

	case 2: // spread_degrees
		probs := rng.getGunProbs(gunProbsSpreadDegrees)
		minV := clamp(cost/-1.5, -35, 35)
		val := clamp(float64(rng.RandomDistribution(int32(probs.min), int32(probs.max), int32(probs.mean), probs.sharpness)), minV, 35)
		gun.SpreadDegrees = val
		gun.Cost -= (16 - val)

	case 3: // speed_multiplier
		probs := rng.getGunProbs(gunProbsSpeedMultiplier)
		gun.SpeedMultiplier = float64(rng.RandomDistributionf(probs.min, probs.max, probs.mean, probs.sharpness))

	case 4: // deck_capacity
		probs := rng.getGunProbs(gunProbsDecapacity)
		minV := float64(1)
		maxV := clamp(cost/5+6, 1, 20)
		if gun.ForceUnshuffle == 1 {
			minV = 1
			maxV = (cost - 15) / 5
			if maxV > 6 {
				maxV = 6 + (cost-(15+6*5))/10
			}
		}
		maxV = clamp(maxV, 1, 20)
		val := clamp(float64(rng.RandomDistribution(int32(probs.min), int32(probs.max), int32(probs.mean), probs.sharpness)), minV, maxV)
		gun.DeckCapacity = val
		gun.Cost -= (val - 6) * 5

	case 5: // shuffle_deck_when_empty
		random := rng.RandomInt(0, 1)
		if gun.ForceUnshuffle == 1 {
			random = 1
		}
		deck := gun.DeckCapacity
		if random == 1 && cost >= 15+deck*5 && deck <= 9 {
			gun.ShuffleDeckWhenEmpty = 0
			gun.Cost -= (15 + deck*5)
		}

	case 6: // actions_per_round
		deck := gun.DeckCapacity
		actionCosts := []float64{0, 5 + deck*2, 15 + deck*3.5, 35 + deck*5, 45 + deck*deck}
		minV := float64(1)
		maxV := float64(1)
		for i := 1; i <= len(actionCosts); i++ {
			if actionCosts[i-1] <= cost {
				maxV = float64(i)
			}
		}
		maxV = clamp(maxV, 1, deck)
		probs := rng.getGunProbs(gunProbsActionsPerRound)
		val := math.Floor(clamp(float64(rng.RandomDistribution(int32(probs.min), int32(probs.max), int32(probs.mean), probs.sharpness)), minV, maxV))
		gun.ActionsPerRound = val
		idx := int(clamp(val, 1, float64(len(actionCosts)))) - 1
		gun.Cost -= actionCosts[idx]
	}
}

// wandAddRandomCards mirrors wand_add_random_cards from Wand.ts.
func wandAddRandomCards(rng *RNG, x, y float64, gun *Gun, level int) GunCards {
	res := GunCards{}
	isRare := gun.IsRare

	// good_cards initial set — may be overwritten; Random call still must happen
	if rng.RandomInt(0, 100) < 7 {
		rng.RandomInt(20, 50)
	}

	origLevel := level
	level = level - 1
	deckCapacity := gun.DeckCapacity
	actionsPerRound := gun.ActionsPerRound

	cardCount := float64(rng.RandomInt(1, 3))
	bulletCard := GetRandomActionWithType(rng, x, y, level, ACTION_PROJECTILE, 0)

	if rng.RandomInt(0, 100) < 50 && cardCount < 3 {
		cardCount++
	}
	if rng.RandomInt(0, 100) < 10 || isRare == 1 {
		cardCount += float64(rng.RandomInt(1, 2))
	}
	goodCards := float64(rng.RandomInt(5, 45))
	// Random(0.51*deck, deck) — use RandomRounded to match TS banker's rounding on float args
	cardCount = float64(rng.RandomRounded(0.51*deckCapacity, deckCapacity))
	cardCount = clamp(cardCount, 1, deckCapacity-1)

	randomBullets := 0
	if rng.RandomInt(0, 100) < int32(origLevel*10-5) {
		randomBullets = 1
	}

	goodCardCount := 0
	if rng.RandomInt(0, 100) < 4 || isRare == 1 {
		p := rng.RandomInt(0, 100)
		var card string
		if p < 77 {
			card = GetRandomActionWithType(rng, x, y, level+1, ACTION_MODIFIER, 666)
		} else if p < 85 {
			card = GetRandomActionWithType(rng, x, y, level+1, ACTION_MODIFIER, 666)
			goodCardCount++
		} else if p < 93 {
			card = GetRandomActionWithType(rng, x, y, level+1, ACTION_STATIC_PROJECTILE, 666)
		} else {
			card = GetRandomActionWithType(rng, x, y, level+1, ACTION_PROJECTILE, 666)
		}
		res.PermanentCard = card
	}

	if rng.RandomInt(0, 100) < 50 {
		extraLevel := level
		for rng.RandomInt(1, 10) == 10 {
			extraLevel++
			bulletCard = GetRandomActionWithType(rng, x, y, extraLevel, ACTION_PROJECTILE, 0)
		}
		if cardCount < 3 {
			if cardCount > 1 && rng.RandomInt(0, 100) < 20 {
				card := GetRandomActionWithType(rng, x, y, level, ACTION_MODIFIER, 2)
				res.Cards = append(res.Cards, card)
				cardCount--
			}
			for i := float64(1); i <= cardCount; i++ {
				res.Cards = append(res.Cards, bulletCard)
			}
		} else {
			if rng.RandomInt(0, 100) < 40 {
				card := GetRandomActionWithType(rng, x, y, level, ACTION_DRAW_MANY, 1)
				res.Cards = append(res.Cards, card)
				cardCount--
			}
			if cardCount > 3 && rng.RandomInt(0, 100) < 40 {
				card := GetRandomActionWithType(rng, x, y, level, ACTION_DRAW_MANY, 1)
				res.Cards = append(res.Cards, card)
				cardCount--
			}
			if rng.RandomInt(0, 100) < 80 {
				card := GetRandomActionWithType(rng, x, y, level, ACTION_MODIFIER, 2)
				res.Cards = append(res.Cards, card)
				cardCount--
			}
			for i := float64(1); i <= cardCount; i++ {
				res.Cards = append(res.Cards, bulletCard)
			}
		}
	} else {
		for i := float64(1); i <= cardCount; i++ {
			if rng.RandomInt(0, 100) < int32(goodCards) && cardCount > 2 {
				var card string
				if goodCardCount == 0 && actionsPerRound == 1 {
					card = GetRandomActionWithType(rng, x, y, level, ACTION_DRAW_MANY, int(i))
					goodCardCount++
				} else {
					if rng.RandomInt(0, 100) < 83 {
						card = GetRandomActionWithType(rng, x, y, level, ACTION_MODIFIER, int(i))
					} else {
						card = GetRandomActionWithType(rng, x, y, level, ACTION_DRAW_MANY, int(i))
					}
				}
				res.Cards = append(res.Cards, card)
			} else {
				res.Cards = append(res.Cards, bulletCard)
				if randomBullets == 1 {
					bulletCard = GetRandomActionWithType(rng, x, y, level, ACTION_PROJECTILE, int(i))
				}
			}
		}
	}
	return res
}

// ProvideWand generates a wand at position (x, y) with given cost, level, and flags.
func ProvideWand(rng *RNG, x, y float64, cost float64, level int, forceUnshuffle, unshufflePerk bool) WandResult {
	rng.SetRandomSeed(x, y)
	gun := getGunData(rng, cost, level, forceUnshuffle, unshufflePerk)
	ui := wandGetUI(rng, &gun)
	cards := wandAddRandomCards(rng, x, y, &gun, level)
	return WandResult{Gun: gun, UI: ui, Cards: cards}
}
