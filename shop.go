package main

import "math"

// ShopType mirrors IShopType from Shop.ts.
type ShopType int

const (
	ShopTypeItem ShopType = 2
	ShopTypeWand ShopType = 1
)

// ShopItem is a spell item in a shop.
type ShopItem struct {
	SpellID    string
	Price      float64
	IsStealable bool
}

// ShopWand is a wand in a shop.
type ShopWand struct {
	Wand  WandResult
	Price float64
}

// ShopResult is one shop floor's contents.
type ShopResult struct {
	Type  ShopType
	Items []ShopItem
	Wands []ShopWand
}

// shopBiomes maps biomepixel (floor(y/512)) to biome level, matching TS biomes array.
var shopBiomes = []int{
	0,    // 0
	0, 0, 0,  // 1-3
	1, 1, 1,  // 4-6
	2, 2, 2, 2, 2, 2, // 7-12
	3, 3, 3, 3,  // 13-16
	4, 4, 4, 4,  // 17-20
	5, 5, 5, 5,  // 21-24
	6, 6, 6, 6, 6, 6, 6, 6, 6, // 25-33
}

func shopGetLevel(y float64) int {
	biomepixel := int(math.Floor(y / 512))
	biomeid := 0
	if biomepixel < len(shopBiomes) {
		biomeid = shopBiomes[biomepixel]
	}
	if biomepixel > 35 {
		biomeid = 7
	}
	return biomeid
}

// shopGetSpellPrice returns price for a spell item, mirroring generate_shop_item.
// We don't have spell prices in spells.json (stripped), so we use 0 for spell.price.
func shopGetSpellPrice(spellID string, level, biomeid int, cheapItem bool) float64 {
	biomeidSq := biomeid * biomeid
	// spell.price is not available (stripped from spells.json) — use 0
	price := math.Max(math.Floor(float64(0*0.3+70*float64(biomeidSq))/10)*10, 10)
	if cheapItem {
		price = 0.5 * price
	}
	if biomeidSq >= 10 {
		price *= 5.0
	}
	return price
}

var shopGunParams = map[bool]map[int]struct{ cost, level int }{
	true: { // shuffle
		1: {30, 1}, 2: {40, 2}, 3: {60, 3}, 4: {80, 4},
		5: {100, 5}, 6: {120, 6}, 10: {200, 11},
	},
	false: { // unshuffle
		1: {25, 1}, 2: {40, 2}, 3: {60, 3}, 4: {80, 4},
		5: {100, 5}, 6: {120, 6}, 10: {180, 11},
	},
}

func generateShopItem(rng *RNG, x, y float64, cheapItem bool, biomeidOverride *int) ShopItem {
	rng.SetRandomSeed(x, y)
	level := shopGetLevel(y)
	if biomeidOverride != nil {
		level = *biomeidOverride
	}
	biomeid := level * level
	spellID := GetRandomAction(rng, x, y, level, 0)
	price := shopGetSpellPrice(spellID, level, biomeid, cheapItem)
	return ShopItem{SpellID: spellID, Price: price, IsStealable: true}
}

func generateShopWand(rng *RNG, x, y float64, cheapItem bool, unshufflePerk bool, biomeidOverride *int) ShopWand {
	rng.SetRandomSeed(x, y)
	level := shopGetLevel(y)
	if biomeidOverride != nil {
		level = *biomeidOverride
	}
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	rand := rng.RandomInt(0, 100)
	isShuffle := rand <= 50
	config := shopGunParams[isShuffle][level]
	biomeidF := 0.5*float64(level) + 0.5*float64(level)*float64(level)
	wandcost := 50 + biomeidF*210 + float64(rng.RandomInt(-15, 15))*10
	if cheapItem {
		wandcost = 0.5 * wandcost
	}
	rx := float64(roundHalfToEvenI32(x))
	ry := float64(roundHalfToEvenI32(y))
	wand := ProvideWand(rng, rx, ry, float64(config.cost), config.level, !isShuffle, unshufflePerk)
	wand.Gun.Cost = wandcost
	return ShopWand{Wand: wand, Price: wandcost}
}

// SpawnAllShopItems mirrors spawn_all_shop_items from Shop.ts.
func SpawnAllShopItems(rng *RNG, x, y float64, unshufflePerk bool) ShopResult {
	rng.SetRandomSeed(x, y)
	count := 5
	width := float64(132)
	itemWidth := width / float64(count)
	saleItemI := rng.RandomInt(0, int32(count-1))
	shopType := ShopTypeItem
	if rng.RandomInt(0, 100) > 50 {
		shopType = ShopTypeWand
	}
	res := ShopResult{Type: shopType}
	if shopType == ShopTypeItem {
		// TS stores items[0..count-1] from y-30, then items[count..2*count-1] from y
		for i := 0; i < count; i++ {
			ix := x + float64(i)*itemWidth
			res.Items = append(res.Items, generateShopItem(rng, ix, y-30, false, nil))
		}
		for i := 0; i < count; i++ {
			ix := x + float64(i)*itemWidth
			res.Items = append(res.Items, generateShopItem(rng, ix, y, i == int(saleItemI), nil))
		}
	} else {
		for i := 0; i < count; i++ {
			ix := x + float64(i)*itemWidth
			res.Wands = append(res.Wands, generateShopWand(rng, ix, y, i == int(saleItemI), unshufflePerk, nil))
		}
	}
	return res
}

// ProvideShopLevel generates the shop at the given temple level (0-indexed).
func ProvideShopLevel(rng *RNG, level int, unshufflePerk bool) ShopResult {
	if level >= len(templeLocations) {
		return ShopResult{}
	}
	temple := templeLocations[level]
	offsetX := float64(0 - 299)
	offsetY := float64(0 - 15)
	x := temple.X + offsetX
	y := temple.Y + offsetY
	return SpawnAllShopItems(rng, x, y, unshufflePerk)
}
