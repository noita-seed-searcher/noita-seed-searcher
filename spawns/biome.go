package main

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	_ "image/png"
)

//go:embed data/biome_maps/biome_map.png data/biome_maps/biome_map_newgame_plus.png data/biome_maps/biome_map_nightmare.png
var biomeMapFS embed.FS

// BiomeMap is the per-seed biome color map: a width*height grid of 0xFFRRGGBB
// pixels, plus the derived heaven/hell maps and the list of orb rooms.
// Ported from noita-telescope/js/biome_generator.js (generateBiomeData).
type BiomeMap struct {
	Pixels []uint32 // len W*H, 0xFFRRGGBB
	Heaven []uint32
	Hell   []uint32
	Orbs   []Orb
	W, H   int
}

// Orb is an orb-room marker placed on the biome map.
type Orb struct {
	X, Y int
	Name string
}

const (
	biomeWNG0 = 70
	biomeWNGP = 64
	biomeH    = 48
)

// selectBiomeBase returns the embedded base-map filename and dimensions for a
// (gameMode, ng) combination, matching app.js's base/width selection.
func selectBiomeBase(gameMode string, ng int) (file string, w, h int) {
	isNGP := ng > 0
	switch {
	case isNGP:
		return "data/biome_maps/biome_map_newgame_plus.png", biomeWNGP, biomeH
	case gameMode == "nightmare":
		return "data/biome_maps/biome_map_nightmare.png", biomeWNGP, biomeH
	default:
		return "data/biome_maps/biome_map.png", biomeWNG0, biomeH
	}
}

// loadBiomeBase decodes an embedded base map into a 0xFFRRGGBB pixel slice,
// matching the JS `0xFF000000 | r<<16 | g<<8 | b` conversion from straight RGBA.
func loadBiomeBase(file string, w, h int) ([]uint32, error) {
	data, err := biomeMapFS.ReadFile(file)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	// Normalize to straight (non-premultiplied) RGBA, like canvas getImageData.
	b := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, b.Min, draw.Src)

	pixels := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := nrgba.PixOffset(x, y)
			r := uint32(nrgba.Pix[i])
			g := uint32(nrgba.Pix[i+1])
			bl := uint32(nrgba.Pix[i+2])
			pixels[y*w+x] = 0xFF000000 | (r << 16) | (g << 8) | bl
		}
	}
	return pixels, nil
}

// biomePainter mirrors the BiomePainter class in biome_generator.js.
type biomePainter struct {
	pixels      []uint32
	w, h        int
	rng         *NollaPrng
	isNightmare bool
}

func (p *biomePainter) set(x, y int, c uint32) {
	if x < 0 || x >= p.w || y < 0 || y >= p.h {
		return
	}
	p.pixels[y*p.w+x] = c
}

func (p *biomePainter) rect(x, y, w, h int, c uint32, buf int) {
	ex := p.rng.random(0, buf)
	x -= ex
	w += ex + p.rng.random(0, buf)
	for iy := y; iy < y+h; iy++ {
		for ix := x; ix < x+w; ix++ {
			p.set(ix, iy, c)
		}
	}
}

func (p *biomePainter) rectSplit(x, y, w, h int, c1, c2 uint32, buf int) {
	ex := p.rng.random(0, buf)
	x -= ex
	w += ex + p.rng.random(0, buf)
	cut := p.rng.random(y+1, y+h-2)

	for ix := x; ix < x+w; ix++ {
		for iy := y; iy < y+h; iy++ {
			if iy < cut {
				p.set(ix, iy, c1)
			} else {
				p.set(ix, iy, c2)
			}
		}
		cut += p.rng.random(-1, 1)
		cut = maxInt(y+1, minInt(y+h-2, cut))
	}
}

func (p *biomePainter) cave(x, y, dir int, c uint32, length int) {
	for i := 1; i <= length; i++ {
		p.set(x, y, c)

		// Random walk X
		if i < 5 || p.rng.random(0, 100) < 75 {
			x += dir
		} else {
			x -= dir
		}
		x = maxInt(2, minInt(62, x))

		p.set(x, y, c)

		if p.isNightmare {
			y = maxInt(17, minInt(45, y))
		}

		// Random walk Y
		if i > 3 {
			if p.rng.random(0, 100) < 65 {
				y++
			} else {
				y--
			}
		}

		if !p.isNightmare {
			y = maxInt(17, minInt(45, y))
		}

		// Blobbing
		if i > 6 {
			if p.rng.random(0, 100) < 35 {
				p.set(x-1, y, c)
			}
			if p.rng.random(0, 100) < 35 {
				p.set(x+1, y, c)
			}
			if p.rng.random(0, 100) < 35 {
				p.set(x, y-1, c)
			}
			if p.rng.random(0, 100) < 35 {
				p.set(x, y+1, c)
			}
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateBiomeData ports generateBiomeData(seed, ng, gameMode, ...) from
// biome_generator.js. seed is the world seed; ng the New Game+ count.
func generateBiomeData(seed uint32, ng int, gameMode string) (*BiomeMap, error) {
	file, width, height := selectBiomeBase(gameMode, ng)
	pixels, err := loadBiomeBase(file, width, height)
	if err != nil {
		return nil, err
	}
	orbs := []Orb{}
	isNightmare := gameMode == "nightmare"

	// NG0 normal: static base map; only heaven/hell are derived.
	if ng == 0 && !isNightmare {
		heaven, hell := buildHeavenHell(pixels, width, height)
		return &BiomeMap{Pixels: pixels, Heaven: heaven, Hell: hell, Orbs: orbs, W: width, H: height}, nil
	}

	rng := &NollaPrng{}
	rng.setRandomSeed(seed+uint32(ng), 4573, 4621)
	painter := &biomePainter{pixels: pixels, w: width, h: height, rng: rng, isNightmare: isNightmare}

	// 1. Biome colors (standard palette)
	b := map[string]uint32{
		"coal":      0xFFD56517,
		"coll":      0xFFD56517,
		"fungi":     0xFFE861F0,
		"excav":     0xFF124445,
		"snow":      0xFF1775D5,
		"hiisi":     0xFF0046FF,
		"j1":        0xFFA08400,
		"j2":        0xFF808000,
		"vault":     0xFF008000,
		"sand":      0xFFE1CD32,
		"snowvault": 0xFF0080A8,
		"wand":      0xFF006C42,
		"crypt":     0xFF786C42,
	}
	if isNightmare && ng == 0 {
		b["coal"] = 0xFFD57917
	}

	// 2. NG+ variations
	doWalls := false
	if ng > 0 {
		if ng%2 == 0 {
			// Tower
			b["coal"] = 0xFF3D3E37
			b["coll"] = 0xFF3D3E37
			b["fungi"] = 0xFF3D3E3B
			b["excav"] = 0xFF3D3E38
			b["snow"] = 0xFF3D3E39
			b["hiisi"] = 0xFF3D3E3A
			b["j1"] = 0xFF3D3E3C
			b["j2"] = 0xFF3D3E3C
			b["vault"] = 0xFF3D3E3D
			b["crypt"] = 0xFF3D3E3E
		}

		if ng%3 == 0 {
			// Shuffled world
			pool := []uint32{
				0xFFD56517, 0xFFD56517, 0xFFE861F0, 0xFF124445,
				0xFF1775D5, 0xFF0046FF, 0xFFA08400, 0xFF808000,
				0xFF008000, 0xFFE1CD32, 0xFF0080A8, 0xFF006C42,
				0xFF786C42,
			}
			shuffleU32(pool, rng)
			b["coal"] = pool[0]
			b["coll"] = pool[1]
			b["fungi"] = pool[2]
			b["excav"] = pool[3]
			b["snow"] = pool[4]
			b["hiisi"] = pool[5]
			b["j1"] = pool[6]
			b["j2"] = pool[7]
			b["vault"] = pool[8]
			b["crypt"] = pool[9]
		}

		doWalls = ng%5 == 0

		if ng%7 == 0 {
			// Specific color replacement at (16, 5)
			t := (ng / 7) % 3
			var c uint32
			switch t {
			case 0:
				c = 0xFFCC9944
			case 1:
				c = 0xFFD6D8E3
			default:
				c = 0xFF33E311
			}
			target := pixels[5*width+16]
			for i := range pixels {
				if pixels[i] == target {
					pixels[i] = c
				}
			}
		}
	}

	// 3. Swaps
	swap := func(k1, k2 string) {
		if rng.random(0, 100) < 35 {
			b[k1], b[k2] = b[k2], b[k1]
		}
	}
	if !isNightmare || ng > 0 {
		swap("coal", "coll")
	} else {
		b["coal"], b["excav"] = b["excav"], b["coal"]
	}
	swap("fungi", "excav")
	swap("snow", "hiisi")
	swap("j1", "j2")
	swap("sand", "fungi")
	swap("wand", "sand")

	// 4. Caves
	doCave := func(x, y, dir int, c uint32, lMin, lMax int) {
		if rng.random(0, 100) < 65 {
			painter.cave(x, y, dir, c, rng.random(lMin, lMax))
		}
	}

	doCave(27, 15, -1, b["fungi"], 4, 50)
	doCave(35, 15, 1, b["fungi"], 4, 50)
	doCave(27, 18, -1, b["fungi"], 4, 50)
	doCave(35, 18, 1, b["fungi"], 4, 50)

	if rng.random(0, 100) < 65 {
		painter.cave(27, 20+rng.random(0, 5), -1, b["wand"], rng.random(5, 50))
	}
	if rng.random(0, 100) < 65 {
		painter.cave(35, 20+rng.random(0, 5), 1, b["wand"], rng.random(5, 50))
	}
	if rng.random(0, 100) < 65 {
		painter.cave(27, 27+rng.random(0, 6), -1, b["sand"], rng.random(5, 50))
	}
	if rng.random(0, 100) < 65 {
		painter.cave(35, 27+rng.random(0, 6), 1, b["sand"], rng.random(5, 50))
	}

	// 5. Rect areas
	painter.rect(32, 14, 3, 2, b["coal"], 0)
	painter.rect(28, 15, 4, 1, b["coll"], 1)
	if !isNightmare || ng > 0 {
		painter.rect(28, 17, 4, 2, b["excav"], 2)
		painter.rectSplit(28, 20, 7, 6, b["snow"], b["hiisi"], 3)
	} else {
		painter.rect(28, 17, 4, 4, b["snow"], 2)
		painter.rect(28, 22, 7, 4, b["hiisi"], 3)
	}
	painter.rectSplit(28, 27, 7, 4, b["j1"], b["j2"], 4)
	painter.rectSplit(28, 29, 7, 5, b["j2"], b["vault"], 4)
	if !isNightmare || ng > 0 {
		painter.rect(29, 35, 11, 3, b["crypt"], 0)
	}

	if doWalls {
		wallLeft := rng.random(2, 6)
		wallRight := rng.random(1, 4)
		painter.rect(23, 15, wallLeft, 25, 0xFF3D3D3D, 0)
		painter.rect(33+(4-wallRight), 16, wallRight, 22, 0xFF3D3D3D, 0)
	}

	// 6. Orbs
	addOrb := func(x, y int, name string, color uint32) {
		painter.set(x, y, color)
		orbs = append(orbs, Orb{X: x, Y: y, Name: name})
	}

	addOrb(51, 11, "Pyramid", 0xFFC88F5F)
	addOrb(33, 11, "Floating Island", 0xFFC08082)
	if !isNightmare || ng > 0 {
		addOrb(rng.random(0, 5)+10, rng.random(0, 2)+18, "Vault", 0xFFFFD102)
		addOrb(rng.random(0, 5)+49, rng.random(0, 3)+17, "Pyramid (Inside)", 0xFFFFD104)

		hx := rng.random(0, 9) + 27
		hy := rng.random(0, 2) + 44
		if ng == 3 || ng >= 25 {
			hy = 47
		}
		addOrb(hx, hy, "Hell", 0xFFFFD108)

		addOrb(rng.random(0, 6)+12, rng.random(0, 3)+40, "Snowcave", 0xFFFFD109)
		addOrb(rng.random(0, 4)+51, rng.random(0, 5)+41, "Desert", 0xFFFFD110)
		addOrb(rng.random(0, 5)+58, rng.random(0, 5)+34, "Nuke", 0xFFFFD103)
		addOrb(rng.random(0, 9)+40, rng.random(0, 11)+21, "Orb 1", 0xFFFFD105)
		addOrb(rng.random(0, 7)+17, rng.random(0, 8)+21, "Orb 2", 0xFFFFD106)
		addOrb(rng.random(0, 7)+1, rng.random(0, 9)+24, "Orb 3", 0xFFFFD107)
	} else {
		addOrb(12, 19, "Vault", 0xFFFFD102)
		addOrb(51, 19, "Pyramid (Inside)", 0xFFFFD104)
		addOrb(31, 45, "Hell", 0xFFFFD108)
		addOrb(14, 42, "Snowcave", 0xFFFFD109)
		addOrb(52, 45, "Desert", 0xFFFFD110)
	}

	// Prevent these from getting overwritten by caves
	const colorEndRoom uint32 = 0xFF50EED7
	const colorBossArena uint32 = 0xFF14EED7

	painter.set(44, 43, colorEndRoom)

	painter.rect(35, 38, 5, 2, colorBossArena, 0)
	painter.set(37, 40, colorBossArena)
	painter.set(38, 40, colorBossArena)

	heaven, hell := buildHeavenHell(pixels, width, height)
	return &BiomeMap{Pixels: pixels, Heaven: heaven, Hell: hell, Orbs: orbs, W: width, H: height}, nil
}

// buildHeavenHell repeats the first/last row across all rows (matching JS).
func buildHeavenHell(pixels []uint32, width, height int) (heaven, hell []uint32) {
	heaven = make([]uint32, len(pixels))
	hell = make([]uint32, len(pixels))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			heaven[y*width+x] = pixels[x%width]
			hell[y*width+x] = pixels[(height-1)*width+(x%width)]
		}
	}
	return heaven, hell
}

// shuffleU32 is the uint32 variant of shuffleTable (utils.js), used for the
// NG+ shuffled-world color pool.
func shuffleU32(arr []uint32, p *NollaPrng) {
	for i := len(arr) - 1; i >= 1; i-- {
		j := p.random(0, i)
		arr[i], arr[j] = arr[j], arr[i]
	}
}
