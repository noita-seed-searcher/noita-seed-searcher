package main

// Alchemy algorithm ported from wasm/entries/alchemy.zig.

var alchemyLiquids = []int{
	0,  // acid
	1,  // alcohol
	2,  // blood
	3,  // blood_fungi
	4,  // blood_worm
	5,  // cement
	6,  // lava
	7,  // magic_liquid_berserk
	8,  // magic_liquid_charm
	9,  // magic_liquid_faster_levitation
	10, // magic_liquid_faster_levitation_and_movement
	11, // magic_liquid_invisibility
	12, // magic_liquid_mana_regeneration
	13, // magic_liquid_movement_faster
	14, // magic_liquid_protection_all
	15, // magic_liquid_teleportation
	16, // magic_liquid_unstable_polymorph
	17, // magic_liquid_unstable_teleportation
	18, // magic_liquid_worm_attractor
	19, // material_confusion
	20, // mud
	21, // oil
	22, // poison
	23, // radioactive_liquid
	24, // swamp
	25, // urine
	26, // water
	27, // water_ice
	28, // water_swamp
	29, // magic_liquid_random_polymorph
}

var alchemySolids = []int{
	30, // bone
	31, // brass
	32, // coal
	33, // copper
	34, // diamond
	35, // fungi
	36, // gold
	37, // grass
	38, // gunpowder
	39, // gunpowder_explosive
	40, // rotten_meat
	41, // sand
	42, // silver
	43, // slime
	44, // snow
	45, // soil
	46, // wax
	47, // honey
}

func alchemyContains(items []int, length int, item int) bool {
	for _, v := range items[:length] {
		if v == item {
			return true
		}
	}
	return false
}

func alchemyPickMaterials(rng *NollaPrng, materials []int, count *int, source []int, needed int) {
	counter := 0
	failed := 0
	for counter < needed && failed < 99999 {
		index := int(rng.next() * float64(len(source)))
		picked := source[index]
		if !alchemyContains(materials, *count, picked) {
			materials[*count] = picked
			*count++
			counter++
		} else {
			failed++
		}
	}
}

func alchemyPickForOutput(rng *NollaPrng, worldSeed uint32, output []int) {
	materials := make([]int, 4)
	count := 0
	alchemyPickMaterials(rng, materials, &count, alchemyLiquids, 3)
	alchemyPickMaterials(rng, materials, &count, alchemySolids, 1)

	// Shuffle with a separate RNG seeded from worldSeed
	shuffleRng := newNollaPrng(float64(worldSeed>>1) + 12534.0)
	for i := count - 1; i >= 0; i-- {
		limit := float64(i + 1)
		index := int(shuffleRng.next() * limit)
		materials[i], materials[index] = materials[index], materials[i]
	}

	rng.next()
	rng.next()

	output[0] = materials[0]
	output[1] = materials[1]
	output[2] = materials[2]
}

// PickAlchemy computes the 6 alchemy materials for a world seed.
// Returns [lc0, lc1, lc2, ap0, ap1, ap2] as material indices.
func PickAlchemy(worldSeed uint32) [6]int {
	rng := newNollaPrng(float64(worldSeed)*0.17127000 + 1323.59030000)
	for i := 0; i < 5; i++ {
		rng.next()
	}

	var result [6]int
	alchemyPickForOutput(&rng, worldSeed, result[0:3])
	alchemyPickForOutput(&rng, worldSeed, result[3:6])
	return result
}

// AlchemyResult holds human-readable alchemy materials.
type AlchemyResult struct {
	LC [3]string
	AP [3]string
}

func GetAlchemyResult(worldSeed uint32) AlchemyResult {
	indices := PickAlchemy(worldSeed)
	var res AlchemyResult
	for i := 0; i < 3; i++ {
		res.LC[i] = wasmMaterials[indices[i]]
		res.AP[i] = wasmMaterials[indices[i+3]]
	}
	return res
}
