package main

// Coalmine pixel-scene chest sources.
//
// In Noita many great chests (and therefore true_orbs) are not placed by a bare
// spawn_chest tile — they live inside a *pixel scene* that a load_pixel_scene /
// load_oiltank / load_pixel_scene2 tile loads. Telescope reproduces this by
// picking a scene (a weighted roll keyed on the load point), then dispatching the
// spawn pixels baked into that scene's PNG at offsets from the scene's top-left.
//
// This file ports just the chest-bearing slice of that pipeline for coalmine:
//   - the scene lists + weights (COALMINE_SCENES in pixel_scene_config.js), and
//   - the chest spawn pixels found in each scene PNG (the prescanPixelScene step),
//     scanned once from noita-telescope/data/pixel_scenes/coalmine/*.png.
// That is enough for the true_orb searcher to see orbs that live in scene chests,
// which a tile-only scan misses entirely (e.g. seed 142278984 @ -34581,951, a
// coalpit02 chest one parallel world over).
//
// Known gaps (the user accepted "chest-bearing scenes only"): the
// CHECK_PIXEL_SCENE_BIOME corner check is not applied (it needs edge noise, which
// is not ported), so a scene sitting across a biome edge — which telescope would
// reject — can still be reported; and non-chest scene contents are ignored.

// sceneOpt is one weighted entry in a scene list; order and prob must match
// telescope so the selection roll lands on the same scene.
type sceneOpt struct {
	prob float64
	name string
}

// chestSpawn is a chest-yielding spawn pixel inside a scene PNG, at (offX, offY)
// from the scene's top-left, to be dispatched through funcName.
type chestSpawn struct {
	offX, offY float64
	funcName   string
}

// chestCand is a resolved, dispatch-ready chest spawn in world coordinates.
type chestCand struct {
	funcName string
	x, y     float64
}

// coalmineSceneLists mirrors COALMINE_SCENES (pixel_scene_config.js). Only the
// lists reachable by a chest-bearing loader are needed.
var coalmineSceneLists = map[string][]sceneOpt{
	"g_pixel_scene_01": {
		{0.5, "coalpit01"}, {0.5, "coalpit02"}, {0.5, "carthill"},
		{0.5, "coalpit03"}, {0.5, "coalpit04"}, {0.5, "coalpit05"},
	},
	"g_pixel_scene_02": {
		{0.5, "shrine01"}, {0.5, "shrine02"}, {0.5, "slimepit"}, {0.5, "laboratory"},
		{0.5, "swarm"}, {0.5, "symbolroom"}, {0.5, "physics_01"}, {0.5, "physics_02"},
		{0.5, "physics_03"}, {1.5, "shop"}, {0.5, "radioactivecave"},
		{0.75, "wandtrap_h_02"}, {0.75, "wandtrap_h_04"}, {0.75, "wandtrap_h_06"},
		{0.75, "wandtrap_h_07"}, {0.5, "physics_swing_puzzle"}, {0.5, "receptacle_oil"},
	},
	"g_oiltank": {
		{1.0, "oiltank_1"}, {0.0004, "oiltank_1"}, {0.01, "oiltank_2"}, {1.0, "oiltank_2"},
		{1.0, "oiltank_3"}, {1.0, "oiltank_4"}, {1.0, "oiltank_5"}, {0.05, "oiltank_puzzle"},
	},
	"g_oiltank_alt": {
		{1.0, "oiltank_alt"},
	},
}

// coalmineSceneChests holds the chest-yielding spawn pixels per scene, scanned
// from data/pixel_scenes/coalmine/<scene>.png. Scenes with no entry have no chest
// pixel. Offsets are relative to the scene's top-left (= the load point).
var coalmineSceneChests = map[string][]chestSpawn{
	"coalpit02":            {{94, 224, "spawn_chest"}},
	"coalpit05":            {{66, 215, "spawn_heart"}},
	"shrine01":             {{133, 86, "spawn_heart"}},
	"slimepit":             {{133, 107, "spawn_heart"}},
	"physics_03":           {{140, 113, "spawn_heart"}},
	"radioactivecave":      {{130, 44, "spawn_heart"}},
	"physics_swing_puzzle": {{199, 89, "spawn_swing_puzzle_target"}},
	"oiltank_puzzle":       {{55, 234, "spawn_oiltank_puzzle"}},
}

// pixelSceneLoaders are the coalmine tile functions that load a pixel scene and
// thus may contain a chest. Used to decide when to expand a detected spawn.
var pixelSceneLoaders = map[string]bool{
	"load_pixel_scene":  true,
	"load_pixel_scene2": true,
	"load_oiltank":      true,
	"load_oiltank_alt":  true,
}

// selectScene ports loadRandomPixelScene's weighted pick: r = ProceduralRandom *
// total, then walk the list subtracting weights. Zero-or-negative weights are
// skipped (never selected), matching the JS. Returns "" if nothing is picked.
func selectScene(ws uint32, ng int, x, y float64, list []sceneOpt) string {
	var total float64
	for _, s := range list {
		total += s.prob
	}
	prng := &NollaPrng{}
	r := prng.proceduralRandom(ws+uint32(ng), x, y) * total
	for _, s := range list {
		if s.prob <= 0 {
			continue
		}
		if r <= s.prob {
			return s.name
		}
		r -= s.prob
	}
	return ""
}

// coalmineSceneChestSpawns expands a coalmine pixel-scene loader at load point
// (x, y) — already parallel-world-shifted by the caller — into the chest spawn
// points of whichever scene it selects. The selection rolls (the load_pixel_scene
// / load_oiltank coin flip and the weighted scene pick) are keyed on the load
// point, so they differ per parallel world, which is the point of scanning them.
func coalmineSceneChestSpawns(funcName string, ws uint32, ng int, x, y float64) []chestCand {
	var listKey string
	switch funcName {
	case "load_pixel_scene":
		prng := &NollaPrng{}
		prng.setRandomSeed(ws+uint32(ng), x, y)
		if prng.random(1, 100) > 50 {
			listKey = "g_oiltank"
		} else {
			listKey = "g_pixel_scene_01"
		}
	case "load_oiltank":
		prng := &NollaPrng{}
		prng.setRandomSeed(ws+uint32(ng), x, y)
		if prng.random(1, 100) <= 50 {
			listKey = "g_oiltank"
		} else {
			listKey = "g_pixel_scene_01"
		}
	case "load_oiltank_alt":
		listKey = "g_oiltank_alt"
	case "load_pixel_scene2":
		listKey = "g_pixel_scene_02"
	default:
		return nil
	}
	scene := selectScene(ws, ng, x, y, coalmineSceneLists[listKey])
	chests := coalmineSceneChests[scene]
	if len(chests) == 0 {
		return nil
	}
	out := make([]chestCand, 0, len(chests))
	for _, c := range chests {
		out = append(out, chestCand{c.funcName, x + c.offX, y + c.offY})
	}
	return out
}
