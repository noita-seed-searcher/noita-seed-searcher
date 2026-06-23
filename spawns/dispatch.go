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

	case "spawn_swing_puzzle_target":
		// The swing puzzle drops a chest at a fixed offset (spawn_functions.js).
		res := SpawnChest(ws, ng, x-75, y-70, false, false, false)
		if res == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: res.Type, X: x - 75, Y: y - 70, Chest: res}

	case "spawn_oiltank_puzzle":
		// The oiltank puzzle's material roll doesn't affect the chest (it reseeds
		// on its own position), so we skip straight to the chest at y-25.
		res := SpawnChest(ws, ng, x, y-25, false, false, false)
		if res == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: res.Type, X: x, Y: y - 25, Chest: res}

	case "spawn_trapwand":
		options := []string{"premade_1", "premade_2", "premade_3", "premade_4", "premade_5",
			"premade_6", "premade_7", "premade_8", "premade_9", "wand_level_01"}
		prng := &NollaPrng{}
		prng.setRandomSeed(ws+uint32(ng), x, y)
		rnd := prng.random(1, len(options))
		// generateWandByType (not GenerateWand) so premade_* types are handled.
		w := generateWandByType(ws, ng, x, y, options[rnd-1], false)
		if w == nil {
			return nil
		}
		return &Spawn{FuncName: funcName, Kind: "wand", X: x, Y: y, Item: &Item{ItemType: "wand", Wand: w, X: x, Y: y}}

	case "spawn_props3":
		prng := &NollaPrng{}
		prng.setRandomSeed(ws+uint32(ng), x, y)
		switch biome {
		case "coalmine", "coalmine_alt":
			if prng.next()*0.4 > 0.1 {
				it := createPotion(ws, ng, x+5, y, "normal", gameMode)
				return &Spawn{FuncName: funcName, Kind: "potion", X: x, Y: y, Item: it}
			}
		case "snowcastle":
			r := prng.next() * 1.325
			if r > 1.3 {
				it := createPotion(ws, ng, x+5, y, "normal", gameMode)
				return &Spawn{FuncName: funcName, Kind: "potion", X: x, Y: y, Item: it}
			} else if r > 1.2 {
				return &Spawn{FuncName: funcName, Kind: "potion", X: x + 5, Y: y, Item: &Item{ItemType: "potion", Material: "alcohol", X: x + 5, Y: y}}
			}
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

// listNaturalSpawns ties the whole chain together: biome map -> per-biome
// region detect -> tiling+hacks -> spawn-point scan -> dispatch, returning
// every item-producing natural spawn across all biomes and parallel worlds for a seed.
func listNaturalSpawns(seed uint32, ng, pwMax, pwMaxV int) ([]*Spawn, error) {
	bm, err := generateBiomeData(seed, ng, "normal")
	if err != nil {
		return nil, err
	}
	isNGP := ng > 0
	worldSize := 70 * 512
	if isNGP {
		worldSize = 64*512 - 8
	}
	const pwVSize = 24570

	type spawnPoint struct {
		funcName string
		x0, y0   float64
		biome    string
	}

	tilesetCache := map[string]*stbhwTileset{}
	var points []spawnPoint
	for _, entry := range biomeConfig {
		if entry.wangFile == "" {
			continue
		}
		ts, ok := tilesetCache[entry.wangFile]
		if !ok {
			ts, err = buildBiomeTileset(entry.wangFile)
			if err != nil {
				return nil, err
			}
			tilesetCache[entry.wangFile] = ts
		}
		regions, bboxes := findBiomeRegions(bm.Pixels, bm.W, bm.H, entry.color)
		for i := range bboxes {
			layer := generateTileLayer(bboxes[i], regions[i], ts, seed, ng, entry.biomeName, "normal", entry.randomColors)
			if layer == nil {
				continue
			}
			for _, d := range prescanSpawnFunctions(layer, ng > 0, "normal") {
				points = append(points, spawnPoint{d.funcName, float64(d.x), float64(d.y), entry.biomeName})
			}
		}
	}

	var spawns []*Spawn
	for pwV := -pwMaxV; pwV <= pwMaxV; pwV++ {
		for pw := -pwMax; pw <= pwMax; pw++ {
			dx := float64(pw * worldSize)
			dy := float64(pwV * pwVSize)
			for _, p := range points {
				s := spawnSwitchItem(p.funcName, seed, ng, p.x0+dx, p.y0+dy, p.biome, "normal")
				if s != nil {
					s.Biome = p.biome
					s.PW = pw
					s.PWV = pwV
					spawns = append(spawns, s)
				}
			}
		}
	}
	return spawns, nil
}
