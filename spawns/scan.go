package main

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

// coalmineColor is the coalmine region color in the static NG0 biome map.
const coalmineColor uint32 = 0xffd57917

// biomeSpawnFunctionMap ports BIOME_SPAWN_FUNCTION_MAP (default first, then
// biome-specific). Extended as more biomes are ported.
var biomeSpawnFunctionMap = map[string][]spawnFn{
	"general":  defaultSpawns,
	"coalmine": append(append([]spawnFn{}, defaultSpawns...), coalmineSpawns...),
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
			index := getSpawnFunctionIndex(biome, colorInt)
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
