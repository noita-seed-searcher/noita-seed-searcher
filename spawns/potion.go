package main

import "time"

// Item represents any generated item (potion, wand, gold, etc.)
type Item struct {
	ItemType string // "potion", "pouch", "wand", "gold", "spell", "heart", ...
	Material string // for potions/pouches
	Spell    string // for spell items
	Amount   int    // for gold
	Count    int    // deduplicated count
	X, Y     float64
	Wand     *Wand // for wand items
	Active   bool  // for runestones
	Sub      string // item sub-identifier e.g. "heart_bigger"
}

func (it *Item) Key() string {
	if it.Wand != nil {
		return it.Wand.Sprite // wands keyed by sprite
	}
	return it.ItemType + ":" + it.Material + ":" + it.Spell + ":" + it.Sub
}

// createPotion mirrors createPotion() from potion_generation.js.
func createPotion(ws uint32, ng int, x, y float64, potionType, gameMode string) *Item {
	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x-4.5, y-4)

	switch potionType {
	case "normal":
		if gameMode == "nightmare" {
			rnd := p.random(1, 100)
			var mat string
			if rnd <= 50 {
				mat = potionMaterialsMagicNightmare[p.random(0, len(potionMaterialsMagicNightmare)-1)]
			} else {
				mat = potionMaterialsStandard[p.random(0, len(potionMaterialsStandard)-1)]
			}
			return &Item{ItemType: "potion", Material: mat, X: x, Y: y}
		}

		rnd := p.random(0, 100)
		var mat string
		if rnd <= 75 {
			if p.random(0, 100000) <= 50 {
				mat = "magic_liquid_hp_regeneration"
			} else if p.random(200, 100000) <= 250 {
				mat = "purifying_powder"
			} else if p.random(250, 100000) <= 500 {
				mat = "magic_liquid_weakness"
			} else {
				mat = potionMaterialsMagic[p.random(0, len(potionMaterialsMagic)-1)]
			}
		} else {
			mat = potionMaterialsStandard[p.random(0, len(potionMaterialsStandard)-1)]
		}

		// Holiday overrides
		now := time.Now()
		m, d := int(now.Month()), now.Day()
		if (m == 4 && d == 30) || (m == 5 && d == 1) {
			if p.random(0, 100) <= 20 {
				if p.random(0, 5) <= 4 {
					mat = "sima"
				} else {
					mat = "beer"
				}
			}
		}
		if m == 6 && d >= 19 && d <= 25 {
			if p.random(0, 100) <= 9 {
				if p.random(0, 3) <= 2 {
					mat = "juhannussima"
				} else {
					mat = "beer"
				}
			}
		}
		if (m == 3 && d >= 29) || (m == 4 && d <= 4) {
			if p.random(0, 100) <= 10 {
				mat = "mammi"
			}
		}
		if m == 2 && d == 14 {
			if p.random(0, 100) <= 8 {
				mat = "magic_liquid_charm"
			}
		}
		return &Item{ItemType: "potion", Material: mat, X: x, Y: y}

	case "secret":
		mat := potionMaterialsSecret[p.random(0, len(potionMaterialsSecret)-1)]
		return &Item{ItemType: "potion", Material: mat, X: x, Y: y}

	case "random":
		var mat string
		if p.random(0, 100) <= 50 {
			mat = potionLiquids[p.random(0, len(potionLiquids)-1)]
		} else {
			mat = potionSands[p.random(0, len(potionSands)-1)]
		}
		return &Item{ItemType: "potion", Material: mat, X: x, Y: y}
	}
	return nil
}

// createPowderPouch mirrors createPowderPouch() from potion_generation.js.
func createPowderPouch(ws uint32, ng int, x, y float64) *Item {
	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x-4.5, y-5.5)

	rnd := p.random(0, 100)
	var mat string
	if rnd <= 75 {
		mat = powderMaterialsMagic[p.random(0, len(powderMaterialsMagic)-1)]
	} else {
		mat = powderMaterialsStandard[p.random(0, len(powderMaterialsStandard)-1)]
	}
	return &Item{ItemType: "pouch", Material: mat, X: x, Y: y}
}

// generateItem mirrors generateItem() from potion_generation.js.
func generateItem(ws uint32, ng int, x, y float64, greedCurse bool) *Item {
	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x, y)
	rnd := p.random(1, 1000)
	if rnd > 995 && y >= 512*3 {
		return &Item{ItemType: "mimic_potion", X: x, Y: y}
	}

	p.setRandomSeed(ws+uint32(ng), x+425, y-243)
	rnd = p.random(1, 91)
	switch {
	case rnd <= 65:
		return createPotion(ws, ng, x, y-2, "normal", "normal")
	case rnd <= 70:
		return createPowderPouch(ws, ng, x, y-2)
	case rnd <= 71:
		if isSpellUnlocked(363) {
			if greedCurse {
				return &Item{ItemType: "greed_die", X: x, Y: y}
			}
			return &Item{ItemType: "chaos_die", X: x, Y: y}
		}
		return nil
	case rnd <= 72:
		runestoneOpts := []string{"runestone_light", "runestone_fire", "runestone_magma", "runestone_weight", "runestone_emptiness", "runestone_edges", "runestone_metal"}
		p.setRandomSeed(ws, x+2617.941, y-1229.3581)
		idx := p.random(0, 6)
		active := p.random(1, 10) == 2
		return &Item{ItemType: runestoneOpts[idx], Active: active, X: x, Y: y}
	case rnd <= 73:
		return &Item{ItemType: "egg_purple", X: x, Y: y}
	case rnd <= 77:
		return &Item{ItemType: "egg_slime", X: x, Y: y}
	case rnd <= 79:
		return &Item{ItemType: "egg_monster", X: x, Y: y}
	case rnd <= 83:
		return &Item{ItemType: "kiuaskivi", X: x, Y: y}
	case rnd <= 85:
		return &Item{ItemType: "ukkoskivi", X: x, Y: y}
	case rnd <= 89:
		return &Item{ItemType: "broken_wand", X: x, Y: y}
	default:
		if greedCurse {
			return &Item{ItemType: "greed_orb", X: x, Y: y}
		}
		return &Item{ItemType: "shiny_orb", X: x, Y: y}
	}
}

// generateItemLiquidcave mirrors generateItemLiquidcave() from potion_generation.js.
func generateItemLiquidcave(ws uint32, ng int, x, y float64) *Item {
	p := newPrng(0)
	p.setRandomSeed(ws+uint32(ng), x+425, y-243)
	rnd := p.random(1, 86)
	switch {
	case rnd <= 49:
		return createPotion(ws, ng, x, y-2, "normal", "normal")
	case rnd <= 52:
		return &Item{ItemType: "egg_purple", X: x, Y: y}
	case rnd <= 55:
		return &Item{ItemType: "egg_fire", X: x, Y: y}
	case rnd <= 58:
		return &Item{ItemType: "egg_slime", X: x, Y: y}
	case rnd <= 64:
		return &Item{ItemType: "egg_monster", X: x, Y: y}
	case rnd <= 70:
		return &Item{ItemType: "kiuaskivi", X: x, Y: y}
	case rnd <= 76:
		return &Item{ItemType: "ukkoskivi", X: x, Y: y}
	case rnd <= 82:
		return &Item{ItemType: "kuu", X: x, Y: y}
	default:
		return &Item{ItemType: "broken_wand", X: x, Y: y}
	}
}

// SpawnItem generates an item at the given position for a biome.
func SpawnItem(ws uint32, ng int, x, y float64, biome string, greedCurse bool) *Item {
	var item *Item
	if biome == "liquidcave" {
		item = generateItemLiquidcave(ws, ng, x, y)
	} else {
		item = generateItem(ws, ng, x, y, greedCurse)
	}
	if item != nil {
		item.X = x
		item.Y = y
	}
	return item
}
