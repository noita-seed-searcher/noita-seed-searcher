package main

// Port of noita-telescope/js/stbhw.js — the STB herringbone Wang-tile generator
// (corner mode only, which is what Noita's biome tilesets use). The module-level
// color grids in the JS are held on stbhwGen here to avoid global state.

const (
	stbhwMaxW = 300
	stbhwMaxH = 300
)

type stbhwTile struct {
	pixels           []byte
	a, b, c, d, e, f int32
}

type stbhwTileset struct {
	isCorner     bool
	numColor     [6]int32
	shortSideLen int
	hTiles       []stbhwTile
	vTiles       []stbhwTile
	numHTiles    int
	numVTiles    int
	numVaryX     int
	numVaryY     int
}

// stbhwGen holds the per-generation corner-color grid and the PRNG.
type stbhwGen struct {
	cColor              []int32
	prng                *NollaPrng
	repetitionReduction bool
}

func newStbhwGen(prng *NollaPrng) *stbhwGen {
	c := make([]int32, stbhwMaxW*stbhwMaxH)
	for i := range c {
		c[i] = -1
	}
	return &stbhwGen{cColor: c, prng: prng, repetitionReduction: true}
}

func (g *stbhwGen) getC(x, y int) int32 {
	if x < 0 || x >= stbhwMaxW || y < 0 || y >= stbhwMaxH {
		return -1
	}
	return g.cColor[y*stbhwMaxW+x]
}

func (g *stbhwGen) setC(x, y int, v int32) {
	if x >= 0 && x < stbhwMaxW && y >= 0 && y < stbhwMaxH {
		g.cColor[y*stbhwMaxW+x] = v
	}
}

func (g *stbhwGen) rand() uint32 { return g.prng.nextU() }

// stbhwGenerateImage ports stbhw_generate_image. Renders into pixels (RGB,
// stride bytes/row) and returns the tile-index grid plus its dimensions.
func (g *stbhwGen) stbhwGenerateImage(ts *stbhwTileset, pixels []byte, stride, w, h int) (tileIndices []int32, xmax, ymax int, ok bool) {
	for i := range g.cColor {
		g.cColor[i] = -1
	}

	sidelen := ts.shortSideLen
	if sidelen <= 0 {
		return nil, 0, 0, false
	}

	if h > 1028 {
		h = 1028
	}

	xmax = w/sidelen + 6
	ymax = h/sidelen + 6
	tileIndices = make([]int32, xmax*ymax)
	numVaryProduct := uint32(ts.numVaryX * ts.numVaryY)

	if ts.isCorner {
		// PASS 1: color generation
		for j := 0; j < ymax; j++ {
			for i := 0; i < xmax; i++ {
				p := (i - j + 1) & 3
				g.setC(i, j, int32(g.rand()%uint32(ts.numColor[p])))
			}
		}

		// PASS 2: repetition reduction
		if g.repetitionReduction {
			for j := 0; j < ymax-3; j++ {
				for i := 0; i < xmax-3; i++ {
					if g.match(i, j) && g.match(i, j+1) && g.match(i, j+2) &&
						g.match(i+1, j) && g.match(i+1, j+1) && g.match(i+1, j+2) {
						p := ((i + 1) - (j + 1) + 1) & 3
						if ts.numColor[p] > 1 {
							curr := g.getC(i+1, j+1)
							g.setC(i+1, j+1, g.changeColor(curr, ts.numColor[p]))
						}
					}
					if g.match(i, j) && g.match(i+1, j) && g.match(i+2, j) &&
						g.match(i, j+1) && g.match(i+1, j+1) && g.match(i+2, j+1) {
						p := ((i + 2) - (j + 1) + 1) & 3
						if ts.numColor[p] > 1 {
							curr := g.getC(i+2, j+1)
							g.setC(i+2, j+1, g.changeColor(curr, ts.numColor[p]))
						}
					}
				}
			}
		}

		// PASS 3: logic & draw
		yIdx := -1
		ypos := -1 * sidelen

		for j := -1; yIdx*sidelen < h; j++ {
			phase := j & 3
			i := 0
			if phase != 0 {
				i = phase - 4
			}

			for ; ; i += 4 {
				xIdx := i
				xpos := xIdx * sidelen
				if xpos >= w {
					break
				}

				// Horizontal tile
				if xIdx+2 >= 0 && yIdx >= 0 {
					ti := g.chooseTile(ts.hTiles, ts.numHTiles,
						i+2, j+2, i+3, j+2, i+4, j+2,
						i+2, j+3, i+3, j+3, i+4, j+3, numVaryProduct)
					if ti == -1 {
						return nil, 0, 0, false
					}
					if xIdx >= 0 {
						tileIndices[yIdx*xmax+xIdx] = int32(ti)
					}
					if xIdx+1 >= 0 {
						tileIndices[yIdx*xmax+xIdx+1] = int32(ti) | 0x8000
					}
					g.drawHTile(pixels, stride, w, h, xpos, ypos, &ts.hTiles[ti], sidelen)
				}

				// Vertical tile
				xIdxV := i + 3
				xposV := xIdxV * sidelen
				if xposV < w {
					ti := g.chooseTile(ts.vTiles, ts.numVTiles,
						i+5, j+2, i+5, j+3, i+5, j+4,
						i+6, j+2, i+6, j+3, i+6, j+4, numVaryProduct)
					if ti == -1 {
						return nil, 0, 0, false
					}
					if yIdx >= 0 {
						tileIndices[yIdx*xmax+xIdxV] = int32(ti) | 0x4000
					}
					if (yIdx+1)*sidelen < (h + sidelen) {
						tileIndices[(yIdx+1)*xmax+xIdxV] = int32(ti) | 0xC000
					}
					g.drawVTile(pixels, stride, w, h, xposV, ypos, &ts.vTiles[ti], sidelen)
				}
			}
			yIdx++
			ypos += sidelen
		}
	}

	return tileIndices, xmax, ymax, true
}

// chooseTile ports stbhw__choose_tile_refactor2. Returns the chosen tile index
// or -1 if no tile matches the corner constraints.
func (g *stbhwGen) chooseTile(tiles []stbhwTile, numTiles int,
	ax, ay, bx, by, cx, cy, dx, dy, ex, ey, fx, fy int, numVaryProduct uint32) int {
	a := g.getC(ax, ay)
	b := g.getC(bx, by)
	c := g.getC(cx, cy)
	d := g.getC(dx, dy)
	e := g.getC(ex, ey)
	f := g.getC(fx, fy)

	first := -1
	second := -1
	for i := 0; i < numTiles; i++ {
		h := &tiles[i]
		if (a < 0 || a == h.a) && (b < 0 || b == h.b) && (c < 0 || c == h.c) &&
			(d < 0 || d == h.d) && (e < 0 || e == h.e) && (f < 0 || f == h.f) {
			if first < 0 {
				first = i
			} else if second < 0 {
				second = i
				break
			}
		}
	}
	if first < 0 {
		return -1
	}
	stride := 0
	if second != -1 {
		stride = second - first
	}
	m := int(g.rand() % numVaryProduct)
	finalIdx := first + m*stride

	t := &tiles[finalIdx]
	g.setC(ax, ay, t.a)
	g.setC(bx, by, t.b)
	g.setC(cx, cy, t.c)
	g.setC(dx, dy, t.d)
	g.setC(ex, ey, t.e)
	g.setC(fx, fy, t.f)
	return finalIdx
}

func (g *stbhwGen) match(i, j int) bool {
	return g.getC(i, j) == g.getC(i+1, j+1)
}

// changeColor ports stbhw__change_color (no-weights branch; weights unused in Noita).
func (g *stbhwGen) changeColor(oldColor, numOptions int32) int32 {
	offset := 1 + int32(g.rand()%uint32(numOptions-1))
	return (oldColor + offset) % numOptions
}

func (g *stbhwGen) drawHTile(out []byte, stride, w, h, x, y int, tile *stbhwTile, sz int) {
	for j := 0; j < sz; j++ {
		if y+j >= 0 && y+j < h {
			for i := 0; i < sz*2; i++ {
				if x+i >= 0 && x+i < w {
					srcIdx := (j*(sz*2) + i) * 3
					dstIdx := (y+j)*stride + (x+i)*3
					out[dstIdx] = tile.pixels[srcIdx]
					out[dstIdx+1] = tile.pixels[srcIdx+1]
					out[dstIdx+2] = tile.pixels[srcIdx+2]
				}
			}
		}
	}
}

func (g *stbhwGen) drawVTile(out []byte, stride, w, h, x, y int, tile *stbhwTile, sz int) {
	for j := 0; j < sz*2; j++ {
		if y+j >= 0 && y+j < h {
			for i := 0; i < sz; i++ {
				if x+i >= 0 && x+i < w {
					srcIdx := (j*sz + i) * 3
					dstIdx := (y+j)*stride + (x+i)*3
					out[dstIdx] = tile.pixels[srcIdx]
					out[dstIdx+1] = tile.pixels[srcIdx+1]
					out[dstIdx+2] = tile.pixels[srcIdx+2]
				}
			}
		}
	}
}

// buildTilesetParams bundles the source image for parse_rect (mirrors the JS `p`).
type buildTilesetParams struct {
	ts     *stbhwTileset
	data   []byte
	stride int
	w, h   int
}

// buildTilesetFromImage ports stbhw_build_tileset_from_image (no override).
// data is the wang image as RGB bytes, stride bytes/row. Returns false on a bad
// header (matching the JS `return 0`).
func buildTilesetFromImage(ts *stbhwTileset, data []byte, stride, w, h int) bool {
	var header [9]byte
	for i := 0; i < 9; i++ {
		idx := (w * 3) - 1 - i
		if idx >= 0 && idx < len(data) {
			header[i] = data[idx] ^ byte((i*55)%256)
		}
	}

	if header[7] == 0xc0 {
		ts.isCorner = true
		ts.numColor = [6]int32{int32(header[0]), int32(header[1]), int32(header[2]), int32(header[3]), 0, 0}
		ts.numVaryX = int(header[4])
		ts.numVaryY = int(header[5])
		ts.shortSideLen = int(header[6])
	} else {
		ts.isCorner = false
		ts.numColor = [6]int32{int32(header[0]), int32(header[1]), int32(header[2]), int32(header[3]), int32(header[4]), int32(header[5])}
		ts.numVaryX = int(header[6])
		ts.numVaryY = int(header[7])
		ts.shortSideLen = int(header[8])
	}

	if ts.shortSideLen <= 0 || ts.shortSideLen > 100 {
		return false
	}

	p := &buildTilesetParams{ts: ts, data: data, stride: stride, w: w, h: h}
	ypos := 2
	ts.hTiles = nil
	ts.vTiles = nil
	ts.numHTiles = 0
	ts.numVTiles = 0

	if ts.isCorner {
		for k := int32(0); k < ts.numColor[2]; k++ {
			for j := int32(0); j < ts.numColor[1]; j++ {
				for i := int32(0); i < ts.numColor[0]; i++ {
					for q := 0; q < ts.numVaryY; q++ {
						processHRow(p, 0, ypos, 0, ts.numColor[1]-1, 0, ts.numColor[2]-1, 0, ts.numColor[3]-1, i, i, j, j, k, k, ts.numVaryX)
						ypos += ts.shortSideLen + 3
					}
				}
			}
		}
		ypos += 2
		for k := int32(0); k < ts.numColor[3]; k++ {
			for j := int32(0); j < ts.numColor[0]; j++ {
				for i := int32(0); i < ts.numColor[1]; i++ {
					for q := 0; q < ts.numVaryX; q++ {
						processVRow(p, 0, ypos, 0, ts.numColor[0]-1, 0, ts.numColor[3]-1, 0, ts.numColor[2]-1, i, i, j, j, k, k, ts.numVaryY)
						ypos += ts.shortSideLen*2 + 3
					}
				}
			}
		}
	} else {
		for k := int32(0); k < ts.numColor[3]; k++ {
			for j := int32(0); j < ts.numColor[4]; j++ {
				for i := int32(0); i < ts.numColor[2]; i++ {
					for q := 0; q < ts.numVaryY; q++ {
						processHRow(p, 0, ypos, 0, ts.numColor[2]-1, k, k, 0, ts.numColor[1]-1, j, j, 0, ts.numColor[0]-1, i, i, ts.numVaryX)
						ypos += ts.shortSideLen + 3
					}
				}
			}
		}
		ypos += 2
		for k := int32(0); k < ts.numColor[3]; k++ {
			for j := int32(0); j < ts.numColor[4]; j++ {
				for i := int32(0); i < ts.numColor[5]; i++ {
					for q := 0; q < ts.numVaryX; q++ {
						processVRow(p, 0, ypos, 0, ts.numColor[0]-1, i, i, 0, ts.numColor[1]-1, j, j, 0, ts.numColor[5]-1, k, k, ts.numVaryY)
						ypos += ts.shortSideLen*2 + 3
					}
				}
			}
		}
	}

	return true
}

func processHRow(p *buildTilesetParams, xpos, ypos int, a0, a1, b0, b1, c0, c1, d0, d1, e0, e1, f0, f1 int32, variants int) {
	for v := 0; v < variants; v++ {
		for f := f0; f <= f1; f++ {
			for e := e0; e <= e1; e++ {
				for d := d0; d <= d1; d++ {
					for c := c0; c <= c1; c++ {
						for b := b0; b <= b1; b++ {
							for a := a0; a <= a1; a++ {
								parseRect(p, xpos, ypos, a, b, c, d, e, f, false)
								xpos += 2*p.ts.shortSideLen + 3
							}
						}
					}
				}
			}
		}
	}
}

func processVRow(p *buildTilesetParams, xpos, ypos int, a0, a1, b0, b1, c0, c1, d0, d1, e0, e1, f0, f1 int32, variants int) {
	for v := 0; v < variants; v++ {
		for f := f0; f <= f1; f++ {
			for e := e0; e <= e1; e++ {
				for d := d0; d <= d1; d++ {
					for c := c0; c <= c1; c++ {
						for b := b0; b <= b1; b++ {
							for a := a0; a <= a1; a++ {
								parseRect(p, xpos, ypos, a, b, c, d, e, f, true)
								xpos += p.ts.shortSideLen + 3
							}
						}
					}
				}
			}
		}
	}
}

func parseRect(p *buildTilesetParams, xpos, ypos int, a, b, c, d, e, f int32, isV bool) {
	len_ := p.ts.shortSideLen
	wT := len_ * 2
	hT := len_
	if isV {
		wT = len_
		hT = len_ * 2
	}
	pixels := make([]byte, wT*hT*3)
	xpos++
	ypos++
	for j := 0; j < hT; j++ {
		for i := 0; i < wT; i++ {
			start := (ypos+j)*p.stride + (xpos+i)*3
			dest := (j*wT + i) * 3
			if start+2 < len(p.data) {
				pixels[dest] = p.data[start]
				pixels[dest+1] = p.data[start+1]
				pixels[dest+2] = p.data[start+2]
			}
		}
	}
	t := stbhwTile{pixels: pixels, a: a, b: b, c: c, d: d, e: e, f: f}
	if isV {
		p.ts.vTiles = append(p.ts.vTiles, t)
		p.ts.numVTiles++
	} else {
		p.ts.hTiles = append(p.ts.hTiles, t)
		p.ts.numHTiles++
	}
}
