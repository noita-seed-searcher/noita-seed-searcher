package main

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	_ "image/png"
)

//go:embed data/wang_tiles/coalmine.png
var wangFS embed.FS

// wangFileFor maps a biome name to its embedded Wang-tile asset. Extended as
// more biomes are ported; coalmine (and its tower variant) is enough for the
// first level.
func wangFileFor(biome string) (string, bool) {
	switch biome {
	case "coalmine", "solid_wall_tower_1":
		return "data/wang_tiles/coalmine.png", true
	default:
		return "", false
	}
}

// loadWangRGB decodes an embedded Wang PNG to packed RGB bytes (w*h*3),
// matching getCachedTileset's RGBA->RGB conversion in tile_generator.js.
func loadWangRGB(file string) (rgb []byte, w, h int, err error) {
	data, err := wangFS.ReadFile(file)
	if err != nil {
		return nil, 0, 0, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}
	b := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, b.Min, draw.Src)
	w, h = b.Dx(), b.Dy()
	rgb = make([]byte, w*h*3)
	for i := 0; i < w*h; i++ {
		rgb[i*3] = nrgba.Pix[i*4]
		rgb[i*3+1] = nrgba.Pix[i*4+1]
		rgb[i*3+2] = nrgba.Pix[i*4+2]
	}
	return rgb, w, h, nil
}

// buildBiomeTileset decodes the Wang PNG for a biome and builds its tileset.
func buildBiomeTileset(file string) (*stbhwTileset, error) {
	rgb, w, h, err := loadWangRGB(file)
	if err != nil {
		return nil, err
	}
	ts := &stbhwTileset{}
	buildTilesetFromImage(ts, rgb, w*3, w, h)
	return ts, nil
}

type mapDims struct{ w, h int }

// calculateMapDimensions ports calculateMapDimensions: chunks at index%5==4
// get one extra pixel of width/height.
func calculateMapDimensions(bbox [4]int) mapDims {
	minX, minY, maxX, maxY := bbox[0], bbox[1], bbox[2], bbox[3]
	w := 0
	for x := minX; x <= maxX; x++ {
		w += 51
		if x%5 == 4 {
			w++
		}
	}
	h := 0
	for y := minY; y <= maxY; y++ {
		h += 51
		if y%5 == 4 {
			h++
		}
	}
	return mapDims{w, h}
}

type point struct{ X, Y int }

// findBiomeRegions ports findBiomeRegions: flood-fill connected components of
// targetColor, merging any region fully contained in an earlier region's bbox.
func findBiomeRegions(pixels []uint32, width, height int, targetColor uint32) (regions [][]point, bboxes [][4]int) {
	visited := make([]bool, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if !visited[idx] && pixels[idx] == targetColor {
				regionPoints := []point{}
				queue := []point{{x, y}}
				visited[idx] = true
				minX, maxX, minY, maxY := width, 0, height, 0

				for head := 0; head < len(queue); head++ {
					cur := queue[head]
					cx, cy := cur.X, cur.Y
					regionPoints = append(regionPoints, cur)
					if cx < minX {
						minX = cx
					}
					if cx > maxX {
						maxX = cx
					}
					if cy < minY {
						minY = cy
					}
					if cy > maxY {
						maxY = cy
					}
					neighbors := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
					for _, d := range neighbors {
						nx, ny := cx+d[0], cy+d[1]
						if nx >= 0 && nx < width && ny >= 0 && ny < height {
							nIdx := ny*width + nx
							if !visited[nIdx] && pixels[nIdx] == targetColor {
								visited[nIdx] = true
								queue = append(queue, point{nx, ny})
							}
						}
					}
				}

				valid := true
				for i := range regions {
					r := bboxes[i]
					if minX >= r[0] && maxX >= r[0] && minY >= r[1] && maxY >= r[1] &&
						minX <= r[2] && maxX <= r[2] && minY <= r[3] && maxY <= r[3] {
						// Fully contained: merge into the existing region.
						regions[i] = append(regions[i], regionPoints...)
						valid = false
					}
				}
				if valid {
					regions = append(regions, regionPoints)
					bboxes = append(bboxes, [4]int{minX, minY, maxX, maxY})
				}
			}
		}
	}
	return regions, bboxes
}

// rawTile is the pre-hack output of generateRawTileBuffer: the rendered tile
// buffer plus tile-index grid and metadata. Biome hacks / pathfinding /
// masking (a later stage) operate on this.
type rawTile struct {
	buffer      []byte
	width       int
	height      int
	tileIndices []int32
	xmax, ymax  int
	tileSize    int
	numHTiles   int
	numVTiles   int
	minX, minY  int
	mapH        int
}

// generateRawTileBuffer ports the pre-hack portion of generateRawTileBuffer:
// dimensions, the world-seed reseed dance, and stbhw_generate_image. The biome
// hacks that follow in the JS are deferred to a later stage.
func generateRawTileBuffer(bbox [4]int, ts *stbhwTileset, worldSeed uint32, ngPlus, extraRerolls int) *rawTile {
	minX, minY := bbox[0], bbox[1]
	dims := calculateMapDimensions(bbox)
	mapW, mapH := dims.w, dims.h
	outH := mapH + 4

	if len(ts.hTiles) == 0 {
		return nil
	}

	prng := &NollaPrng{}
	g := newStbhwGen(prng)
	prng.setRandomFromWorldSeed(float64(worldSeed) + float64(ngPlus))
	prng.next()

	wsng := int64(worldSeed) + int64(ngPlus)
	iters := mapW + int(wsng) - 11*(mapW/11) - 12*int(wsng/12)
	for i := 0; i < iters; i++ {
		prng.next()
	}
	for i := 0; i < extraRerolls; i++ {
		prng.next()
	}

	prng.seed = float64(prng.nextU())
	prng.next()

	rawBuffer := make([]byte, mapW*outH*3)
	tileIndices, xmax, ymax, ok := g.stbhwGenerateImage(ts, rawBuffer, mapW*3, mapW, outH)
	if !ok {
		return nil
	}

	return &rawTile{
		buffer:      rawBuffer,
		width:       mapW,
		height:      outH,
		tileIndices: tileIndices,
		xmax:        xmax,
		ymax:        ymax,
		tileSize:    ts.shortSideLen,
		numHTiles:   ts.numHTiles,
		numVTiles:   ts.numVTiles,
		minX:        minX,
		minY:        minY,
		mapH:        mapH,
	}
}
