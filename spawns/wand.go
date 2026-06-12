package main

import (
	"fmt"
	"math"
)

// Wand holds all generated properties of a wand.
type Wand struct {
	X, Y                   float64
	Cards                  []string
	AlwaysCasts            []string
	Level                  int
	Cost                   float64
	DeckCapacity           float64
	ActionsPerRound        float64
	ReloadTime             float64
	ShuffleDeckWhenEmpty   int
	FireRateWait           float64
	SpreadDegrees          float64
	SpeedMultiplier        float64
	ManaChargeSpeed        float64
	ManaMax                float64
	ForceUnshuffle         int
	IsRare                 int
	WandType               string // "normal" or "better"
	CardCount              int
	OriginalForceUnshuffle int
	Name                   string
	Sprite                 string
	GripX, GripY           int
	TipX, TipY             int
}

// generateGun mirrors generateGun() from gun_generation.js.
func generateGun(worldSeed uint32, ngPlusCount int, wandType string, cost float64, level int, forceUnshuffle bool, x, y float64, noMoreShuffle bool) *Wand {
	gun := &Wand{
		X:                    x,
		Y:                    y,
		Cards:                []string{},
		AlwaysCasts:          []string{},
		Level:                level,
		Cost:                 cost,
		ShuffleDeckWhenEmpty: 1,
		WandType:             wandType,
	}

	p := newPrng(0)
	p.setRandomSeed(worldSeed+uint32(ngPlusCount), x, y)

	if level == 1 {
		if p.random(0, 100) < 50 {
			gun.Cost += 5
		}
	}
	gun.Cost += float64(p.random(-3, 3))

	gun.ManaChargeSpeed = float64(50*level) + float64(p.random(-5, 5*level))
	gun.ManaMax = float64(50+150*level) + float64(p.random(-5, 5))*10

	rnd := p.random(0, 100)
	if rnd < 20 {
		gun.ManaChargeSpeed = (float64(50*level) + float64(p.random(-5, 5*level))) / 5
		gun.ManaMax = (float64(50+150*level) + float64(p.random(-5, 5))*10) * 3
		if wandType == "better" && gun.ManaChargeSpeed < 10 {
			gun.ManaChargeSpeed = 10
		}
	}

	rnd = p.random(0, 100)
	if wandType == "better" {
		if rnd < 15+level*6 {
			gun.ForceUnshuffle = 1
		}
	} else {
		if rnd < 15 {
			gun.ManaChargeSpeed = (float64(50*level) + float64(p.random(-5, 5*level))) * 5
			gun.ManaMax = (float64(50+150*level) + float64(p.random(-5, 5))*10) / 3
		}
		if gun.ManaMax < 50 {
			gun.ManaMax = 50
		}
		if gun.ManaChargeSpeed < 10 {
			gun.ManaChargeSpeed = 10
		}
		rnd = p.random(0, 100)
		if rnd < 15+level*6 {
			gun.ForceUnshuffle = 1
		}
	}

	rnd = p.random(0, 100)
	if rnd < 5 {
		gun.IsRare = 1
		gun.Cost += 65
	}

	vars1 := []string{"reload_time", "fire_rate_wait", "spread_degrees", "speed_multiplier"}
	vars2 := []string{"deck_capacity"}
	vars3 := []string{"shuffle_deck_when_empty", "actions_per_round"}

	shuffleTable(vars1, p)
	if gun.ForceUnshuffle != 1 {
		shuffleTable(vars3, p)
	}

	for _, v := range vars1 {
		applyRandomVariable(gun, v, p)
	}
	for _, v := range vars2 {
		applyRandomVariable(gun, v, p)
	}
	for _, v := range vars3 {
		applyRandomVariable(gun, v, p)
	}

	if gun.Cost > 5 {
		rareNonincreaseRoll := p.random(0, 1000)
		if rareNonincreaseRoll < 995 {
			if gun.ShuffleDeckWhenEmpty == 1 {
				gun.DeckCapacity += gun.Cost / 5
			} else {
				gun.DeckCapacity += gun.Cost / 10
			}
			gun.Cost = 0
		}
	}

	if wandType == "better" {
		idx := p.random(0, len(gunNames)-1)
		gun.Name = gunNames[idx] + " Wand"
	} else {
		gun.Name = "Wand"
	}

	if forceUnshuffle || noMoreShuffle {
		gun.ShuffleDeckWhenEmpty = 0
	}
	if forceUnshuffle {
		gun.OriginalForceUnshuffle = 1
	}

	rareCapacityRoll := p.random(0, 10000)
	if rareCapacityRoll <= 9999 {
		gun.DeckCapacity = clamp(gun.DeckCapacity, 2, 26)
	}
	if gun.DeckCapacity <= 1 {
		gun.DeckCapacity = 2
	}

	if gun.ReloadTime >= 60 {
		gun.ActionsPerRound += 1
		for p.random(0, 100) < 70 {
			gun.ActionsPerRound += 1
		}
		if p.random(0, 100) < 50 {
			newAPR := gun.DeckCapacity
			for i := 1; i <= 6; i++ {
				tmp := float64(p.random(roundHalfToEven(gun.ActionsPerRound), roundHalfToEven(gun.DeckCapacity)))
				if tmp < newAPR {
					newAPR = tmp
				}
			}
			gun.ActionsPerRound = newAPR
		}
	}

	gun.ActionsPerRound = clamp(gun.ActionsPerRound, 1, gun.DeckCapacity)

	if wandType == "better" {
		betterAddRandomCards(worldSeed, ngPlusCount, gun, x, y, level, p)
		getWandSprite(gun, p)
	} else {
		getWandSprite(gun, p)
		addRandomCards(worldSeed, ngPlusCount, gun, x, y, level, p)
	}

	return gun
}

func applyRandomVariable(gun *Wand, variable string, p *NollaPrng) {
	probs := getGunProbs(gun.WandType, variable, p)
	if probs == nil && variable != "shuffle_deck_when_empty" {
		return
	}

	switch variable {
	case "reload_time":
		minV := clamp(60-gun.Cost*5, 1, 240)
		gun.ReloadTime = clamp(float64(p.randomDistribution(int(probs.min), int(probs.max), int(probs.mean), probs.sharpness)), minV, 1024)
		gun.Cost -= (60 - gun.ReloadTime) / 5

	case "fire_rate_wait":
		minV := clamp(16-gun.Cost, -50, 50)
		gun.FireRateWait = clamp(float64(p.randomDistribution(int(probs.min), int(probs.max), int(probs.mean), probs.sharpness)), minV, 50)
		gun.Cost -= 16 - gun.FireRateWait

	case "spread_degrees":
		minV := clamp(gun.Cost/-1.5, -35, 35)
		gun.SpreadDegrees = clamp(float64(p.randomDistribution(int(probs.min), int(probs.max), int(probs.mean), probs.sharpness)), minV, 35)
		gun.Cost -= 16 - gun.SpreadDegrees

	case "speed_multiplier":
		gun.SpeedMultiplier = p.randomDistributionF(probs.min, probs.max, probs.mean, probs.sharpness)

	case "deck_capacity":
		var minV, maxV float64
		if gun.ForceUnshuffle == 1 {
			minV = 1
			maxV = (gun.Cost - 15) / 5
			if maxV > 6 {
				maxV = 6 + (gun.Cost-(15+6*5))/10
			}
		} else {
			minV = 1
			maxV = clamp(gun.Cost/5+6, 1, 20)
		}
		maxV = clamp(maxV, 1, 20)
		gun.DeckCapacity = clamp(float64(p.randomDistribution(int(probs.min), int(probs.max), int(probs.mean), probs.sharpness)), minV, maxV)
		gun.Cost -= (gun.DeckCapacity - 6) * 5

	case "shuffle_deck_when_empty":
		r := p.random(0, 1)
		if gun.ForceUnshuffle == 1 {
			r = 1
		}
		if r == 1 && gun.Cost >= 15+gun.DeckCapacity*5 && gun.DeckCapacity <= 9 {
			gun.ShuffleDeckWhenEmpty = 0
			gun.Cost -= 15 + gun.DeckCapacity*5
		}

	case "actions_per_round":
		actionCosts := [5]float64{0, 5 + gun.DeckCapacity*2, 15 + gun.DeckCapacity*3.5, 35 + gun.DeckCapacity*5, 45 + gun.DeckCapacity*gun.DeckCapacity}
		maxV := 1.0
		for i := 0; i < 5; i++ {
			if actionCosts[i] <= gun.Cost {
				maxV = float64(i + 1)
			}
		}
		maxV = clamp(maxV, 1, gun.DeckCapacity)
		gun.ActionsPerRound = math.Floor(clamp(float64(p.randomDistribution(int(probs.min), int(probs.max), int(probs.mean), probs.sharpness)), 1, maxV))
		idx := int(clamp(gun.ActionsPerRound, 1, 5)) - 1
		gun.Cost -= actionCosts[idx]
	}
}

func wandDiff(gun *Wand, wand [11]int) float64 {
	// Map gun stats into wand space first (mirrors getWandSprite's gunInWandSpace)
	frw := clamp((gun.FireRateWait+5)/7-1, 0, 4)
	apr := clamp(gun.ActionsPerRound-1, 0, 2)
	sde := clamp(float64(gun.ShuffleDeckWhenEmpty), 0, 1)
	dc := clamp((gun.DeckCapacity-3)/3, 0, 7)
	sd := clamp((gun.SpreadDegrees+5)/5-1, 0, 2)
	rt := clamp((gun.ReloadTime+5)/25-1, 0, 2)

	score := math.Abs(frw-float64(wand[5]))*2 +
		math.Abs(apr-float64(wand[6]))*20 +
		math.Abs(sde-float64(wand[7]))*30 +
		math.Abs(dc-float64(wand[8]))*5 +
		math.Abs(sd-float64(wand[9])) +
		math.Abs(rt-float64(wand[10]))
	return score
}

func getWandSprite(gun *Wand, p *NollaPrng) {
	bestWand := wandShapes[0]
	bestScore := 1000.0

	for _, shape := range wandShapes {
		score := wandDiff(gun, shape)
		if score <= bestScore {
			bestWand = shape
			bestScore = score
			if score == 0 {
				if p.random(0, 100) < 33 {
					break
				}
			}
		}
	}

	gun.GripX = bestWand[1]
	gun.GripY = bestWand[2]
	gun.TipX = bestWand[3]
	gun.TipY = bestWand[4]
	gun.Sprite = fmt.Sprintf("wand_%04d", bestWand[0])
}

func betterAddRandomCards(worldSeed uint32, ngPlusCount int, gun *Wand, x, y float64, level int, p *NollaPrng) {
	if p.random(0, 100) < 7 {
		p.random(20, 50) // consume good_cards roll
	}

	origLevel := level
	level -= 1
	deckCapacity := gun.DeckCapacity
	actionsPerRound := gun.ActionsPerRound

	cardCount := p.random(1, 3)
	bulletCard := GetRandomActionWithType(x, y, level, PROJECTILE, worldSeed, 0)
	goodCardCount := 0

	if p.random(0, 100) < 50 && cardCount < 3 {
		cardCount++
	}
	if p.random(0, 100) < 10 || gun.IsRare == 1 {
		cardCount += p.random(1, 2)
	}

	_ = p.random(5, 45) // good_cards re-roll
	cardCount = p.random(roundHalfToEven(0.51*deckCapacity), roundHalfToEven(deckCapacity))
	cardCount = int(clamp(float64(cardCount), 1, deckCapacity-1))
	gun.CardCount = cardCount

	if p.random(0, 100) < origLevel*10-5 {
		// random_bullets = 1 (tracked below)
	}

	if p.random(0, 100) < 4 || gun.IsRare == 1 {
		pRoll := p.random(0, 100)
		var card string
		if pRoll < 77 {
			card = GetRandomActionWithType(x, y, level+1, MODIFIER, worldSeed, 666)
		} else if pRoll < 85 {
			card = GetRandomActionWithType(x, y, level+1, MODIFIER, worldSeed, 666)
			goodCardCount++
		} else if pRoll < 93 {
			card = GetRandomActionWithType(x, y, level+1, STATIC_PROJECTILE, worldSeed, 666)
		} else {
			card = GetRandomActionWithType(x, y, level+1, PROJECTILE, worldSeed, 666)
		}
		gun.AlwaysCasts = []string{card}
	}
	_ = goodCardCount
	_ = actionsPerRound

	if cardCount < 3 {
		if cardCount > 1 && p.random(0, 100) < 20 {
			card := GetRandomActionWithType(x, y, level, MODIFIER, worldSeed, 2)
			gun.Cards = append(gun.Cards, card)
			cardCount--
		}
		for i := 0; i < cardCount; i++ {
			gun.Cards = append(gun.Cards, bulletCard)
		}
	} else {
		if p.random(0, 100) < 40 {
			card := GetRandomActionWithType(x, y, level, DRAW_MANY, worldSeed, 1)
			gun.Cards = append(gun.Cards, card)
			cardCount--
		}
		if cardCount > 3 && p.random(0, 100) < 40 {
			card := GetRandomActionWithType(x, y, level, DRAW_MANY, worldSeed, 1)
			gun.Cards = append(gun.Cards, card)
			cardCount--
		}
		if p.random(0, 100) < 80 {
			card := GetRandomActionWithType(x, y, level, MODIFIER, worldSeed, 2)
			gun.Cards = append(gun.Cards, card)
			cardCount--
		}
		for i := 0; i < cardCount; i++ {
			gun.Cards = append(gun.Cards, bulletCard)
		}
	}
}

func addRandomCards(worldSeed uint32, ngPlusCount int, gun *Wand, x, y float64, level int, p *NollaPrng) {
	goodCards := 5
	if p.random(0, 100) < 7 {
		goodCards = p.random(20, 50)
	}

	origLevel := level
	level -= 1
	deckCapacity := gun.DeckCapacity
	actionsPerRound := gun.ActionsPerRound

	cardCount := p.random(1, 3)
	bulletCard := GetRandomActionWithType(x, y, level, PROJECTILE, worldSeed, 0)
	goodCardCount := 0
	randomBullets := 0

	if p.random(0, 100) < 50 && cardCount < 3 {
		cardCount++
	}
	if p.random(0, 100) < 10 || gun.IsRare == 1 {
		cardCount += p.random(1, 2)
	}

	goodCards = p.random(5, 45)
	cardCountRaw := p.random(roundHalfToEven(0.51*deckCapacity), roundHalfToEven(deckCapacity))
	cardCountF := clamp(float64(cardCountRaw), 1, deckCapacity-1)
	cardCount = int(cardCountF)
	gun.CardCount = cardCount

	if p.random(0, 100) < origLevel*10-5 {
		randomBullets = 1
	}

	if p.random(0, 100) < 4 || gun.IsRare == 1 {
		pRoll := p.random(0, 100)
		var card string
		if pRoll < 77 {
			card = GetRandomActionWithType(x, y, level+1, MODIFIER, worldSeed, 666)
		} else if pRoll < 85 {
			card = GetRandomActionWithType(x, y, level+1, MODIFIER, worldSeed, 666)
			goodCardCount++
		} else if pRoll < 93 {
			card = GetRandomActionWithType(x, y, level+1, STATIC_PROJECTILE, worldSeed, 666)
		} else {
			card = GetRandomActionWithType(x, y, level+1, PROJECTILE, worldSeed, 666)
		}
		gun.AlwaysCasts = []string{card}
	}
	_ = goodCardCount

	if p.random(0, 100) < 50 {
		extraLevel := level
		for p.random(1, 10) == 10 {
			extraLevel++
			bulletCard = GetRandomActionWithType(x, y, extraLevel, PROJECTILE, worldSeed, 0)
		}
		if cardCount < 3 {
			if cardCountF > 1 && p.random(0, 100) < 20 {
				card := GetRandomActionWithType(x, y, level, MODIFIER, worldSeed, 2)
				gun.Cards = append(gun.Cards, card)
				cardCount--
			}
			for i := 0; i < cardCount; i++ {
				gun.Cards = append(gun.Cards, bulletCard)
			}
		} else {
			fcnt := cardCountF // float shadow matches JS card_count for > comparisons
			if p.random(0, 100) < 40 {
				card := GetRandomActionWithType(x, y, level, DRAW_MANY, worldSeed, 1)
				gun.Cards = append(gun.Cards, card)
				cardCount--
				fcnt--
			}
			if fcnt > 3 && p.random(0, 100) < 40 {
				card := GetRandomActionWithType(x, y, level, DRAW_MANY, worldSeed, 1)
				gun.Cards = append(gun.Cards, card)
				cardCount--
				fcnt--
			}
			if p.random(0, 100) < 80 {
				card := GetRandomActionWithType(x, y, level, MODIFIER, worldSeed, 2)
				gun.Cards = append(gun.Cards, card)
				cardCount--
				fcnt--
			}
			for i := 0; i < cardCount; i++ {
				gun.Cards = append(gun.Cards, bulletCard)
			}
		}
	} else {
		for i := 0; i < cardCount; i++ {
			r := p.random(0, 100)
			if r < goodCards && cardCountF > 2 {
				var card string
				if goodCardCount == 0 && actionsPerRound == 1 {
					card = GetRandomActionWithType(x, y, level, DRAW_MANY, worldSeed, i+1)
					goodCardCount++
				} else {
					if p.random(0, 100) < 83 {
						card = GetRandomActionWithType(x, y, level, MODIFIER, worldSeed, i+1)
					} else {
						card = GetRandomActionWithType(x, y, level, DRAW_MANY, worldSeed, i+1)
					}
				}
				gun.Cards = append(gun.Cards, card)
			} else {
				gun.Cards = append(gun.Cards, bulletCard)
				if randomBullets == 1 {
					bulletCard = GetRandomActionWithType(x, y, level, PROJECTILE, worldSeed, i+1)
				}
			}
		}
	}
}

// GenerateWand is the entry point: looks up wandType by name and calls generateGun.
func GenerateWand(worldSeed uint32, ngPlusCount int, wandTypeName string, x, y float64) *Wand {
	td, ok := wandTypes[wandTypeName]
	if !ok {
		return nil
	}
	return generateGun(worldSeed, ngPlusCount, td.wandType, td.cost, td.level, td.forceUnshuffle, x, y, false)
}

// premadeWand is the static template for a premade wand.
type premadeWand struct {
	name               string
	actionsPerRound    int
	deckCapacity       int
	reloadTime         int
	shuffleDeckWhenEmpty int
	fireRateWait       int
	spreadDegrees      int
	manaMax            float64
	manaChargeSpeed    float64
	speedMultiplier    float64
	sprite             string
	gripX, gripY       int
	tipX, tipY         int
}

var premadeWands = []premadeWand{
	{"Good Rapid bolt",     1, 7,  27, 1,  2,  8,  220, 45, 1.03398,  "wand_0484", 2,2, 10,2},
	{"Good Rapid bolt",     1, 5,  25, 1,  2,  2,  250, 53, 0.982264, "wand_0654", 1,2, 10,2},
	{"Neutral Rapid bolt",  1, 6,  90, 1,  2, -3,  180, 54, 0.994967, "wand_0958", 2,3, 10,3},
	{"Antique Rapid bolt",  1, 3,  16, 0, 23, -1,  190, 51, 0.934037, "wand_0516", 1,5,  8,5},
	{"Slim Rapid bolt",     1, 5,   4, 1,  6,  6,  210, 52, 0.824541, "wand_0484", 2,2, 10,2},
	{"Superior Rapid bolt", 1, 6,  23, 0, 13,  3,  250, 49, 0.985946, "wand_0058", 2,3, 10,3},
	{"Shiny Rapid bolt",    1, 3,  40, 0, 26,  1,  170, 55, 0.955205, "wand_0120", 1,8,  8,8},
	{"Solid Rapid bolt",    1, 7,  26, 1, 10,  2,  160, 52, 0.987743, "wand_0898", 2,2, 10,2},
	{"Turbo Rapid bolt",    1, 6,  49, 1,  3,  3,  160, 46, 0.853983, "wand_0724", 1,1, 10,1},
	{"Large Rapid bolt",    1, 3,  26, 0,  3, -1,  210, 46, 1.05598,  "wand_0326", 1,3,  8,3},
	{"Vanilla Rapid bolt",  1, 2,  26, 0, 20, -2,  250, 50, 1.02408,  "wand_0374", 1,3,  8,3},
	{"Bad Rapid bolt",      1, 6,  50, 0, 14, -2,  250, 49, 0.995504, "wand_0324", 1,4, 10,4},
	{"Large Rapid bolt",    1, 5,  45, 0,  4,  1,  220, 51, 0.969829, "wand_0309", 1,3,  9,3},
	{"Slick Rapid bolt",    1, 2,  12, 0,  6,  1,  210, 50, 0.988051, "wand_0230", 1,3,  8,3},
	{"Bulky Rapid bolt",    1, 11, 53, 1, 24,  1,  170, 49, 0.904531, "wand_0063", 1,5, 14,5},
	{"Giga Rapid bolt",     1, 9,  18, 1, 26, -3,  750, 10, 1.07678,  "wand_0830", 2,5, 12,5},
	{"Type a Rapid bolt",   1, 6,  14, 1,  2,  5,  190, 52, 0.987879, "wand_1000", 2,2, 10,2},
}

// generateLevel1Wand mirrors generateLevel1Wand() from wand_generation.js.
// type is "premade_N" (1-indexed).
func generateLevel1Wand(ws uint32, ng int, x, y float64, wandTypeName string) *Wand {
	// Parse index from "premade_N"
	idx := 0
	for i := len("premade_"); i < len(wandTypeName); i++ {
		idx = idx*10 + int(wandTypeName[i]-'0')
	}
	idx = (idx - 1) % len(premadeWands)
	ref := premadeWands[idx]

	gun := &Wand{
		Name:               ref.name,
		ActionsPerRound:    float64(ref.actionsPerRound),
		DeckCapacity:       float64(ref.deckCapacity),
		ReloadTime:         float64(ref.reloadTime),
		ShuffleDeckWhenEmpty: ref.shuffleDeckWhenEmpty,
		FireRateWait:       float64(ref.fireRateWait),
		SpreadDegrees:      float64(ref.spreadDegrees),
		ManaMax:            ref.manaMax,
		ManaChargeSpeed:    ref.manaChargeSpeed,
		SpeedMultiplier:    ref.speedMultiplier,
		Sprite:             ref.sprite,
		GripX:              ref.gripX, GripY: ref.gripY,
		TipX:               ref.tipX, TipY: ref.tipY,
		Cards:              []string{},
		AlwaysCasts:        []string{},
		WandType:           wandTypeName,
		X: x, Y: y,
	}

	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x, y)

	total := ref.reloadTime + ref.fireRateWait + ref.spreadDegrees
	total += p.random(-10, 20)

	level1Cards := []string{
		"LIGHT_BULLET", "RUBBER_BALL", "ARROW", "DISC_BULLET",
		"BOUNCY_ORB", "BULLET", "AIR_BULLET", "SLIMEBALL",
	}

	cardCount := p.random(1, 5)

	if p.random(1, 100) <= 85 {
		level1Cards = append(level1Cards, "BUBBLESHOT")
		if p.random(1, 100) <= 70 {
			level1Cards = append(level1Cards, "SPITTER")
			if p.random(1, 100) <= 40 {
				level1Cards = append(level1Cards, "LIGHT_BULLET_TRIGGER")
				cardCount = 1
				if p.random(1, 100) <= 20 {
					level1Cards = append(level1Cards, "DISC_BULLET_BIG")
					if p.random(1, 100) <= 10 {
						level1Cards = append(level1Cards, "TENTACLE_PORTAL")
						if p.random(1, 100) <= 10 {
							level1Cards = append(level1Cards, "BLACK_HOLE_BIG")
						}
					}
				}
			}
		}
	}

	if total > 50 {
		level1Cards = []string{"GRENADE", "BOMB", "ROCKET"}
		if p.random(1, 100) <= 75 {
			level1Cards = append(level1Cards, "DYNAMITE")
			if p.random(1, 100) <= 50 {
				level1Cards = append(level1Cards, "FIREBALL")
				if p.random(1, 100) <= 40 {
					level1Cards = append(level1Cards, "ACIDSHOT")
					if p.random(1, 100) <= 30 {
						level1Cards = append(level1Cards, "GLITTER_BOMB")
						if p.random(1, 100) <= 30 {
							level1Cards = append(level1Cards, "MINE")
						}
					}
				}
			}
		}
		cardCount = 1
	}

	doUtil := p.random(0, 100)
	if doUtil < 30 {
		level1Cards = []string{
			"CLOUD_WATER", "X_RAY", "FREEZE_FIELD", "BLACK_HOLE", "TORCH", "SHIELD_FIELD",
		}
		if p.random(1, 100) <= 75 {
			level1Cards = append(level1Cards, "ELECTROCUTION_FIELD")
			if p.random(1, 100) <= 50 {
				level1Cards = append(level1Cards, "DIGGER")
				if p.random(1, 100) <= 50 {
					level1Cards = append(level1Cards, "TORCH_ELECTRIC")
					if p.random(1, 100) <= 50 {
						level1Cards = append(level1Cards, "POWERDIGGER")
						if p.random(1, 100) <= 50 {
							level1Cards = append(level1Cards, "SOILBALL")
							if p.random(1, 100) <= 50 {
								level1Cards = append(level1Cards, "LUMINOUS_DRILL")
								if p.random(1, 100) <= 50 {
									level1Cards = append(level1Cards, "CHAINSAW")
								}
							}
						}
					}
				}
			}
		}
		cardCount = 1
	}

	if cardCount > ref.deckCapacity {
		cardCount = ref.deckCapacity
	}

	cardIdx := p.random(0, len(level1Cards)-1)
	card := level1Cards[cardIdx]

	for i := 0; i < cardCount; i++ {
		gun.Cards = append(gun.Cards, card)
	}

	gun.CardCount = cardCount
	return gun
}
