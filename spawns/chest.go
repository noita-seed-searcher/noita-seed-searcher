package main

import "math"

// ChestResult holds the output of a chest generation.
type ChestResult struct {
	Type  string // "chest" or "great_chest"
	Items []*Item
	X, Y  float64
}

// createWandItem mirrors createWand() from wand_generation.js.
// addOffset: if true, adds 510,683 to coords before generating.
func createWandItem(ws uint32, ng int, x, y float64, wandTypeName string, addOffset, noMoreShuffle bool) *Item {
	wx := math.Floor(x)
	wy := math.Floor(y)
	if addOffset {
		wx += 510
		wy += 683
	}
	td, ok := wandTypes[wandTypeName]
	if !ok {
		return nil
	}
	wand := generateGun(ws, ng, td.wandType, td.cost, td.level, td.forceUnshuffle, wx, wy, noMoreShuffle)
	wand.X = wx
	wand.Y = wy
	return &Item{ItemType: "wand", Wand: wand, X: wx, Y: wy}
}

func deduplicateItems(items []*Item) []*Item {
	var out []*Item
	for _, item := range items {
		key := item.Key()
		found := false
		for _, d := range out {
			if d.Key() == key {
				d.Count++
				if d.Amount > 0 {
					d.Amount += item.Amount
				}
				found = true
				break
			}
		}
		if !found {
			cp := *item
			if cp.Count == 0 {
				cp.Count = 1
			}
			out = append(out, &cp)
		}
	}
	return out
}

// SpawnChest mirrors spawnChest() from chest_generation.js.
func SpawnChest(ws uint32, ng int, x, y float64, isTower, greedCurse, noMoreShuffle bool) *ChestResult {
	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x, y)

	greatChestRate := 2000
	if greedCurse || isTower {
		greatChestRate = 100
	}
	rnd := p.random(1, greatChestRate)
	if rnd >= greatChestRate-1 {
		return GenerateGreatChest(ws, ng, x, y, noMoreShuffle)
	}
	return GenerateChest(ws, ng, x, y, noMoreShuffle, greedCurse)
}

// GenerateGreatChest mirrors generateGreatChest() from chest_generation.js.
func GenerateGreatChest(ws uint32, ng int, x, y float64, noMoreShuffle bool) *ChestResult {
	x = roundRNGPos(x)
	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x, y)

	var items []*Item
	count := 1

	// Very special (~impossible)
	if p.random(0, 100000) >= 100000 {
		count = 0
		if p.random(0, 1000) == 999 {
			items = append(items, &Item{ItemType: "true_orb", X: x, Y: y})
		} else {
			items = append(items, &Item{ItemType: "sampo", X: x, Y: y})
		}
	}

	for count > 0 {
		count--
		rnd := p.random(1, 100)
		switch {
		case rnd <= 10:
			rnd2 := p.random(0, 100)
			if rnd2 <= 30 {
				pot := createPotion(ws, ng, x, y, "normal", "normal")
				items = append(items, pot, pot)
				items = append(items, createPotion(ws, ng, x, y, "secret", "normal"))
			} else {
				pot := createPotion(ws, ng, x, y, "secret", "normal")
				items = append(items, pot, pot)
				items = append(items, createPotion(ws, ng, x, y, "random", "normal"))
			}
		case rnd <= 15:
			items = append(items, &Item{ItemType: "gold", Amount: 1000, X: x, Y: y})
		case rnd <= 18:
			rnd2 := p.random(1, 30)
			if rnd2 == 30 {
				items = append(items, &Item{ItemType: "kakkakikkare", X: x, Y: y})
			} else {
				items = append(items, &Item{ItemType: "vuoksikivi", X: x, Y: y})
			}
		case rnd <= 39:
			rnd2 := p.random(0, 100)
			var wandType string
			switch {
			case rnd2 <= 25:
				wandType = "wand_level_04"
			case rnd2 <= 50:
				wandType = "wand_unshuffle_04"
			case rnd2 <= 75:
				wandType = "wand_level_05"
			case rnd2 <= 90:
				wandType = "wand_unshuffle_05"
			case rnd2 <= 96:
				wandType = "wand_level_06"
			case rnd2 <= 98:
				wandType = "wand_unshuffle_06"
			case rnd2 <= 99:
				wandType = "wand_level_06"
			default:
				wandType = "wand_level_10"
			}
			items = append(items, createWandItem(ws, ng, x, y, wandType, false, noMoreShuffle))
		case rnd <= 60:
			rnd2 := p.random(0, 100)
			switch {
			case rnd2 <= 89:
				items = append(items, &Item{ItemType: "heart", X: x, Y: y})
			case rnd2 <= 99:
				items = append(items, &Item{ItemType: "heart_bigger", X: x, Y: y})
			default:
				items = append(items, &Item{ItemType: "full_heal", X: x, Y: y})
			}
		case rnd <= 98:
			count += 2
		default:
			count += 3
		}
	}

	return &ChestResult{Type: "great_chest", Items: deduplicateItems(items), X: x, Y: y}
}

// GenerateChest mirrors generateChest() from chest_generation.js.
func GenerateChest(ws uint32, ng int, x, y float64, noMoreShuffle, greedCurse bool) *ChestResult {
	x = roundRNGPos(x)
	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x+509.7, y+683.1)

	var items []*Item
	count := 1

	for count > 0 {
		count--
		rnd := p.random(1, 100)
		switch {
		case rnd <= 7:
			items = append(items, &Item{ItemType: "bomb", X: x, Y: y})

		case rnd <= 40:
			totalGold := 0
			rnd2 := p.random(0, 100)
			var amount int
			switch {
			case rnd2 <= 80:
				amount = 7
			case rnd2 <= 95:
				amount = 10
			default:
				amount = 20
			}
			rnd2 = p.random(0, 100)
			switch {
			case rnd2 > 30 && rnd2 <= 80:
				p.next()
				p.next()
				totalGold += 50
			case rnd2 <= 95:
				p.next()
				p.next()
				totalGold += 200
			case rnd2 <= 99:
				p.next()
				p.next()
				totalGold += 1000
			default:
				tamount := p.random(1, 3)
				for i := 0; i < tamount; i++ {
					p.next()
					p.next()
					totalGold += 50
				}
				if p.random(0, 100) > 50 {
					tamount = p.random(1, 3)
					for i := 0; i < tamount; i++ {
						p.next()
						p.next()
						totalGold += 200
					}
				}
				if p.random(0, 100) > 80 {
					tamount = p.random(1, 3)
					for i := 0; i < tamount; i++ {
						p.next()
						p.next()
						totalGold += 1000
					}
				}
			}
			for i := 0; i < amount; i++ {
				p.next()
				p.next()
				totalGold += 10
			}
			items = append(items, &Item{ItemType: "gold", Amount: totalGold, X: x, Y: y})

		case rnd <= 50:
			rnd2 := p.random(0, 100)
			switch {
			case rnd2 <= 94:
				items = append(items, createPotion(ws, ng, x+510, y+683, "normal", "normal"))
			case rnd2 <= 98:
				items = append(items, createPowderPouch(ws, ng, x+510, y+683))
			default:
				rnd3 := p.random(0, 100)
				if rnd3 <= 98 {
					items = append(items, createPotion(ws, ng, x+510, y+683, "secret", "normal"))
				} else {
					items = append(items, createPotion(ws, ng, x+510, y+683, "random", "normal"))
				}
			}

		case rnd <= 54:
			rnd2 := p.random(0, 100)
			if rnd2 <= 98 {
				items = append(items, &Item{ItemType: "spell_refresh", X: x, Y: y})
			} else {
				items = append(items, &Item{ItemType: "refresh_mimic", X: x, Y: y})
			}

		case rnd <= 60:
			opts := []string{"kammi", "kuu", "ukkoskivi", "paha_silma", "kiuaskivi", "???", "chaos_die", "shiny_orb"}
			selected := p.random(0, len(opts)-1)
			if opts[selected] == "???" {
				runestoneOpts := []string{"runestone_light", "runestone_fire", "runestone_magma", "runestone_weight", "runestone_emptiness", "runestone_edges", "runestone_metal"}
				rsIdx := p.random(0, len(runestoneOpts)-1)
				items = append(items, &Item{ItemType: runestoneOpts[rsIdx], X: x, Y: y})
			} else if opts[selected] == "chaos_die" {
				if isSpellUnlocked(363) {
					if greedCurse {
						items = append(items, &Item{ItemType: "greed_die", X: x, Y: y})
					} else {
						items = append(items, &Item{ItemType: "chaos_die", X: x, Y: y})
					}
				} else {
					items = append(items, &Item{ItemType: "blocked_by_unlock", X: x, Y: y})
					items = append(items, createPotion(ws, ng, x, y-12, "normal", "normal"))
				}
			} else if opts[selected] == "shiny_orb" {
				if greedCurse {
					items = append(items, &Item{ItemType: "greed_orb", X: x, Y: y})
				} else {
					items = append(items, &Item{ItemType: "shiny_orb", X: x, Y: y})
				}
			} else {
				items = append(items, &Item{ItemType: opts[selected], X: x, Y: y})
			}

		case rnd <= 65:
			amount := 1
			rnd2 := p.random(0, 100)
			switch {
			case rnd2 <= 50:
				amount = 1
			case rnd2 <= 70:
				amount = 2
			case rnd2 <= 80:
				amount = 3
			case rnd2 <= 90:
				amount = 4
			default:
				amount = 5
			}
			for i := 0; i < amount; i++ {
				p.next() // consume the extra next() in the JS
				spell := MakeRandomSpell(p)
				items = append(items, &Item{ItemType: "spell", Spell: spell, X: x, Y: y})
			}

		case rnd <= 84:
			rnd2 := p.random(0, 100)
			var wandType string
			switch {
			case rnd2 <= 25:
				wandType = "wand_level_01"
			case rnd2 <= 50:
				wandType = "wand_unshuffle_01"
			case rnd2 <= 75:
				wandType = "wand_level_02"
			case rnd2 <= 90:
				wandType = "wand_unshuffle_02"
			case rnd2 <= 96:
				wandType = "wand_level_03"
			case rnd2 <= 98:
				wandType = "wand_unshuffle_03"
			case rnd2 <= 99:
				wandType = "wand_level_04"
			default:
				wandType = "wand_unshuffle_04"
			}
			items = append(items, createWandItem(ws, ng, x, y, wandType, true, noMoreShuffle))

		case rnd <= 95:
			rnd2 := p.random(0, 100)
			switch {
			case rnd2 <= 88:
				items = append(items, &Item{ItemType: "heart", X: x, Y: y})
			case rnd2 <= 89:
				items = append(items, &Item{ItemType: "heart_mimic", X: x, Y: y})
			case rnd2 <= 99:
				items = append(items, &Item{ItemType: "heart_bigger", X: x, Y: y})
			default:
				items = append(items, &Item{ItemType: "full_heal", X: x, Y: y})
			}

		case rnd <= 98:
			items = append(items, &Item{ItemType: "gold", Amount: 200, X: x, Y: y})

		case rnd <= 99:
			count += 2

		default:
			count += 3
		}
	}

	return &ChestResult{Type: "chest", Items: deduplicateItems(items), X: x, Y: y}
}
