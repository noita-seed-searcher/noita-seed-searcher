package main

import "strings"

// Port of the item-producing subset of spawn_functions.js spawnSwitch, wiring
// each detected spawn point to the package's existing generators. Non-item
// functions (enemies/props/lamps/structures/puzzles) return nil. Pixel-scene /
// shop functions are flagged as unsupported pending the pixel-scene subsystem.
//
// Note: telescope re-resolves the biome at each point via
// getBiomeAtWorldCoordinates (edge noise) and applies blockEdgeSpawns /
// excludeEdgeCases settings before dispatch; those are not yet applied here, so
// we dispatch with the source biome. This affects only spawn points near biome
// edges.
func spawnSwitchItem(funcName string, ws uint32, ng int, x, y float64, biome, gameMode string) *Spawn {
	switch funcName {
	case "spawn_chest":
		res := SpawnChest(ws, ng, x, y, strings.Contains(biome, "tower"), false, false)
		if res == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: res.Type, X: x, Y: y, Chest: res}

	case "spawn_heart":
		s := spawnHeart(ws, ng, x, y, biome)
		if s != nil {
			s.FuncName = funcName
		}
		return s

	case "spawn_bbqbox":
		prng := &NollaPrng{}
		prng.setRandomSeed(ws+uint32(ng), x, y)
		var s *Spawn
		if prng.random(1, 100) <= 99 {
			s = spawnHeart(ws, ng, x+10, y+10, biome)
		} else {
			s = spawnJar(x, y)
		}
		if s != nil {
			s.FuncName = funcName
		}
		return s

	case "spawn_trapwand":
		options := []string{"premade_1", "premade_2", "premade_3", "premade_4", "premade_5",
			"premade_6", "premade_7", "premade_8", "premade_9", "wand_level_01"}
		prng := &NollaPrng{}
		prng.setRandomSeed(ws+uint32(ng), x, y)
		rnd := prng.random(1, len(options))
		w := GenerateWand(ws, ng, options[rnd-1], x, y)
		if w == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: "wand", X: x, Y: y, Item: &Item{ItemType: "wand", Wand: w, X: x, Y: y}}

	case "spawn_props3":
		// In coalmine, props3 rolls for a potion.
		prng := &NollaPrng{}
		prng.setRandomSeed(ws+uint32(ng), x, y)
		r := prng.next() * 0.4
		if r > 0.1 {
			it := createPotion(ws, ng, x+5, y, "normal", gameMode)
			return &Spawn{FuncName: funcName, Kind: "potion", X: x, Y: y, Item: it}
		}
		return nil

	case "spawn_potions", "spawn_potion":
		it := SpawnItem(ws, ng, x, y, biome, false)
		if it == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: "item", X: x, Y: y, Item: it}

	case "spawn_potion_altar":
		it := SpawnPotionAltar(ws, ng, x, y, biome, gameMode, false)
		if it == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: "potion_altar", X: x, Y: y, Item: it}

	case "spawn_items", "spawn_wand_altar":
		it := SpawnWandAltar(ws, ng, x, y, biome, false)
		if it == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: "wand_altar", X: x, Y: y, Item: it}

	case "spawn_wands", "spawn_wand":
		it := SpawnWand(ws, ng, x, y, biome, false)
		if it == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: "wand", X: x, Y: y, Item: it}

	case "load_oiltank", "load_oiltank_alt", "load_altar", "load_pixel_scene", "load_pixel_scene2", "spawn_shopitem":
		// Need the pixel-scene / shop subsystem (deferred).
		return &Spawn{FuncName: funcName, Kind: "pixel_scene", X: x, Y: y, Note: "unsupported (pixel scene / shop)"}

	default:
		return nil
	}
}

// listNaturalSpawns ties the whole chain together: biome map -> region detect ->
// tiling+hacks -> spawn-point scan -> dispatch, returning every item-producing
// natural spawn on the coalmine (first) level for a seed. ng>0 (procedural map)
// is not yet handled here (the coalmine color is palette-shuffled in NG+).
func listNaturalSpawns(seed uint32, ng int) ([]*Spawn, error) {
	bm, err := generateBiomeData(seed, ng, "normal")
	if err != nil {
		return nil, err
	}
	ts, err := buildBiomeTileset("data/wang_tiles/coalmine.png")
	if err != nil {
		return nil, err
	}
	regions, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, coalmineColor)

	var spawns []*Spawn
	for i := range bboxes {
		layer := generateTileLayer(bboxes[i], regions[i], ts, seed, ng, "coalmine", "normal", nil)
		if layer == nil {
			continue
		}
		for _, d := range prescanSpawnFunctions(layer, ng > 0, "normal") {
			if s := spawnSwitchItem(d.funcName, seed, ng, float64(d.x), float64(d.y), "coalmine", "normal"); s != nil {
				spawns = append(spawns, s)
			}
		}
	}
	return spawns, nil
}
