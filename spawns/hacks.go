package main

import (
	"bytes"
	"image"
	"image/draw"
	"math"
)

// Port of the post-stbhw buffer transforms from noita-telescope:
// biome_hacks.js, pathfinding.js (findMinPath), pixel_scene_generation.js
// (blockOutRooms) and tile_generator.js (applyMasking). These mutate the tile
// buffer that the PoI scanner later reads.

// getWorldCenter ports utils.js getWorldCenter.
func getWorldCenter(isNGP bool, gameMode string) int {
	if gameMode == "nightmare" {
		return 32
	}
	if isNGP {
		return 32
	}
	return 35
}

const (
	biomePathFindWorldPosX      = 159
	biomePathFindWorldPosXMines = 975
)

// getMainBiomePathStartX ports biome_hacks.js getMainBiomePathStartX.
func getMainBiomePathStartX(biomeName string, chunkX int, isNGP bool, gameMode string) int {
	startX := int(math.Floor(float64(biomePathFindWorldPosX-(chunkX-getWorldCenter(isNGP, gameMode))*512) / 10.0))
	if biomeName == "coalmine" {
		startX = int(math.Floor(float64(biomePathFindWorldPosXMines-(chunkX-getWorldCenter(isNGP, gameMode))*512) / 10.0))
	}
	return startX
}

// applyMainBiomeHack ports biome_hacks.js applyMainBiomeHack: clear an
// entrance strip at the top of the main biome.
func applyMainBiomeHack(chunkX int, pixels []byte, width, height int, biomeName string, isNGP bool, gameMode string) {
	startX := getMainBiomePathStartX(biomeName, chunkX, isNGP, gameMode)
	for y := 0; y < 11; y++ {
		for x := startX; x < startX+7; x++ {
			if x >= 0 && x < width && y >= 0 && y < height {
				idx := (y*width + x) * 3
				pixels[idx] = 0
				pixels[idx+1] = 0
				pixels[idx+2] = 0
			}
		}
	}
}

// overlay is a decoded RGBA overlay image.
type overlay struct {
	data []byte // RGBA, len w*h*4
	w, h int
}

// loadOverlay decodes an embedded RGBA overlay PNG (straight alpha).
func loadOverlay(file string) (*overlay, error) {
	raw, err := wangFS.ReadFile(file)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, b.Min, draw.Src)
	return &overlay{data: nrgba.Pix, w: b.Dx(), h: b.Dy()}, nil
}

// applyCoalmineHack / undoCoalmineHack port biome_hacks.js. They differ only in
// how the "border" pixels are written (1,1,1 vs 0,0,0).
func applyOverlayHack(pixels []byte, width, height int, ov *overlay, borderVal byte) {
	hMax := height
	if ov.h < hMax {
		hMax = ov.h
	}
	wMax := width
	if ov.w < wMax {
		wMax = ov.w
	}
	for y := 0; y < hMax; y++ {
		for x := 0; x < wMax; x++ {
			oIdx := (y*ov.w + x) * 4
			r := ov.data[oIdx]
			g := ov.data[oIdx+1]
			b := ov.data[oIdx+2]
			a := ov.data[oIdx+3]
			if a == 0 {
				continue
			}
			hex := (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
			pIdx := ((y+4)*width + x) * 3
			if pIdx < 0 || pIdx+2 >= len(pixels) {
				continue
			}
			if hex == 0x000042 {
				pixels[pIdx] = 0
				pixels[pIdx+1] = 0
				pixels[pIdx+2] = 0
			} else if hex != 0xffffff {
				pixels[pIdx] = borderVal
				pixels[pIdx+1] = borderVal
				pixels[pIdx+2] = borderVal
			}
		}
	}
}

func applyCoalmineHack(pixels []byte, width, height int, ov *overlay) {
	applyOverlayHack(pixels, width, height, ov, 1)
}

func undoCoalmineHack(pixels []byte, width, height int, ov *overlay) {
	applyOverlayHack(pixels, width, height, ov, 0)
}

// floodFillColor fills the connected region of pixels matching (mr,mg,mb)
// starting at (x,y) with (vr,vg,vb). Mirrors the repeated flood-fills in
// biome_hacks.js (clearPath/applyCoffeeHack/applyRandomColors).
func floodFillColor(pixels []byte, width, height, x, y int, mr, mg, mb, vr, vg, vb byte) {
	type pt struct{ x, y int }
	queue := []pt{{x, y}}
	visited := map[int]bool{}
	for head := 0; head < len(queue); head++ {
		cx, cy := queue[head].x, queue[head].y
		idx := (cy*width + cx) * 3
		pixels[idx] = vr
		pixels[idx+1] = vg
		pixels[idx+2] = vb
		neighbors := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, d := range neighbors {
			nx, ny := cx+d[0], cy+d[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				key := ny*width + nx
				nidx := key * 3
				if !visited[key] && pixels[nidx] == mr && pixels[nidx+1] == mg && pixels[nidx+2] == mb {
					visited[key] = true
					queue = append(queue, pt{nx, ny})
				}
			}
		}
	}
}

// clearPath ports biome_hacks.js clearPath: turn coffee (0xc0ffee) pixels along
// the path to black.
func clearPath(pixels []byte, width, height int, path []point) {
	for _, p := range path {
		x, y := p.X, p.Y
		if x >= 0 && x < width && y >= 0 && y < height {
			idx := (y*width + x) * 3
			if pixels[idx] == 0xc0 && pixels[idx+1] == 0xff && pixels[idx+2] == 0xee {
				floodFillColor(pixels, width, height, x, y, 0xc0, 0xff, 0xee, 0, 0, 0)
			}
		}
	}
}

// applyCoffeeHack ports biome_hacks.js applyCoffeeHack: fill remaining coffee
// regions with black or white per a world-seed PRNG roll.
func applyCoffeeHack(pixels []byte, width, height int, worldSeed uint32) {
	prng := &NollaPrng{}
	prng.setRandomFromWorldSeed(float64(worldSeed))
	prng.next()
	for y := 4; y < height; y++ {
		for x := 0; x < width; x++ {
			startIdx := (y*width + x) * 3
			if pixels[startIdx] == 0xc0 && pixels[startIdx+1] == 0xff && pixels[startIdx+2] == 0xee {
				var sel byte = 0x00
				if prng.next() < 0.5 {
					sel = 0xff
				}
				floodFillColor(pixels, width, height, x, y, 0xc0, 0xff, 0xee, sel, sel, sel)
			}
		}
	}
}

// applyRandomColors ports biome_hacks.js applyRandomColors (used by liquidcave).
// randomColors maps a source 0xRRGGBB to a list of replacement 0xRRGGBB options.
func applyRandomColors(pixels []byte, width, height int, worldSeed uint32, ngPlus int, randomColors map[uint32][]uint32) {
	for color, options := range randomColors {
		prng := &NollaPrng{}
		prng.setRandomFromWorldSeed(float64(worldSeed) + float64(ngPlus))
		prng.next()
		wsng := int64(worldSeed) + int64(ngPlus)
		iters := width + int(wsng) - 11*(width/11) - 12*int(wsng/12)
		for i := 0; i < iters; i++ {
			prng.next()
		}
		prng.next()

		mr := byte((color >> 16) & 0xff)
		mg := byte((color >> 8) & 0xff)
		mb := byte(color & 0xff)
		for y := 4; y < height; y++ {
			for x := 0; x < width; x++ {
				startIdx := (y*width + x) * 3
				if pixels[startIdx] == mr && pixels[startIdx+1] == mg && pixels[startIdx+2] == mb {
					sel := options[int(prng.next()*float64(len(options)))]
					vr := byte((sel >> 16) & 0xff)
					vg := byte((sel >> 8) & 0xff)
					vb := byte(sel & 0xff)
					floodFillColor(pixels, width, height, x, y, mr, mg, mb, vr, vg, vb)
				}
			}
		}
	}
}

// applyPostprocessingHacks ports biome_hacks.js applyPostprocessingHacks.
func applyPostprocessingHacks(pixels []byte, width, height int, worldSeed uint32, ngPlus int, path []point, randomColors map[uint32][]uint32) {
	clearPath(pixels, width, height, path)
	applyCoffeeHack(pixels, width, height, worldSeed)
	if randomColors != nil {
		applyRandomColors(pixels, width, height, worldSeed, ngPlus, randomColors)
	}
}

// blockedColors ports pixel_scene_config.js BLOCKED_COLORS.
var blockedColors = map[uint32]bool{
	0x00ac6e: true, 0x70d79e: true, 0x70d79f: true, 0x70d7a0: true,
	0x70d7a1: true, 0x7868ff: true, 0xc35700: true, 0xff0080: true,
	0xff00ff: true, 0xff0aff: true, 0x00AC64: true,
}

// room is a blocked-out pixel-scene room (pixel_scene_generation.js).
type room struct {
	color                      uint32
	startX, startY, endX, endY int
}

// blockOutRooms ports pixel_scene_generation.js blockOutRooms.
func blockOutRooms(pixels []byte, width, height int) []room {
	rooms := []room{}
	for y := 4; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			color := (uint32(pixels[idx]) << 16) | (uint32(pixels[idx+1]) << 8) | uint32(pixels[idx+2])
			if color == 0x000000 || color == 0xffffff {
				continue
			}
			if !blockedColors[color] {
				continue
			}

			startX := x + 1
			startY := y + 1
			endX := x + 1
			endY := y + 1
			foundEnd := false
			for !foundEnd && endX < width {
				tempIdx := (startY*width + endX) * 3
				tempColor := (uint32(pixels[tempIdx]) << 16) | (uint32(pixels[tempIdx+1]) << 8) | uint32(pixels[tempIdx+2])
				if tempColor == 0x000000 || tempColor == 0x323232 {
					endX++
					continue
				}
				endX--
				foundEnd = true
			}
			if endX >= width {
				endX = width - 1
			}
			foundEnd = false
			for !foundEnd && endY < height {
				tempIdx := (endY*width + startX) * 3
				tempColor := (uint32(pixels[tempIdx]) << 16) | (uint32(pixels[tempIdx+1]) << 8) | uint32(pixels[tempIdx+2])
				if tempColor == 0x000000 || tempColor == 0x323232 {
					endY++
					continue
				}
				endY--
				foundEnd = true
			}
			if endY >= height {
				endY = height - 1
			}

			if endX > startX && endY > startY {
				for by := startY; by <= endY; by++ {
					for bx := startX; bx <= endX; bx++ {
						bIdx := (by*width + bx) * 3
						pixels[bIdx] = 0xff
						pixels[bIdx+1] = 0x01
						pixels[bIdx+2] = 0xff
					}
				}
			}
			rooms = append(rooms, room{color: color, startX: startX, startY: startY, endX: endX, endY: endY})
		}
	}
	return rooms
}

// findSequences ports pathfinding.js findSequences: black runs along a row.
func findSequences(pixels []byte, width, rowY, stride int) [][2]int {
	seqs := [][2]int{}
	start := -1
	rowOffset := rowY * width
	for x := 0; x < width; x++ {
		idx := (rowOffset + x) * stride
		isBlack := pixels[idx] == 0 && pixels[idx+1] == 0 && pixels[idx+2] == 0
		if isBlack {
			if start == -1 {
				start = x
			}
		} else {
			if start != -1 {
				seqs = append(seqs, [2]int{start, x - 1})
				start = -1
			}
		}
	}
	if start != -1 {
		seqs = append(seqs, [2]int{start, width - 1})
	}
	return seqs
}

// findMinPath ports pathfinding.js findMinPath: BFS a walkable path from the top
// entrance to the bottom row. Returns nil if no path exists (triggers a reroll).
func findMinPath(bbox [4]int, pixels []byte, width, height int, biomeName string, isNGPlus bool, gameMode string) []point {
	const stride = 3
	startY := 4
	var topSequences [][2]int

	if bbox[0] <= getWorldCenter(isNGPlus, gameMode) && bbox[2] >= getWorldCenter(isNGPlus, gameMode) {
		startY = 4
		startX := getMainBiomePathStartX(biomeName, bbox[0], isNGPlus, gameMode)
		idx := (startY*width + startX) * stride
		if startX >= 0 && startX < width && idx+2 < len(pixels) &&
			pixels[idx] == 0 && pixels[idx+1] == 0 && pixels[idx+2] == 0 {
			topSequences = append(topSequences, [2]int{startX, startX})
		}
	} else {
		topSequences = findSequences(pixels, width, startY, stride)
	}

	if len(topSequences) == 0 {
		return nil
	}

	directions := [4][2]int{{0, 1}, {-1, 0}, {1, 0}, {0, -1}}

	for _, startSeq := range topSequences {
		startX := (startSeq[0] + startSeq[1]) / 2

		visited := make([]bool, width*height)
		parents := make([]int32, width*height)
		for i := range parents {
			parents[i] = -1
		}

		queue := []point{{startX, startY}}
		visited[startY*width+startX] = true
		parents[startY*width+startX] = -2

		found := false
		var finalNode point

		for head := 0; head < len(queue); head++ {
			curr := queue[head]
			if curr.Y == height-1 {
				found = true
				finalNode = curr
				break
			}
			for _, d := range directions {
				nx, ny := curr.X+d[0], curr.Y+d[1]
				if nx >= 0 && nx < width && ny > 3 && ny < height {
					nIdx := ny*width + nx
					if !visited[nIdx] {
						pIdx := nIdx * stride
						pixelColor := (uint32(pixels[pIdx]) << 16) | (uint32(pixels[pIdx+1]) << 8) | uint32(pixels[pIdx+2])
						if pixelColor == 0x000000 || pixelColor == 0xc0ffee || pixelColor == 0x8aff80 {
							visited[nIdx] = true
							parents[nIdx] = int32(curr.Y*width + curr.X)
							queue = append(queue, point{nx, ny})
						}
					}
				}
			}
		}

		if found {
			path := []point{}
			currIdx := int32(finalNode.Y*width + finalNode.X)
			for currIdx != -2 && currIdx != -1 {
				py := int(currIdx) / width
				px := int(currIdx) % width
				path = append(path, point{px, py})
				pIdx := parents[currIdx]
				if pIdx == -2 {
					break
				}
				currIdx = pIdx
			}
			// reverse
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path
		}
	}
	return nil
}

// applyMasking ports tile_generator.js applyMasking. We only need the buffer
// side-effect (invalid chunks zeroed); the imgData render output is omitted.
func applyMasking(pixels []byte, mapW int, bbox [4]int, validChunks map[[2]int]bool, offsetY int) {
	minCX, minCY, maxCX, maxCY := bbox[0], bbox[1], bbox[2], bbox[3]
	tx := 0
	for cx := minCX; cx <= maxCX; cx++ {
		cw := 51
		if cx%5 == 4 {
			cw++
		}
		ty := 0
		for cy := minCY; cy <= maxCY; cy++ {
			ch := 51
			if cy%5 == 4 {
				ch++
			}
			if !validChunks[[2]int{cx, cy}] {
				for y := 0; y < ch; y++ {
					for x := 0; x < cw; x++ {
						srcIdx := ((ty+y+offsetY)*mapW + (tx + x)) * 3
						if srcIdx >= 0 && srcIdx+2 < len(pixels) {
							pixels[srcIdx] = 0
							pixels[srcIdx+1] = 0
							pixels[srcIdx+2] = 0
						}
					}
				}
			}
			ty += ch
		}
		tx += cw
	}
}
