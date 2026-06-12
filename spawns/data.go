package main

// probEntry represents one entry in a GUN_PROBS probability table.
type probEntry struct {
	prob      float64
	min, max  float64
	mean      float64
	sharpness float64
}

// probTable is a slice of probability entries with cached total.
type probTable struct {
	entries   []probEntry
	totalProb float64
}

func (t *probTable) ensureTotal() {
	if t.totalProb == 0 {
		for _, e := range t.entries {
			t.totalProb += e.prob
		}
	}
}

// select picks an entry using the prng's next() call.
func (t *probTable) selectEntry(p *NollaPrng) probEntry {
	t.ensureTotal()
	r := p.next() * t.totalProb
	for _, e := range t.entries {
		if r < e.prob {
			return e
		}
		r -= e.prob
	}
	return t.entries[len(t.entries)-1]
}

// gunProbs maps wandType → variable → probTable.
var gunProbs = map[string]map[string]probTable{
	"normal": {
		"deck_capacity": {entries: []probEntry{
			{1, 3, 10, 6, 2},
			{0.1, 2, 7, 4, 4},
			{0.05, 1, 5, 3, 4},
			{0.15, 5, 11, 8, 2},
			{0.12, 2, 20, 8, 4},
			{0.15, 3, 12, 6, 6},
			{1, 1, 20, 6, 0},
		}},
		"reload_time": {entries: []probEntry{
			{1, 5, 60, 30, 2},
			{0.5, 1, 100, 40, 2},
			{0.02, 1, 100, 20, 0},
			{0.35, 1, 240, 40, 0},
		}},
		"fire_rate_wait": {entries: []probEntry{
			{1, 1, 30, 5, 2},
			{0.1, 1, 50, 15, 3},
			{0.1, -15, 15, 0, 3},
			{0.45, 0, 35, 12, 0},
		}},
		"spread_degrees": {entries: []probEntry{
			{1, -5, 10, 0, 3},
			{0.1, -35, 35, 0, 0},
		}},
		"speed_multiplier": {entries: []probEntry{
			{1, 0.8, 1.2, 1.0, 6.0},
			{0.05, 1.0, 2, 1.1, 3.0},
			{0.05, 0.5, 1.0, 0.9, 3.0},
			{1, 0.8, 1.2, 1.0, 0},
			{0.001, 1.0, 10.0, 5.0, 2.0},
		}},
		"actions_per_round": {entries: []probEntry{
			{1, 1, 3, 1, 3},
			{0.2, 2, 4, 2, 8},
			{0.05, 1, 5, 2, 2},
			{1, 1, 5, 2, 0},
		}},
	},
	"better": {
		"deck_capacity":    {entries: []probEntry{{1, 5, 13, 8, 2}}},
		"reload_time":      {entries: []probEntry{{1, 5, 40, 20, 2}}},
		"fire_rate_wait":   {entries: []probEntry{{1, 1, 35, 5, 2}}},
		"spread_degrees":   {entries: []probEntry{{1, -1, 2, 0, 3}}},
		"speed_multiplier": {entries: []probEntry{{1, 0.8, 1.2, 1.0, 6.0}}},
		"actions_per_round": {entries: []probEntry{{1, 1, 3, 1, 3}}},
	},
}

// getGunProbs selects a prob entry for (wandType, variable), consuming one Next() call.
// Returns nil and does NOT consume RNG if the variable has no table.
func getGunProbs(wandType, variable string, p *NollaPrng) *probEntry {
	wt, ok := gunProbs[wandType]
	if !ok {
		return nil
	}
	tbl, ok := wt[variable]
	if !ok {
		return nil
	}
	e := tbl.selectEntry(p)
	return &e
}

// WandTypeData holds parameters for a wand tier.
type WandTypeData struct {
	wandType       string // "normal" or "better"
	cost           float64
	level          int
	forceUnshuffle bool
}

var wandTypes = map[string]WandTypeData{
	"wand_level_01":        {"normal", 30, 1, false},
	"wand_level_01_better": {"better", 30, 1, false},
	"wand_unshuffle_01":    {"normal", 25, 1, true},
	"wand_level_02":        {"normal", 40, 2, false},
	"wand_level_02_better": {"better", 40, 2, false},
	"wand_unshuffle_02":    {"normal", 40, 2, true},
	"wand_level_03":        {"normal", 60, 3, false},
	"wand_level_03_better": {"better", 60, 3, false},
	"wand_unshuffle_03":    {"normal", 60, 3, true},
	"wand_level_04":        {"normal", 80, 4, false},
	"wand_level_04_better": {"better", 80, 4, false},
	"wand_unshuffle_04":    {"normal", 80, 4, true},
	"wand_level_05":        {"normal", 100, 5, false},
	"wand_level_05_better": {"better", 100, 5, false},
	"wand_unshuffle_05":    {"normal", 100, 5, true},
	"wand_level_06":        {"normal", 120, 6, false},
	"wand_level_06_better": {"better", 120, 6, false},
	"wand_unshuffle_06":    {"normal", 120, 6, true},
	"wand_level_10":        {"normal", 200, 11, false},
	"wand_unshuffle_10":    {"normal", 180, 11, true},
}

var gunNames = []string{
	"Deadly", "Rusty", "Old", "New", "Shiny", "Lethal", "Dangerous", "Large", "Enormous",
	"Tiny", "Small", "Big", "Pretty", "Terrifying", "Confusing", "Mystery", "Superior",
	"Inferior", "Destructive", "Chaotic", "Lawful", "Good", "Bad", "Neutral", "Worn",
	"Polished", "Waxen", "Strong", "Weak", "Complex", "Tactical", "Horrifying", "Scary",
	"Scratched", "Untested", "Prototype", "Type a", "Type b", "Type x", "Secret", "Special",
	"Unique", "Mega", "Super", "Giga", "Turbo", "Hyper", "Alpha", "Omega", "Extreme",
	"Vanilla", "Flavourful", "Sturdy", "Solid", "Used", "Unused", "Grey", "Gray", "Sepia",
	"Secretly", "Actual", "Genuine", "Powerful", "Double", "Triple", "Stereo", "Ancient",
	"Antique", "Rustic", "Artisan", "Slick", "Slim", "Bulky", "Heavy", "Efficient", "Fast",
	"Quick", "Rapid", "Slow", "Veteran", "Agile", "Bitcoin", "Online",
}

// Potion material tables.
var potionMaterialsStandard = []string{
	"lava", "water", "blood", "alcohol", "oil", "slime", "acid",
	"radioactive_liquid", "gunpowder_unstable", "liquid_fire", "blood_cold",
}

var potionMaterialsMagic = []string{
	"magic_liquid_unstable_teleportation", "magic_liquid_polymorph",
	"magic_liquid_random_polymorph", "magic_liquid_berserk", "magic_liquid_charm",
	"magic_liquid_invisibility", "material_confusion", "magic_liquid_movement_faster",
	"magic_liquid_faster_levitation", "magic_liquid_worm_attractor",
	"magic_liquid_protection_all", "magic_liquid_mana_regeneration",
}

var potionMaterialsMagicNightmare = []string{
	"magic_liquid_unstable_teleportation", "magic_liquid_polymorph",
	"magic_liquid_random_polymorph", "magic_liquid_berserk", "magic_liquid_charm",
	"material_confusion", "magic_liquid_movement_faster", "magic_liquid_worm_attractor",
}

var potionMaterialsSecret = []string{
	"magic_liquid_hp_regeneration_unstable", "blood_worm", "gold", "snow", "glowshroom",
	"bush_seed", "cement", "salt", "sodium", "mushroom_seed", "plant_seed", "urine",
	"purifying_powder",
}

var potionLiquids = []string{
	"water", "water_temp", "water_ice", "water_swamp", "oil", "alcohol", "beer", "milk",
	"molut", "sima", "juhannussima", "magic_liquid", "material_confusion",
	"material_darkness", "material_rainbow", "magic_liquid_weakness",
	"magic_liquid_movement_faster", "magic_liquid_faster_levitation",
	"magic_liquid_faster_levitation_and_movement", "magic_liquid_worm_attractor",
	"magic_liquid_protection_all", "magic_liquid_mana_regeneration",
	"magic_liquid_unstable_teleportation", "magic_liquid_teleportation",
	"magic_liquid_hp_regeneration", "magic_liquid_hp_regeneration_unstable",
	"magic_liquid_polymorph", "magic_liquid_random_polymorph",
	"magic_liquid_unstable_polymorph", "magic_liquid_berserk", "magic_liquid_charm",
	"magic_liquid_invisibility", "cloud_radioactive", "cloud_blood", "cloud_slime",
	"swamp", "blood", "blood_fading", "blood_fungi", "blood_worm", "porridge",
	"blood_cold", "radioactive_liquid", "radioactive_liquid_fading", "plasma_fading",
	"gold_molten", "wax_molten", "silver_molten", "copper_molten", "brass_molten",
	"glass_molten", "glass_broken_molten", "steel_molten", "creepy_liquid", "cement",
	"slime", "slush", "vomit", "plastic_red_molten", "plastic_grey_molten",
	"acid", "lava", "urine", "rocket_particles", "peat", "plastic_prop_molten",
	"plastic_molten", "slime_yellow", "slime_green", "aluminium_oxide_molten",
	"steel_rust_molten", "metal_prop_molten", "aluminium_robot_molten",
	"aluminium_molten", "metal_nohit_molten", "metal_rust_molten", "metal_molten",
	"metal_sand_molten", "steelsmoke_static_molten", "steelmoss_static_molten",
	"steelmoss_slanted_molten", "steel_static_molten", "plasma_fading_bright",
	"radioactive_liquid_yellow", "cursed_liquid", "poison", "blood_fading_slow",
	"pus", "midas", "midas_precursor", "liquid_fire_weak", "liquid_fire",
	"just_death", "mimic_liquid", "void_liquid", "water_salt", "water_fading",
	"pea_soup",
}

var potionSands = []string{
	"mud", "concrete_sand", "sand", "bone", "soil", "sandstone", "fungisoil",
	"honey", "glue", "explosion_dirt", "snow", "snow_sticky", "rotten_meat",
	"meat_slime_sand", "rotten_meat_radioactive", "ice", "sand_herb", "wax",
	"gold", "silver", "copper", "brass", "diamond", "coal", "sulphur", "salt",
	"sodium_unstable", "gunpowder", "gunpowder_explosive", "gunpowder_tnt",
	"gunpowder_unstable", "gunpowder_unstable_big", "monster_powder_test",
	"rat_powder", "fungus_powder", "orb_powder", "gunpowder_unstable_boss_limbs",
	"plastic_red", "plastic_grey", "grass", "grass_holy", "grass_darker",
	"grass_ice", "grass_dry", "fungi", "spore", "moss", "plant_material",
	"plant_material_red", "plant_material_dark", "ceiling_plant_material",
	"mushroom_seed", "plant_seed", "mushroom", "mushroom_giant_red",
	"mushroom_giant_blue", "glowshroom", "bush_seed", "poo", "mammi",
	"glass_broken", "moss_rust", "fungi_creeping_secret", "fungi_creeping",
	"grass_dark", "fungi_green", "shock_powder", "fungus_powder_bad",
	"burning_powder", "purifying_powder", "sodium", "metal_sand", "steel_sand",
	"gold_radioactive", "endslime_blood", "sandstone_surface", "soil_dark",
	"soil_dead", "soil_lush_dark", "soil_lush", "sand_petrify", "lavasand",
	"sand_surface", "sand_blue", "plasma_fading_pink", "plasma_fading_green",
	"fungi_yellow",
}

// POWDER_MATERIALS_MAGIC from potion_config.js (weights are unused).
var powderMaterialsMagic = []string{
	"copper", "silver", "gold", "brass", "bone", "purifying_powder", "fungi",
}

// POWDER_MATERIALS_STANDARD from potion_config.js (weights are unused).
var powderMaterialsStandard = []string{
	"sand", "soil", "snow", "salt", "coal", "gunpowder", "fungisoil",
}
