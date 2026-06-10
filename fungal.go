package main

// FungalShift algorithm ported from wasm/entries/fungal.zig.

const (
	matGold      = 36
	matGoldBox2d = 65
	matGrass     = 37
	matGrassHoly = 74
)

type fungalFromEntry struct {
	probability float64
	materials   []int
}

type fungalToEntry struct {
	probability float64
	material    int
}

var fungalMaterialsFrom = []fungalFromEntry{
	{1.0, []int{26, 48, 49, 27}},          // water, water_static, water_salt, water_ice
	{1.0, []int{6}},                        // lava
	{1.0, []int{23, 22, 50}},              // radioactive_liquid, poison, material_darkness
	{1.0, []int{21, 24, 51}},              // oil, swamp, peat
	{1.0, []int{2}},                        // blood
	{1.0, []int{3, 35, 52}},               // blood_fungi, fungi, fungisoil
	{1.0, []int{53, 4}},                   // blood_cold, blood_worm
	{1.0, []int{0}},                        // acid
	{0.4, []int{54, 55, 56, 57, 58, 59}},  // acid_gas, acid_gas_static, poison_gas, fungal_gas, radioactive_gas, radioactive_gas_static
	{0.4, []int{60, 16}},                   // magic_liquid_polymorph, magic_liquid_unstable_polymorph
	{0.4, []int{7, 8, 11}},                // magic_liquid_berserk, magic_liquid_charm, magic_liquid_invisibility
	{0.6, []int{34}},                       // diamond
	{0.6, []int{42, 31, 33}},              // silver, brass, copper
	{0.2, []int{61, 62}},                   // steam, smoke
	{0.4, []int{41}},                       // sand
	{0.4, []int{63}},                       // snow_sticky
	{0.05, []int{64}},                      // rock_static
	{0.0003, []int{36, 65}},               // gold, gold_box2d
}

var fungalMaterialsTo = []fungalToEntry{
	{1.0, 26},  // water
	{1.0, 6},   // lava
	{1.0, 23},  // radioactive_liquid
	{1.0, 21},  // oil
	{1.0, 2},   // blood
	{1.0, 3},   // blood_fungi
	{1.0, 0},   // acid
	{1.0, 28},  // water_swamp
	{1.0, 1},   // alcohol
	{1.0, 66},  // sima
	{1.0, 4},   // blood_worm
	{1.0, 22},  // poison
	{1.0, 67},  // vomit
	{1.0, 68},  // pea_soup
	{1.0, 35},  // fungi
	{0.8, 41},  // sand
	{0.8, 34},  // diamond
	{0.8, 42},  // silver
	{0.8, 61},  // steam
	{0.5, 64},  // rock_static
	{0.5, 38},  // gunpowder
	{0.5, 50},  // material_darkness
	{0.5, 19},  // material_confusion
	{0.2, 69},  // rock_static_radioactive
	{0.02, 60}, // magic_liquid_polymorph
	{0.02, 29}, // magic_liquid_random_polymorph
	{0.15, 15}, // magic_liquid_teleportation
	{0.10, 70}, // mimic_liquid
	{0.01, 25}, // urine
	{0.01, 71}, // poo
	{0.01, 72}, // void_liquid
	{0.01, 73}, // cheese_static
}

var fungalGreedyMaterials = []int{
	31, // brass
	42, // silver
	23, // radioactive_liquid
	68, // pea_soup
	54, // acid_gas
	71, // poo
	75, // mammi
	76, // rotten_meat_radioactive
	67, // vomit
}

// FungalShift holds one fungal transformation result.
type FungalShift struct {
	FlaskTo   bool
	FlaskFrom bool
	From      []string
	To        string
	GoldToX   string
	GrassToX  string
}

func fungalPickFrom(worldSeed uint32, rnd *RandomPos) fungalFromEntry {
	weightSum := 0.0
	for _, item := range fungalMaterialsFrom {
		weightSum += item.probability
	}
	val := randomNextF(worldSeed, rnd, 0.0, weightSum)
	min := 0.0
	for _, item := range fungalMaterialsFrom {
		max := min + item.probability
		if val >= min && val <= max {
			return item
		}
		min = max
	}
	return fungalMaterialsFrom[0]
}

func fungalPickTo(worldSeed uint32, rnd *RandomPos) fungalToEntry {
	weightSum := 0.0
	for _, item := range fungalMaterialsTo {
		weightSum += item.probability
	}
	val := randomNextF(worldSeed, rnd, 0.0, weightSum)
	min := 0.0
	for _, item := range fungalMaterialsTo {
		max := min + item.probability
		if val >= min && val <= max {
			return item
		}
		min = max
	}
	return fungalMaterialsTo[0]
}

// PickFungal computes up to maxShifts fungal transformations for a world seed.
func PickFungal(worldSeed uint32, maxShifts int) []FungalShift {
	if maxShifts <= 0 || maxShifts > 20 {
		maxShifts = 20
	}

	var randoms NollaPrng
	var shifts []FungalShift

	for iter := 0; iter < maxShifts; iter++ {
		convertTries := 0
		convertedAny := false

		for !convertedAny && convertTries < 20 {
			seed2 := int32(42345 + iter + 1000*convertTries)
			rnd := RandomPos{x: 9123, y: seed2}
			randoms.setRandomSeed(worldSeed, 89346, float64(seed2))

			from := fungalPickFrom(worldSeed, &rnd)
			to := fungalPickTo(worldSeed, &rnd)

			shift := FungalShift{
				To:       materialName(to.material),
				GoldToX:  materialName(matGold),
				GrassToX: materialName(matGrassHoly),
			}

			if randomNextI(worldSeed, &rnd, 1, 100) <= 75 {
				if randomNextI(worldSeed, &rnd, 1, 100) <= 50 {
					shift.FlaskFrom = true
				} else {
					shift.FlaskTo = true
					if randomNextI(worldSeed, &rnd, 1, 1000) != 1 {
						idx := randoms.random(0, int32(len(fungalGreedyMaterials)-1))
						shift.GoldToX = materialName(fungalGreedyMaterials[idx])
						shift.GrassToX = materialName(matGrass)
					}
				}
			}

			for _, mat := range from.materials {
				if len(shift.From) == 0 || mat != to.material {
					shift.From = append(shift.From, materialName(mat))
					convertedAny = true
				}
			}

			shifts = append(shifts, shift)
			convertTries++
		}
	}

	return shifts
}
