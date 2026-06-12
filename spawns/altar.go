package main

import "math"

// altarData holds spawn configuration for a biome's altar.
type altarData struct {
	r0, rl, rg, ru float64
	x, y           float64 // wand generation position offsets
	xoff, yoff     float64 // RNG check position offsets
	tx, ty         float64 // alternative check offsets (overrides xoff/yoff if nonzero)
	gx, gy         float64 // global position adjustment
	rp             float64 // potion spawn threshold
	px, py         float64 // potion item position offsets
	hasTX          bool    // whether tx/ty are set
	hasR0          bool    // whether r0 check is active
	hasRl          bool    // whether rl check is active
	hasRg          bool    // whether rg check is active
	hasRu          bool    // whether ru check is active
}

var altarSpawnData = map[string]altarData{
	"coalmine":      {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"coalmine_alt":  {hasR0: true, r0: 0.47, hasRl: true, rl: 0.725, x: 0, y: -14, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"excavationsite":{hasRl: true, rl: 0.725, x: 0, y: -14, xoff: -11.431, yoff: 10.5257},
	"snowcave":      {hasRg: true, rg: 0.45, x: -5, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"snowcastle":    {hasRl: true, rl: 0.2, x: 5, y: -5, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 6, py: -3},
	"rainforest":    {hasRl: true, rl: 0.27, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"rainforest_open":{hasRl: true, rl: 0.27, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"rainforest_dark":{hasRl: true, rl: 0.27, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"vault":         {hasRg: true, rg: 0.93, x: 5, y: -6, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 6, py: -4},
	"crypt":         {hasRl: true, rl: 0.38, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"fungicave":     {hasRl: true, rl: 0.06, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, px: 0, py: -6},
	"fungiforest":   {hasRl: true, rl: 0.06, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, px: 0, py: -6},
	"wizardcave":    {hasRl: true, rl: 0.38, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"liquidcave":    {hasRg: true, rg: 0.0, x: 0, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"sandcave":      {hasRg: true, rg: 0.94, x: 5, y: -5, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 6, py: -3},
	"robobase":      {hasRg: true, rg: 0.93, x: 5, y: -6, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 6, py: -4},
	"vault_frozen":  {hasRg: true, rg: 0.83, hasRu: true, ru: 0.93, x: 5, y: -6, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 6, py: -4},
	"meat":          {hasRg: true, rg: 0.3, hasRu: true, ru: 0.55, x: -5, y: -14, xoff: -11.631, yoff: 10.2257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_1": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_2": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_3": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.01, px: 5, py: -4},
	"solid_wall_tower_4": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_5": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_6": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_7": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_8": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"solid_wall_tower_9": {hasR0: true, r0: 0.47, hasRl: true, rl: 0.755, x: 5, y: -9, xoff: -11.431, yoff: 10.5257, rp: 0.65, px: 5, py: -4},
	"excavationsite_cube_chamber": {hasRg: true, rg: 1.0, x: 0, y: 0},
	"snowcave_secret_chamber":     {hasRg: true, rg: 1.0, x: 0, y: 0},
	"robot_egg":                   {hasRg: true, rg: 1.0, x: 0, y: 0},
}

type wandSpawnEntry struct {
	types   []string
	weights []float64
}

var wandSpawnData = map[string]wandSpawnEntry{
	"coalmine": {
		types:   []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","premade_10","premade_11","premade_12","premade_13","premade_14","premade_15","premade_16","premade_17","wand_level_01"},
		weights: []float64{1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1.9},
	},
	"coalmine_alt": {
		types:   []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9"},
		weights: []float64{1,1,1,1,1,1,1,1,1},
	},
	"excavationsite": {types: []string{"wand_unshuffle_01","wand_level_02","wand_level_02_better"}, weights: []float64{2,2,2}},
	"snowcave":        {types: []string{"wand_level_02","wand_level_02_better","wand_unshuffle_02"}, weights: []float64{5,5,5}},
	"snowcastle":      {types: []string{"wand_level_03","wand_level_03_better","wand_unshuffle_03"}, weights: []float64{5,5,5}},
	"rainforest":      {types: []string{"wand_level_04","wand_level_05","wand_unshuffle_02","wand_unshuffle_03","wand_level_04_better"}, weights: []float64{5,3,3,3,5}},
	"rainforest_open": {types: []string{"wand_level_04","wand_level_05","wand_unshuffle_02","wand_unshuffle_03","wand_level_04_better"}, weights: []float64{5,3,3,3,5}},
	"vault":           {types: []string{"wand_level_05","wand_level_05_better","wand_unshuffle_03","wand_unshuffle_04"}, weights: []float64{5,5,3,2}},
	"crypt":           {types: []string{"wand_level_06","wand_level_06_better","wand_unshuffle_05","wand_unshuffle_06"}, weights: []float64{5,5,3,2}},
	"fungicave":       {types: []string{"wand_unshuffle_02","wand_unshuffle_01"}, weights: []float64{0.5,0.5}},
	"fungiforest":     {types: []string{"wand_unshuffle_03","wand_unshuffle_04","wand_level_05_better"}, weights: []float64{5,5,5}},
	"rainforest_dark": {types: []string{"wand_level_04","wand_level_05","wand_unshuffle_03","wand_unshuffle_04","wand_level_05_better"}, weights: []float64{5,3,3,3,5}},
	"wizardcave":      {types: []string{"wand_level_06","wand_level_06_better","wand_unshuffle_05","wand_unshuffle_06"}, weights: []float64{5,5,3,2}},
	"liquidcave":      {types: []string{}, weights: []float64{}},
	"sandcave":        {types: []string{"wand_level_04","wand_unshuffle_02"}, weights: []float64{5,5}},
	"robobase":        {types: []string{"wand_level_05","wand_level_05_better","wand_unshuffle_03","wand_unshuffle_04"}, weights: []float64{5,5,3,2}},
	"vault_frozen":    {types: []string{"wand_level_05","wand_unshuffle_03","wand_unshuffle_04"}, weights: []float64{5,3,2}},
	"meat":            {types: []string{"wand_level_05","wand_level_06","wand_unshuffle_04","wand_unshuffle_05"}, weights: []float64{5,5,5,5}},
	"solid_wall_tower_1": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_2": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_3": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_4": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_5": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_6": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_7": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_8": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"solid_wall_tower_9": {types: []string{"premade_1","premade_2","premade_3","premade_4","premade_5","premade_6","premade_7","premade_8","premade_9","wand_level_01"}, weights: []float64{1,1,1,1,1,1,1,1,1,1}},
	"excavationsite_cube_chamber": {types: []string{"wand_level_03","wand_unshuffle_02"}, weights: []float64{0.5,0.5}},
	"snowcave_secret_chamber":     {types: []string{"wand_level_03","wand_unshuffle_02"}, weights: []float64{0.5,0.5}},
	"robot_egg":                   {types: []string{"wand_level_05","wand_level_05_better","wand_unshuffle_03","wand_unshuffle_04"}, weights: []float64{5,5,3,2}},
}

// getWandType mirrors getWandType() from wand_generation.js.
// Uses ProceduralRandom(ws+ng, x-5, y) for weighted selection.
func getWandType(ws uint32, ng int, x, y float64, biome string) string {
	entry, ok := wandSpawnData[biome]
	if !ok || len(entry.types) == 0 {
		return ""
	}
	var totalProb float64
	for _, w := range entry.weights {
		totalProb += w
	}
	p := newPrng(0)
	r := p.proceduralRandom(ws+uint32(ng), x-5, y) * totalProb
	for i, w := range entry.weights {
		if r <= w {
			return entry.types[i]
		}
		r -= w
	}
	return entry.types[len(entry.types)-1]
}

// SpawnWandAltar mirrors spawnWandAltar() from wand_generation.js.
// Returns nil if no wand spawns at this position.
func SpawnWandAltar(ws uint32, ng int, x, y float64, biome string, noMoreShuffle bool) *Item {
	ad, ok := altarSpawnData[biome]
	if !ok {
		return nil
	}
	x += ad.gx
	y += ad.gy

	p := newPrng(0)

	if ad.hasR0 {
		r0 := p.proceduralRandom(ws+uint32(ng), x, y)
		if r0 < ad.r0 {
			return nil
		}
	}

	var tx, ty float64
	if ad.hasTX {
		tx = x + ad.tx
		ty = y + ad.ty
	} else {
		tx = x + ad.xoff
		ty = y + ad.yoff
	}
	r := p.proceduralRandom(ws+uint32(ng), tx, ty)

	if ad.hasRl && r < ad.rl {
		return nil
	}
	if ad.hasRg && r >= ad.rg {
		// utility box or no spawn
		return nil
	}

	wandX := math.Floor(x + ad.x)
	wandY := math.Floor(y + ad.y)
	typeName := getWandType(ws, ng, wandX, wandY, biome)
	if typeName == "" {
		return nil
	}

	// Wand is generated 5px below the type-selection position
	genY := wandY + 5
	wand := generateWandByType(ws, ng, wandX, genY, typeName, noMoreShuffle)
	if wand == nil {
		return nil
	}
	wand.X = wandX
	wand.Y = genY
	return &Item{ItemType: "wand", Wand: wand, X: wandX, Y: genY}
}

// SpawnPotionAltar mirrors spawnPotionAltar() from potion_generation.js.
func SpawnPotionAltar(ws uint32, ng int, x, y float64, biome, gameMode string, greedCurse bool) *Item {
	ad, ok := altarSpawnData[biome]
	if !ok {
		return nil
	}
	x += ad.gx
	y += ad.gy

	p := newPrng(0)
	r := p.proceduralRandom(ws+uint32(ng), x, y)
	rp := ad.rp
	if rp == 0 {
		rp = 0.65
	}
	if r < rp {
		return nil
	}

	itemX := x + ad.px
	itemY := y + ad.py
	var item *Item
	if biome == "liquidcave" {
		item = generateItemLiquidcave(ws, ng, itemX, itemY)
	} else {
		item = generateItem(ws, ng, itemX, itemY, greedCurse)
	}
	if item != nil {
		item.X = itemX
		item.Y = itemY
	}
	return item
}

// SpawnWand mirrors spawnWand() from wand_generation.js.
// x, y are the DISPLAYED wand position (wand.x/wand.y as shown in noita-telescope).
// In the JS, spawnWand is called at (x, y-5), then generateWand at (x, y).
// getWandType is passed (x, y-5) so it uses ProceduralRandom at (x-5, y-5).
func SpawnWand(ws uint32, ng int, x, y float64, biome string, noMoreShuffle bool) *Item {
	wandX := math.Floor(x)
	wandY := math.Floor(y)
	typeName := getWandType(ws, ng, wandX, wandY-5, biome)
	if typeName == "" {
		return nil
	}
	wand := generateWandByType(ws, ng, wandX, wandY, typeName, noMoreShuffle)
	if wand == nil {
		return nil
	}
	wand.X = wandX
	wand.Y = wandY
	return &Item{ItemType: "wand", Wand: wand, X: x, Y: y}
}

// generateWandByType dispatches to premade or regular wand generation.
func generateWandByType(ws uint32, ng int, x, y float64, typeName string, noMoreShuffle bool) *Wand {
	if len(typeName) > 7 && typeName[:7] == "premade" {
		return generateLevel1Wand(ws, ng, x, y, typeName)
	}
	td, ok := wandTypes[typeName]
	if !ok {
		return nil
	}
	return generateGun(ws, ng, td.wandType, td.cost, td.level, td.forceUnshuffle, x, y, noMoreShuffle)
}
