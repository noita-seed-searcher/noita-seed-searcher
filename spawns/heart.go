package main

// Port of noita-telescope/js/heart_generation.js (spawnHeart) and spawnJar
// from potion_generation.js. spawnHeart decides between a heart, a chest/great
// chest, a mimic, a leggy chest, or nothing, reusing the existing chest gens.

// Spawn is a generated natural-spawn result. Exactly one of Chest/Item is set
// for loot kinds; marker kinds (heart/mimic/jar/...) carry only Kind + coords.
type Spawn struct {
	FuncName string
	Biome    string
	Kind     string // heart, mimic, chest_leggy, jar, chest, great_chest, wand, potion, item, pixel_scene, shop, puzzle
	X, Y     float64
	Chest    *ChestResult
	Item     *Item
	Note     string // pixel-scene/shop name or "unsupported"
}

// spawnHeart ports heart_generation.js spawnHeart (non-Valentine's rate 0.7).
func spawnHeart(ws uint32, ng int, x, y float64, biome string) *Spawn {
	prng := &NollaPrng{}
	r := prng.proceduralRandom(ws+uint32(ng), x, y)
	const heartSpawnRate = 0.7

	if r > heartSpawnRate {
		return &Spawn{Kind: "heart", X: x, Y: y}
	}
	if r > 0.3 {
		prng.setRandomSeed(ws+uint32(ng), x+45, y-2123)
		rnd := prng.random(1, 100)
		if rnd <= 90 || y < 512*3 {
			rnd = prng.random(1, 1000)
			_ = prng.random(1, 300) // hasSign roll (unused), kept for PRNG fidelity
			if rnd >= 1000 {
				return &Spawn{Kind: "great_chest", X: x, Y: y, Chest: GenerateGreatChest(ws, ng, x, y, false)}
			}
			return &Spawn{Kind: "chest", X: x, Y: y, Chest: GenerateChest(ws, ng, x, y, false, false)}
		}
		rnd = prng.random(1, 100)
		_ = prng.random(1, 30) // hasSign roll (unused)
		if rnd <= 95 {
			return &Spawn{Kind: "mimic", X: x, Y: y}
		}
		return &Spawn{Kind: "chest_leggy", X: x, Y: y}
	}
	return nil
}

// spawnJar ports potion_generation.js spawnJar.
func spawnJar(x, y float64) *Spawn {
	return &Spawn{Kind: "jar", X: x, Y: y, Item: &Item{ItemType: "jar", Material: "urine", X: x, Y: y}}
}
