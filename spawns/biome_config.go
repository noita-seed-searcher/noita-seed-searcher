package main

type biomeEntry struct {
	biomeName    string
	color        uint32
	name         string
	wangFile     string
	randomColors map[uint32][]uint32
}

var biomeConfig = []biomeEntry{
	{"coalmine", 0xffd57917, "Mines", "data/wang_tiles/coalmine.png", nil},
	{"coalmine_alt", 0xffD56517, "Collapsed Mines", "data/wang_tiles/coalmine_alt.png", nil},
	{"excavationsite", 0xff124445, "Coal Pits", "data/wang_tiles/excavationsite.png", nil},
	{"snowcave", 0xff1775d5, "Snowy Depths", "data/wang_tiles/snowcave.png", nil},
	{"snowcastle", 0xff0046FF, "Hiisi Base", "data/wang_tiles/snowcastle.png", nil},
	{"rainforest", 0xff808000, "Underground Jungle", "data/wang_tiles/rainforest.png", nil},
	{"rainforest_open", 0xffA08400, "Underground Jungle (Open)", "data/wang_tiles/rainforest_open.png", nil},
	{"vault", 0xff008000, "The Vault", "data/wang_tiles/vault.png", nil},
	{"crypt", 0xff786C42, "Temple of the Art", "data/wang_tiles/crypt.png", nil},
	{"fungicave", 0xffe861f0, "Fungal Caverns", "data/wang_tiles/fungicave.png", nil},
	{"fungiforest", 0xffa861ff, "Overgrown Cavern", "data/wang_tiles/fungiforest.png", nil},
	{"rainforest_dark", 0xff375c00, "Lukki Lair", "data/wang_tiles/rainforest_dark.png", nil},
	{"wizardcave", 0xff726186, "Wizard's Den", "data/wang_tiles/wizardcave.png", nil},
	{"liquidcave", 0xff89a04b, "Ancient Laboratory", "data/wang_tiles/liquidcave.png", map[uint32][]uint32{0x01CFEE: {0xF86868, 0x7FCEEA, 0xA3569F, 0xC23055, 0x0BFFE5}}},
	{"robobase", 0xff4e5267, "Power Plant", "data/wang_tiles/robobase.png", nil},
	{"vault_frozen", 0xff0080a8, "Frozen Vault", "data/wang_tiles/vault_frozen.png", nil},
	{"meat", 0xff572828, "Meat Realm", "data/wang_tiles/meat.png", nil},
	{"wandcave", 0xff006C42, "Magical Temple", "", nil},
	{"pyramid", 0xff967f11, "Pyramid", "", nil},
	{"sandcave", 0xffE1CD32, "Sandcave", "", nil},
	{"clouds", 0xff36d5c9, "Cloudscape", "", nil},
	{"the_sky", 0xffD3E6F0, "The Work (Sky)", "", nil},
	{"the_end", 0xff3C0F0A, "The Work (Hell)", "", nil},
	{"winter_caves", 0xff77A5BD, "Snowy Chasm", "", nil},
	{"solid_wall_tower_1", 0xff3d3e37, "Tower (Mines)", "data/wang_tiles/coalmine.png", nil},
	{"solid_wall_tower_2", 0xff3d3e38, "Tower (Coal Mines)", "data/wang_tiles/excavationsite.png", nil},
	{"solid_wall_tower_3", 0xff3d3e39, "Tower (Snowy Depths)", "data/wang_tiles/snowcave.png", nil},
	{"solid_wall_tower_4", 0xff3d3e3a, "Tower (Hiisi Base)", "data/wang_tiles/snowcastle.png", nil},
	{"solid_wall_tower_5", 0xff3d3e3b, "Tower (Fungal Caverns)", "data/wang_tiles/fungicave.png", nil},
	{"solid_wall_tower_6", 0xff3d3e3c, "Tower (Underground Jungle)", "data/wang_tiles/rainforest.png", nil},
	{"solid_wall_tower_7", 0xff3d3e3d, "Tower (The Vault)", "data/wang_tiles/vault.png", nil},
	{"solid_wall_tower_8", 0xff3d3e3e, "Tower (Temple of the Art)", "data/wang_tiles/crypt.png", nil},
	{"solid_wall_tower_9", 0xff3d3e3f, "Tower (Hell)", "data/wang_tiles/the_end.png", nil},
	{"lake_deep", 0xff1158f1, "Lake", "", nil},
	{"biome_watchtower", 0xffb70000, "Watchtower", "data/wang_tiles/static/watchtower_fg.png", nil},
	{"biome_potion_mimics", 0xffff00fe, "Henkevä Temple", "data/wang_tiles/static/potion_mimics_fg.png", nil},
	{"biome_darkness", 0xffff00fd, "Ominous Temple", "data/wang_tiles/static/darkness_fg.png", nil},
	{"biome_boss_sky", 0xffff00fc, "Kivi Temple", "data/wang_tiles/static/boss_fg.png", nil},
	{"biome_barren", 0xffff00fb, "Barren Temple", "data/wang_tiles/static/barren_fg.png", nil},
	{"solid_wall_tower_10", 0xff3d3e41, "Tower (Reward)", "", nil},
}
