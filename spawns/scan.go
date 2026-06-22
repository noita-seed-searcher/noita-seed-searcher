package main

import "sync"

// Port of the spawn-point scanner from noita-telescope/js/poi_scanner.js
// (prescanSpawnFunctions) + the color->function tables from
// spawn_function_config.js + tileToWorldCoordinates from utils.js.
//
// This enumerates every natural spawn point on a tile layer: for each
// spawn-function-colored pixel it records the function name and the world
// coordinates. Dispatching each point to a content generator (spawnSwitch) is a
// later stage; this is the "list all natural spawns" enumeration.

// spawnFn is one entry of a biome's ordered spawn-function table. The index of
// the entry in the biome's combined list is the spawnFunctionIndex.
type spawnFn struct {
	color        uint32
	funcName     string
	isPixelScene bool
	active       bool
}

// defaultSpawns ports DEFAULT_SPAWNS (spawn_function_config.js).
var defaultSpawns = []spawnFn{
	{0xff0000, "spawn_small_enemies", false, true},
	{0x800000, "spawn_big_enemies", false, true},
	{0x00ff00, "spawn_items", true, true},
	{0xc88d1a, "spawn_props", false, true},
	{0xc88000, "spawn_props2", false, true},
	{0xc80040, "spawn_props3", false, true},
	{0xffff00, "spawn_lamp", false, true},
	{0xff0aff, "load_pixel_scene", true, true},
	{0xFF0080, "load_pixel_scene2", true, true},
	{0xFF8000, "spawn_unique_enemy", false, true},
	{0xc84040, "spawn_unique_enemy2", false, true},
	{0x804040, "spawn_unique_enemy3", false, true},
	{0x96C850, "spawn_ghostlamp", false, true},
	{0x60A064, "spawn_candles", false, true},
	{0x50a000, "spawn_potion_altar", true, true},
	{0xbca0f0, "spawn_potions", false, true},
	{0x00FF5A, "spawn_apparition", false, true},
	{0x78FFFF, "spawn_heart", false, true},
	{0x50A0F0, "spawn_wands", false, true},
	{0xbf26a6, "spawn_portal", false, true},
	{0x04A977, "spawn_end_portal", false, true},
	{0xffd171, "spawn_orb", false, true},
	{0xffd181, "spawn_perk", false, true},
	{0xffff81, "spawn_all_perks", false, true},
	{0xc7eb28, "spawn_wand_trap", false, true},
	{0xE8FF80, "spawn_wand_trap_ignite", false, true},
	{0x2768DE, "spawn_wand_trap_electricity_source", false, true},
	{0x2768DF, "spawn_wand_trap_electricity", false, true},
	{0x6b4f9b, "spawn_moon", false, true},
	{0xd7b3e8, "spawn_collapse", false, true},
}

// coalmineSpawns ports COALMINE_SPAWNS (spawn_function_config.js).
var coalmineSpawns = []spawnFn{
	{0x0000ff, "spawn_nest", false, true},
	{0xB40000, "spawn_fungi", false, true},
	{0x969678, "load_structures", false, false},
	{0x967878, "load_large_structures", false, false},
	{0x967896, "load_i_structures", false, false},
	{0x80FF5A, "spawn_vines", false, false},
	{0xC35700, "load_oiltank", true, true},
	{0x55AF4B, "load_altar", true, true},
	{0x23B9C3, "spawn_altar_torch", false, false},
	{0x55AF8C, "spawn_skulls", false, false},
	{0x55FF8C, "spawn_chest", false, true},
	{0x4e175e, "load_oiltank_alt", true, true},
	{0x33934c, "spawn_shopitem", false, true},
	{0x50fafa, "spawn_trapwand", false, true},
	{0xf12ab5, "spawn_bbqbox", false, true},
	{0x005cfd, "spawn_swing_puzzle_box", false, false},
	{0x00b5fc, "spawn_swing_puzzle_target", false, true},
	{0x93ca00, "spawn_oiltank_puzzle", false, true},
	{0xb97300, "spawn_receptacle_oil", false, true},
}

// coalmineAltSpawns ports COALMINE_ALT_SPAWNS (spawn_function_config.js).
var coalmineAltSpawns = []spawnFn{
	{0x0000ff, "spawn_nest", false, true},
	{0xB40000, "spawn_fungi", false, true},
	{0x969678, "load_structures", false, false},
	{0x967878, "load_large_structures", false, false},
	{0x80FF5A, "spawn_vines", false, false},
	{0x33934c, "spawn_shopitem", false, true},
}

// excavationsiteSpawns ports EXCAVATIONSITE_SPAWNS (spawn_function_config.js).
var excavationsiteSpawns = []spawnFn{
	{0x0000ff, "spawn_nest", false, true},
	{0xFF50FF, "spawn_hanger", false, false},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x00ac6e, "load_pixel_scene4_alt", true, true},
	{0x0050FF, "spawn_wheel", false, false},
	{0x0150FF, "spawn_wheel_small", false, false},
	{0x0250FF, "spawn_wheel_tiny", false, false},
	{0x2d2eac, "spawn_rock", false, false},
	{0x0A50FF, "spawn_physicsstructure", false, false},
	{0xc999ff, "spawn_hanging_prop", false, false},
	{0x7868ff, "load_puzzleroom", true, true},
	{0x70d79e, "load_gunpowderpool_01", true, true},
	{0x70d79f, "load_gunpowderpool_02", true, true},
	{0x70d7a0, "load_gunpowderpool_03", true, true},
	{0x70d7a1, "load_gunpowderpool_04", true, true},
	{0x33934c, "spawn_shopitem", false, true},
	{0xb09016, "spawn_meditation_cube", false, true},
	{0x00855c, "spawn_receptacle", false, false},
	{0xb1ff99, "spawn_tower_short", false, false},
	{0x5c8550, "spawn_tower_tall", false, false},
	{0x227fff, "spawn_beam_low", false, false},
	{0x8228ff, "spawn_beam_low_flipped", false, false},
	{0x0098ba, "spawn_beam_steep", false, false},
	{0x7600a9, "spawn_beam_steep_flipped", false, false},
}

// snowcaveSpawns ports SNOWCAVE_SPAWNS (spawn_function_config.js).
var snowcaveSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x00AC33, "load_pixel_scene3", true, true},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x4691c7, "load_puzzle_capsule", true, true},
	{0x3691d7, "load_puzzle_capsule_b", true, true},
	{0x55AF4B, "load_altar", true, true},
	{0x23B9C3, "spawn_altar_torch", false, false},
	{0x55AF8C, "spawn_skulls", false, false},
	{0xF516E3, "spawn_scavenger_party", false, true},
	{0xFFC84E, "spawn_acid", false, false},
	{0x7285c4, "load_acidtank_right", true, true},
	{0x9472c4, "load_acidtank_left", true, true},
	{0x504600, "spawn_stones", false, false},
	{0xc800ff, "load_pixel_scene_alt", true, true},
	{0x33934c, "spawn_shopitem", false, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x434040, "spawn_burning_barrel", false, false},
	{0xb4a00a, "spawn_fish", false, true},
	{0xaa42ff, "spawn_electricity_trap", false, false},
	{0x366178, "spawn_buried_eye_teleporter", false, true},
	{0x876543, "spawn_statue_hand", false, false},
	{0x00855c, "spawn_receptacle", false, false},
}

// snowcastleSpawns ports SNOWCASTLE_SPAWNS (spawn_function_config.js).
var snowcastleSpawns = []spawnFn{
	{0xC8C800, "spawn_lamp2", false, false},
	{0x01a1fa, "spawn_turret", false, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0xc78f20, "spawn_barricade", false, false},
	{0xc022f5, "spawn_forcefield_generator", false, false},
	{0xa3d900, "spawn_brimstone", false, true},
	{0x00d982, "spawn_vasta_or_vihta", false, true},
	{0x932020, "spawn_cook", false, true},
	{0x614630, "load_panel_01", false, false},
	{0x614635, "load_panel_02", false, false},
	{0x61463e, "load_panel_03", false, false},
	{0x614638, "load_panel_04", false, false},
	{0x614646, "load_panel_07", false, false},
	{0x614650, "load_panel_08", false, false},
	{0x614658, "load_panel_09", false, false},
	{0xc133ff, "load_chamfer_top_r", true, true},
	{0x8b33ff, "load_chamfer_top_l", true, true},
	{0x8824b3, "load_chamfer_bottom_r", true, true},
	{0x5f23ad, "load_chamfer_bottom_l", true, true},
	{0x73ffa7, "load_chamfer_inner_top_r", true, true},
	{0xd5ff7f, "load_chamfer_inner_top_l", true, true},
	{0x387d51, "load_chamfer_inner_bottom_r", true, true},
	{0x97b55b, "load_chamfer_inner_bottom_l", true, true},
	{0x44609c, "load_pillar_filler", true, true},
	{0x44449c, "load_pillar_filler_tall", true, true},
	{0xb03058, "load_pod_large", true, true},
	{0xb05830, "load_pod_small_l", true, true},
	{0xb09030, "load_pod_small_r", true, true},
	{0xffa659, "load_furniture", false, false},
	{0xfec390, "load_furniture_bunk", false, false},
	{0x4c63e0, "spawn_root_grower", false, false},
	{0x4cacab, "spawn_forge_check", false, false},
	{0x2a78ff, "spawn_drill_laser", false, false},
}

// rainforestSpawns ports RAINFOREST_SPAWNS (spawn_function_config.js).
var rainforestSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x400000, "spawn_scavengers", false, true},
	{0x400080, "spawn_large_enemies", false, true},
	{0xC8C800, "spawn_lamp2", false, false},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x943030, "spawn_dragonspot", false, true},
	{0x4c63e0, "spawn_root_grower", false, false},
	{0x806326, "spawn_tree", false, false},
}

// rainforestOpenSpawns ports RAINFOREST_OPEN_SPAWNS (spawn_function_config.js).
var rainforestOpenSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x400000, "spawn_scavengers", false, true},
	{0x400080, "spawn_large_enemies", false, true},
	{0xC8C800, "spawn_lamp2", false, false},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x943030, "spawn_dragonspot", false, true},
	{0x4c63e0, "spawn_root_grower", false, false},
	{0x806326, "spawn_tree", false, false},
}

// rainforestDarkSpawns ports RAINFOREST_DARK_SPAWNS (spawn_function_config.js).
var rainforestDarkSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x400000, "spawn_scavengers", false, true},
	{0x400080, "spawn_large_enemies", false, true},
	{0xC8C800, "spawn_lamp2", false, false},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x943030, "spawn_dragonspot", false, true},
	{0x4c63e0, "spawn_root_grower", false, false},
	{0x806326, "spawn_tree", false, false},
}

// vaultSpawns ports VAULT_SPAWNS (spawn_function_config.js).
var vaultSpawns = []spawnFn{
	{0x692e94, "load_pixel_scene_wide", true, true},
	{0x822e5b, "load_pixel_scene_tall", true, true},
	{0x00AC64, "load_warning_strip", false, false},
	{0x01a1fa, "spawn_turret", false, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x504B64, "spawn_machines", false, false},
	{0xc999ff, "spawn_hanging_prop", false, false},
	{0xBE8246, "spawn_pipes_hor", true, true},
	{0xBE8264, "spawn_pipes_turn_right", true, true},
	{0xBE8282, "spawn_pipes_turn_left", true, true},
	{0xBE82A0, "spawn_pipes_ver", true, true},
	{0xBE82BE, "spawn_pipes_cross", true, true},
	{0x2E8246, "spawn_pipes_big_hor", true, true},
	{0x2E8264, "spawn_pipes_big_turn_right", true, true},
	{0x2E8282, "spawn_pipes_big_turn_left", true, true},
	{0x2E82A0, "spawn_pipes_big_ver", true, true},
	{0x5c73da, "spawn_stains", true, true},
	{0x5c73db, "spawn_stains_ceiling", true, true},
	{0xc78f20, "spawn_barricade", false, false},
	{0x4a107d, "load_pillar", false, false},
	{0x7b59ab, "load_pillar_base", false, false},
	{0x40ffce, "load_catwalk", true, true},
	{0xbf4c86, "spawn_apparatus", false, false},
	{0xaa42ff, "spawn_electricity_trap", false, false},
	{0x33934c, "spawn_shopitem", false, true},
	{0xacf14b, "spawn_laser_trap", true, true},
	{0xa45aff, "spawn_lab_puzzle", false, true},
}

// vaultFrozenSpawns ports VAULT_FROZEN_SPAWNS (spawn_function_config.js).
var vaultFrozenSpawns = []spawnFn{
	{0x400000, "spawn_robots", false, true},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x01a1fa, "spawn_turret", false, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x504B64, "spawn_machines", false, false},
	{0xBE8246, "spawn_pipes_hor", true, true},
	{0xBE8264, "spawn_pipes_turn_right", true, true},
	{0xBE8282, "spawn_pipes_turn_left", true, true},
	{0xBE82A0, "spawn_pipes_ver", true, true},
	{0xBE82BE, "spawn_pipes_cross", true, true},
	{0xc78f20, "spawn_barricade", false, false},
}

// cryptSpawns ports CRYPT_SPAWNS (spawn_function_config.js).
var cryptSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x808000, "spawn_statues", false, false},
	{0x00AC33, "load_pixel_scene3", true, true},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x97ab00, "load_pixel_scene5", true, true},
	{0xc9d959, "load_pixel_scene5b", true, true},
	{0xC8C800, "spawn_lamp2", false, false},
	{0x400080, "spawn_large_enemies", false, true},
	{0xC8001A, "spawn_ghost_crystal", false, true},
	{0x82FF5A, "spawn_crawlers", false, false},
	{0x647D7D, "spawn_pressureplates", false, false},
	{0x649B7D, "spawn_doors", false, false},
	{0xA07864, "spawn_scavengers", false, true},
	{0xFFCD2A, "spawn_scorpions", false, true},
	{0x2D1E5A, "spawn_bones", false, false},
	{0x782060, "load_beam", true, true},
	{0x783060, "load_background_scene", false, false},
	{0x378ec4, "load_small_background_scene", false, false},
	{0x786460, "load_cavein", true, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x535988, "spawn_statue_back", false, false},
	{0x33934c, "spawn_shopitem", false, true},
}

// pyramidSpawns ports PYRAMID_SPAWNS (spawn_function_config.js).
var pyramidSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x808000, "spawn_statues", false, false},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0xC8C800, "spawn_lamp2", false, false},
	{0x400080, "spawn_large_enemies", false, true},
	{0xC8001A, "spawn_ghost_crystal", false, true},
	{0x82FF5A, "spawn_crawlers", false, false},
	{0x647D7D, "spawn_pressureplates", false, false},
	{0x649B7D, "spawn_doors", false, false},
	{0xA07864, "spawn_scavengers", false, true},
	{0x00AC33, "load_pixel_scene3", true, true},
	{0xFFCD2A, "spawn_scorpions", false, true},
	{0x905ecb, "spawn_reward_wands", false, false},
	{0x905ecc, "spawn_boss_limbs_trigger", false, false},
}

// fungicaveSpawns ports FUNGICAVE_SPAWNS (spawn_function_config.js).
var fungicaveSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x400000, "spawn_robots", false, true},
	{0x0000ff, "spawn_nest", false, true},
	{0x30b3b0, "spawn_physics_fungus", false, false},
}

// fungiforestSpawns ports FUNGIFOREST_SPAWNS (spawn_function_config.js).
var fungiforestSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x0000ff, "spawn_nest", false, true},
	{0x30b3b0, "spawn_physics_fungus", false, false},
	{0x30b3f0, "spawn_physics_acid_fungus", false, false},
	{0x80FF5A, "spawn_vines", false, false},
	{0x6a8d79, "spawn_fungitrap", false, false},
}

// wandcaveSpawns ports WANDCAVE_SPAWNS (spawn_function_config.js).
var wandcaveSpawns = []spawnFn{
	{0x805000, "spawn_cloud_trap", false, false},
	{0x397780, "load_floor_rubble", false, false},
	{0x00ffa0, "load_floor_rubble_l", false, false},
	{0x1ca7ff, "load_floor_rubble_r", false, false},
}

// wizardcaveSpawns ports WIZARDCAVE_SPAWNS (spawn_function_config.js).
var wizardcaveSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x808000, "spawn_statues", false, false},
	{0x00AC33, "load_pixel_scene3", true, true},
	{0x00AC64, "load_pixel_scene4", true, true},
	{0x97ab00, "load_pixel_scene5", true, true},
	{0xc9d959, "load_pixel_scene5b", true, true},
	{0xC8C800, "spawn_lamp2", false, false},
	{0x400080, "spawn_large_enemies", false, true},
	{0xC8001A, "spawn_ghost_crystal", false, true},
	{0x82FF5A, "spawn_crawlers", false, false},
	{0x647D7D, "spawn_pressureplates", false, false},
	{0x649B7D, "spawn_doors", false, false},
	{0xA07864, "spawn_scavengers", false, true},
	{0xFFCD2A, "spawn_scorpions", false, true},
	{0x2D1E5A, "spawn_bones", false, false},
	{0x782060, "load_beam", true, true},
	{0x783060, "load_background_scene", false, false},
	{0x378ec4, "load_small_background_scene", false, false},
	{0x786460, "load_cavein", true, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x535988, "spawn_statue_back", false, false},
	{0x33934c, "spawn_shopitem", false, true},
}

// liquidcaveSpawns ports LIQUIDCAVE_SPAWNS (spawn_function_config.js).
var liquidcaveSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x00AC64, "load_background_panel_big", false, false},
	{0x967878, "spawn_lasergun", false, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0xc88dab, "spawn_statues", false, true},
}

// robobaseSpawns ports ROBOBASE_SPAWNS (spawn_function_config.js).
var robobaseSpawns = []spawnFn{
	{0x00AC64, "load_warning_strip", false, false},
	{0x01a1fa, "spawn_turret", false, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0xc999ff, "spawn_hanging_prop", false, false},
	{0xc78f20, "spawn_barricade", false, false},
	{0x33934c, "spawn_shopitem", false, true},
	{0x39a760, "spawn_lasergate_ver", false, false},
}

// sandcaveSpawns ports SANDCAVE_SPAWNS (spawn_function_config.js).
var sandcaveSpawns = []spawnFn{
	{0xC8C800, "spawn_lamp2", false, false},
	{0xDC0060, "spawn_props4", false, false},
}

// snowchasmSpawns ports SNOWCHASM_SPAWNS (spawn_function_config.js).
var snowchasmSpawns = []spawnFn{
	{0x33934c, "spawn_shopitem", false, true},
	{0xd0d0b4, "spawn_treasure", false, true},
	{0x41704d, "spawn_specialshop", false, true},
	{0x235a15, "spawn_music_machine", false, true},
	{0xffeedd, "init", false, false},
}

// theEndSpawns ports THE_END_SPAWNS (spawn_function_config.js).
var theEndSpawns = []spawnFn{
	{0x33934c, "spawn_shopitem", false, true},
	{0xbe704d, "spawn_specialshop", false, true},
}

// theSkySpawns ports THE_SKY_SPAWNS (spawn_function_config.js).
var theSkySpawns = []spawnFn{
	{0x33934c, "spawn_shopitem", false, true},
	{0xbe704d, "spawn_specialshop", false, true},
}

// meatSpawns ports MEAT_SPAWNS (spawn_function_config.js).
var meatSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x55AF8C, "spawn_skulls", false, false},
	{0x4c63e1, "spawn_cyst", false, false},
	{0x80FF5A, "spawn_vines", false, false},
	{0xd97f7f, "spawn_mouth", false, true},
	{0xc999ff, "spawn_hanging_prop", false, false},
}

// towerSpawns ports TOWER_SPAWNS (spawn_function_config.js).
var towerSpawns = []spawnFn{
	{0x0000ff, "spawn_nest", false, true},
	{0xB40000, "spawn_fungi", false, true},
	{0x80FF5A, "spawn_vines", false, false},
	{0x55AF8C, "spawn_skulls", false, false},
	{0x55FF8C, "spawn_chest", false, true},
	{0x33934c, "spawn_shopitem", false, true},
}

// snowcastleCavernSpawns ports SNOWCASTLE_CAVERN_SPAWNS (spawn_function_config.js).
var snowcastleCavernSpawns = []spawnFn{
	{0xC8C800, "spawn_lamp", false, false},
	{0x80FF5A, "spawn_vines", false, false},
	{0x33934c, "spawn_shopitem", false, true},
	{0xffeedd, "init", false, false},
	{0x03deaf, "spawn_fish", false, true},
	{0xff2974, "spawn_hourglass_blood", false, false},
	{0xff9122, "spawn_hourglass_master", false, false},
	{0x216bff, "spawn_hourglass_music_trigger", false, false},
}

// excavationsiteCubeChamberSpawns ports EXCAVATIONSITE_CUBE_CHAMBER_SPAWNS (spawn_function_config.js).
var excavationsiteCubeChamberSpawns = []spawnFn{
	{0x55AF8C, "spawn_skulls", false, false},
	{0xffeedd, "init", false, false},
	{0x366178, "spawn_teleporter", false, false},
}

// robotEggSpawns ports ROBOT_EGG_SPAWNS (spawn_function_config.js).
var robotEggSpawns = []spawnFn{
	{0xffeedd, "init", false, false},
	{0x548f77, "spawn_teleport", false, false},
	{0x709615, "spawn_chest", false, false},
}

// templesCommonSpawns ports TEMPLES_COMMON_SPAWNS (spawn_function_config.js).
var templesCommonSpawns = []spawnFn{
	{0x805000, "spawn_cloud_trap", false, false},
	{0x397780, "load_floor_rubble", false, false},
	{0x00ffa0, "load_floor_rubble_l", false, false},
	{0x1ca7ff, "load_floor_rubble_r", false, false},
	{0xffeed1, "spawn_puzzle_watchtower", false, false},
	{0xffeeda, "spawn_puzzle_barren", false, false},
	{0xffeedb, "spawn_puzzle_potion_mimics", false, false},
	{0xffeedc, "spawn_puzzle_darkness", false, false},
	{0xffeedd, "spawn_boss", false, true},
	{0xffeede, "spawn_potion_mimic_empty", false, true},
	{0xffeedf, "spawn_potion_mimic", false, true},
	{0xffeed0, "spawn_fish_many", false, false},
	{0xffeed2, "spawn_boss_phase2_marker", false, true},
	{0xffeed3, "spawn_book_barren", false, false},
	{0xffeed4, "spawn_potion_beer", false, true},
	{0xffeed5, "spawn_potion_milk", false, true},
	{0xffeed6, "spawn_scorpion", false, true},
	{0xffaaaa, "spawn_sign_left", false, false},
	{0xffaadd, "spawn_sign_right", false, false},
}

// watchtowerSpawns ports WATCHTOWER_SPAWNS (spawn_function_config.js).
var watchtowerSpawns = []spawnFn{
	{0xaaff00, "spawn_small_enemies2", false, true},
	{0xffaa00, "spawn_big_enemies2", false, true},
}

// concatSpawns is a helper to concatenate spawn function tables.
func concatSpawns(slices ...[]spawnFn) []spawnFn {
	var out []spawnFn
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

// biomeSpawnFunctionMap ports BIOME_SPAWN_FUNCTION_MAP (default first, then
// biome-specific). Extended as more biomes are ported.
var biomeSpawnFunctionMap = map[string][]spawnFn{
	"general":                     concatSpawns(defaultSpawns),
	"coalmine":                    concatSpawns(defaultSpawns, coalmineSpawns),
	"coalmine_alt":                concatSpawns(defaultSpawns, coalmineAltSpawns),
	"excavationsite":              concatSpawns(defaultSpawns, excavationsiteSpawns),
	"snowcave":                    concatSpawns(defaultSpawns, snowcaveSpawns),
	"snowcastle":                  concatSpawns(defaultSpawns, snowcastleSpawns),
	"rainforest":                  concatSpawns(defaultSpawns, rainforestSpawns),
	"rainforest_open":             concatSpawns(defaultSpawns, rainforestOpenSpawns),
	"rainforest_dark":             concatSpawns(defaultSpawns, rainforestDarkSpawns),
	"vault":                       concatSpawns(defaultSpawns, vaultSpawns),
	"vault_frozen":                concatSpawns(defaultSpawns, vaultFrozenSpawns),
	"crypt":                       concatSpawns(defaultSpawns, cryptSpawns),
	"pyramid":                     concatSpawns(defaultSpawns, pyramidSpawns),
	"fungicave":                   concatSpawns(defaultSpawns, fungicaveSpawns),
	"fungiforest":                 concatSpawns(defaultSpawns, fungiforestSpawns),
	"wandcave":                    concatSpawns(defaultSpawns, wandcaveSpawns),
	"wizardcave":                  concatSpawns(defaultSpawns, wizardcaveSpawns),
	"liquidcave":                  concatSpawns(defaultSpawns, liquidcaveSpawns),
	"robobase":                    concatSpawns(defaultSpawns, robobaseSpawns),
	"sandcave":                    concatSpawns(defaultSpawns, sandcaveSpawns),
	"winter_caves":                concatSpawns(defaultSpawns, snowchasmSpawns),
	"clouds":                      concatSpawns(defaultSpawns),
	"the_sky":                     concatSpawns(defaultSpawns, theSkySpawns),
	"the_end":                     concatSpawns(defaultSpawns, theEndSpawns),
	"temple_altar":                concatSpawns(defaultSpawns),
	"meat":                        concatSpawns(defaultSpawns, meatSpawns),
	"lake_deep":                   concatSpawns(defaultSpawns),
	"solid_wall_tower_1":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_2":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_3":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_4":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_5":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_6":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_7":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_8":          concatSpawns(defaultSpawns, towerSpawns),
	"solid_wall_tower_9":          concatSpawns(defaultSpawns, towerSpawns),
	"snowcastle_cavern":           concatSpawns(defaultSpawns, snowcastleCavernSpawns),
	"excavationsite_cube_chamber": concatSpawns(defaultSpawns, excavationsiteCubeChamberSpawns),
	"robot_egg":                   concatSpawns(defaultSpawns, robotEggSpawns),
	"biome_watchtower":            concatSpawns(defaultSpawns, templesCommonSpawns, watchtowerSpawns),
	"biome_barren":                concatSpawns(defaultSpawns, templesCommonSpawns),
	"biome_potion_mimics":         concatSpawns(defaultSpawns, templesCommonSpawns),
	"biome_darkness":              concatSpawns(defaultSpawns, templesCommonSpawns),
	"biome_boss_sky":              concatSpawns(defaultSpawns, templesCommonSpawns),
}

// getSpawnFunctionIndex ports getSpawnFunctionIndex: first table index whose
// color matches, or -1.
func getSpawnFunctionIndex(biomeName string, color uint32) int {
	fns, ok := biomeSpawnFunctionMap[biomeName]
	if !ok {
		return -1
	}
	for i := range fns {
		if fns[i].color == color {
			return i
		}
	}
	return -1
}

// spawnColorIndex accelerates the per-pixel spawn-color lookup that
// prescanSpawnFunctions does for every non-trivial pixel of every generated
// tile. A biome's fns table can hold ~50 entries, so the original linear scan
// was a top search cost. present is a 2^24-bit membership set over 0xRRGGBB
// colors giving O(1) rejection (the overwhelmingly common case); firstIdx
// resolves the rare matches to the first table index, preserving
// getSpawnFunctionIndex's first-match semantics.
type spawnColorIndex struct {
	present  []uint64
	firstIdx map[uint32]int
}

func (s *spawnColorIndex) lookup(color uint32) int {
	if s.present[color>>6]&(uint64(1)<<(color&63)) == 0 {
		return -1
	}
	if i, ok := s.firstIdx[color]; ok {
		return i
	}
	return -1
}

var spawnColorIndexCache sync.Map // biome name -> *spawnColorIndex

// spawnColorIndexFor returns the cached membership index for a biome's fns
// table, building it once on first use.
func spawnColorIndexFor(biome string, fns []spawnFn) *spawnColorIndex {
	if v, ok := spawnColorIndexCache.Load(biome); ok {
		return v.(*spawnColorIndex)
	}
	sc := &spawnColorIndex{
		present:  make([]uint64, (1<<24)/64),
		firstIdx: make(map[uint32]int, len(fns)),
	}
	for i := range fns {
		c := fns[i].color & 0xffffff
		if _, ok := sc.firstIdx[c]; !ok {
			sc.firstIdx[c] = i
		}
		sc.present[c>>6] |= uint64(1) << (c & 63)
	}
	actual, _ := spawnColorIndexCache.LoadOrStore(biome, sc)
	return actual.(*spawnColorIndex)
}

const (
	worldChunkCenterX    = 35
	worldChunkCenterXNGP = 32
	worldChunkCenterY    = 14
	chunkSize            = 512
	tileSize             = 10
	tileOffsetX          = 5
	tileOffsetY          = -13
)

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// tileToWorldCoordinates ports utils.js tileToWorldCoordinates.
func tileToWorldCoordinates(chunkBaseX, chunkBaseY, tileX, tileY, pw, pwVertical int, isNGP bool, gameMode string) (int, int) {
	ngpLike := isNGP || gameMode == "nightmare"
	wccx := worldChunkCenterX
	if ngpLike {
		wccx = worldChunkCenterXNGP
	}
	worldSize := 70 * chunkSize
	if ngpLike {
		worldSize = 64*chunkSize - 8
	}
	smallChunk := chunkSize / tileSize // 51

	div5offX := 5 * chunkSize * floorDiv(chunkBaseX-wccx, 5)
	mod5offX := (((chunkBaseX - wccx) % 5) + 5) % 5
	worldBaseX := div5offX + mod5offX*smallChunk*tileSize
	worldX := -tileSize + worldBaseX + tileX*tileSize + tileOffsetX

	div5offY := 5 * chunkSize * floorDiv(chunkBaseY-worldChunkCenterY, 5)
	mod5offY := (((chunkBaseY - worldChunkCenterY) % 5) + 5) % 5
	worldBaseY := div5offY + mod5offY*smallChunk*tileSize
	if mod5offY > 0 {
		worldBaseY += tileSize
	}
	worldY := -tileSize + worldBaseY + tileY*tileSize + tileOffsetY

	if ngpLike && mod5offX >= 3 {
		worldX += tileSize
	}
	worldY += tileSize
	if ngpLike {
		worldX -= 4
	}
	worldX += pw * worldSize
	worldY += pwVertical * 24570
	return worldX, worldY
}

// detectedSpawn is one natural spawn point: the function to run and where.
type detectedSpawn struct {
	sourceBiome string
	funcName    string
	index       int
	x, y        int // world coordinates (PW0)
}

// prescanSpawnFunctions ports poi_scanner.js prescanSpawnFunctions for a single
// layer: scan the tile buffer for spawn-function colors and map each to world
// coordinates.
func prescanSpawnFunctions(layer *tileLayer, isNGP bool, gameMode string) []detectedSpawn {
	var detected []detectedSpawn
	biome := layer.biomeName
	width := layer.width
	height := layer.mapH
	fns, ok := biomeSpawnFunctionMap[biome]
	if !ok || len(fns) == 0 {
		return detected
	}
	sci := spawnColorIndexFor(biome, fns)
	detected = make([]detectedSpawn, 0, 64)

	for y := 4; y < height+4; y++ {
		for x := 0; x < width; x++ {
			srcIdx := (y*width + x) * 3
			r := layer.buffer[srcIdx]
			g := layer.buffer[srcIdx+1]
			b := layer.buffer[srcIdx+2]
			colorInt := (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
			if colorInt == 0x000000 || colorInt == 0xffffff {
				continue
			}
			// O(1) membership reject for the common terrain pixel; the linear
			// fns scan here was a top search cost (~50 entries/biome).
			index := sci.lookup(colorInt)
			if index >= 0 {
				wx, wy := tileToWorldCoordinates(layer.minX, layer.minY, x, y-4, 0, 0, isNGP, gameMode)
				detected = append(detected, detectedSpawn{
					sourceBiome: biome,
					funcName:    fns[index].funcName,
					index:       index,
					x:           wx,
					y:           wy,
				})
			}
		}
	}
	return detected
}
