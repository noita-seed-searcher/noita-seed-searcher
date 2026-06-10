package main

// ProvideAlwaysCastPos mirrors providePos from AlwaysCast.ts.
func ProvideAlwaysCastPos(rng *RNG, x, y float64) string {
	rng.SetRandomSeed(x, y)

	goodCards := []string{"DAMAGE", "CRITICAL_HIT", "HOMING", "SPEED", "ACID_TRAIL", "SINEWAVE"}
	card := goodCards[rng.RandomInt(1, int32(len(goodCards)))-1]

	r := rng.RandomInt(1, 100)
	level := 6

	if r <= 50 {
		p := rng.RandomInt(1, 100)
		if p <= 86 {
			card = GetRandomActionWithType(rng, x, y, level, ACTION_MODIFIER, 666)
		} else if p <= 93 {
			card = GetRandomActionWithType(rng, x, y, level, ACTION_STATIC_PROJECTILE, 666)
		} else if p < 100 {
			card = GetRandomActionWithType(rng, x, y, level, ACTION_PROJECTILE, 666)
		} else {
			card = GetRandomActionWithType(rng, x, y, level, ACTION_UTILITY, 666)
		}
	}

	return card
}

// ProvideAlwaysCast mirrors provide from AlwaysCast.ts.
// templeLevel: 0-indexed temple index
// perkNumber: 0-indexed perk position in the row
// perksOnLevel: total perks in that row (e.g. 3)
func ProvideAlwaysCast(rng *RNG, templeLevel, perkNumber, perksOnLevel int, worldOffset int) string {
	if templeLevel >= len(templeLocations) {
		return ""
	}
	temple := templeLocations[templeLevel]
	y := temple.Y
	// x = RoundHalfOfEven(temple.x + (perkNumber + 0.5) * (60 / perksOnLevel)) + 35840 * worldOffset
	rawX := temple.X + (float64(perkNumber)+0.5)*(60.0/float64(perksOnLevel))
	x := float64(roundHalfToEvenI32(rawX)) + float64(35840*worldOffset)
	return ProvideAlwaysCastPos(rng, x, y)
}
