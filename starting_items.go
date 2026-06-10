package main

// StartingFlask ported from InfoProviders/StartingFlask.ts
func GetStartingFlask(rng *RNG) string {
	rng.SetRandomSeed(-4.5, -4)

	material := "unknown"
	res := rng.RandomInt(1, 100)

	if res <= 65 {
		res = rng.RandomInt(1, 100)
		switch {
		case res <= 10:
			material = "mud"
		case res <= 20:
			material = "water_swamp"
		case res <= 30:
			material = "water_salt"
		case res <= 40:
			material = "swamp"
		case res <= 50:
			material = "snow"
		default:
			material = "water"
		}
	} else if res <= 70 {
		material = "blood"
	} else if res <= 99 {
		rng.RandomInt(0, 100) // consume rng
		choices := []string{
			"acid",
			"magic_liquid_polymorph",
			"magic_liquid_random_polymorph",
			"magic_liquid_berserk",
			"magic_liquid_charm",
			"magic_liquid_movement_faster",
		}
		idx := rng.RandomInt(0, int32(len(choices)-1))
		material = choices[idx]
	} else {
		res = rng.RandomInt(0, 100000)
		switch res {
		case 666:
			material = "urine"
		case 79:
			material = "gold"
		default:
			choices := []string{"slime", "gunpowder_unstable"}
			idx := rng.RandomInt(0, int32(len(choices)-1))
			material = choices[idx]
		}
	}
	return material
}

// GetStartingSpell ported from InfoProviders/StartingSpell.ts
func GetStartingSpell(rng *RNG) string {
	rng.SetRandomSeed(0, -11)

	rng.RandomInt(80, 130) // mana_max
	rng.RandomInt(2, 3)    // deck_capacity
	rng.RandomInt(0, 0)    // ui_name (gun.name has 1 element)
	rng.RandomInt(20, 28)  // reload_time
	rng.RandomInt(9, 15)   // fire_rate_wait
	rng.RandomInt(25, 40)  // mana_charge_speed
	rng.RandomInt(1, 3)    // action_count

	actions := []string{"SPITTER", "RUBBER_BALL", "BOUNCY_ORB"}
	if rng.RandomInt(1, 100) < 50 {
		idx := rng.RandomInt(0, int32(len(actions)-1))
		return actions[idx]
	}
	return "LIGHT_BULLET"
}

// StartingBombSpell ported from InfoProviders/StartingBomb.ts
func GetStartingBombSpell(rng *RNG) string {
	rng.SetRandomSeed(-1, 0)

	rng.RandomInt(80, 110)
	rng.RandomInt(1, 1)
	rng.RandomInt(1, 10)
	rng.RandomInt(3, 8)
	rng.RandomInt(5, 20)

	res := rng.RandomInt(1, 100)
	spells := []string{"BOMB", "DYNAMITE", "MINE", "ROCKET", "GRENADE"}

	if res < 50 {
		idx := rng.RandomInt(0, int32(len(spells)-1))
		return spells[idx]
	}
	return "BOMB"
}
