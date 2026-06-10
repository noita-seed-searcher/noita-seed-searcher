package main

import (
	"math"
)

const (
	lcgModulus    = 0x7fffffff
	lcgMultiplier = 0x41a7
	lcgDivisor    = 0x1f31d
	u32MaxHalf    = 0x80000000
)

// NollaPrng is Noita's LCG-based PRNG, ported from nolla_prng.zig.
type NollaPrng struct {
	seed float64
}

func newNollaPrng(seed float64) NollaPrng {
	rng := NollaPrng{seed: seed}
	rng.next()
	return rng
}

func (rng *NollaPrng) next() float64 {
	seedInt := int32(rng.seed)
	nextVal := int32(lcgMultiplier)*seedInt - int32(lcgModulus)*int32(int32(seedInt)/int32(lcgDivisor))
	if nextVal <= 0 {
		nextVal += lcgModulus
	}
	rng.seed = float64(nextVal)
	return rng.seed / lcgModulus
}

func (rng *NollaPrng) random(min, max int32) int32 {
	r := float64(max+1-min) * rng.next()
	return min + int32(r)
}

func (rng *NollaPrng) setRandomFromWorldSeed(seed uint32) {
	rng.seed = float64(seed)
	if rng.seed >= 2147483647.0 {
		rng.seed = float64(seed) * 0.5
	}
}

func truncDoubleToU32(value float64) uint32 {
	truncated := int64(value)
	return uint32(uint64(truncated))
}

func setRandomSeedHelper(value float64) uint32 {
	bits := math.Float64bits(value)
	finite := ((bits>>0x20)&0x7fffffff) < 0x7ff00000
	inRange := -9.223372036854776e18 <= value && value < 9.223372036854776e18
	if !finite || !inRange {
		return 0
	}
	return truncDoubleToU32(value)
}

func setRandomSeedHelper2(a, b, ws uint32) uint32 {
	v2 := ((a - b - ws) ^ (ws >> 0xd))
	v1 := ((b - v2 - ws) ^ (v2 << 8))
	v3 := ((ws - v2 - v1) ^ (v1 >> 0xd))
	v2 = ((v2 - v1 - v3) ^ (v3 >> 0xc))
	v1 = ((v1 - v2 - v3) ^ (v2 << 0x10))
	v3 = ((v3 - v2 - v1) ^ (v1 >> 5))
	v2 = ((v2 - v1 - v3) ^ (v3 >> 3))
	v1 = ((v1 - v2 - v3) ^ (v2 << 10))
	return ((v3 - v2 - v1) ^ (v1 >> 0xf))
}

func (rng *NollaPrng) setRandomSeed(ws uint32, x, y float64) {
	a := ws ^ 0x93262e6f
	b := a & 0xfff
	c := (a >> 0xc) & 0xfff

	xAdjusted := x + float64(b)
	yAdjusted := y + float64(c)

	seedMaterial := xAdjusted * 134217727.0
	e := setRandomSeedHelper(seedMaterial)

	if math.Abs(yAdjusted) >= 102400.0 || math.Abs(xAdjusted) <= 1.0 {
		seedMaterial = yAdjusted * 134217727.0
	} else {
		yWork := yAdjusted * 3483.328
		yWork += float64(e)
		yAdjusted *= yWork
		seedMaterial = yAdjusted
	}

	f := setRandomSeedHelper(seedMaterial)
	g := setRandomSeedHelper2(e, f, ws)

	diddleTable := [17]uint32{0, 4, 6, 25, 12, 39, 52, 9, 21, 64, 78, 92, 104, 118, 18, 32, 44}
	const magicNumber uint32 = 252645135

	t := g
	if g < u32MaxHalf {
		t++
	}
	if g == 0 {
		t++
	}
	t -= g / magicNumber
	idx := g / magicNumber
	var diddleAdd uint32
	if idx < 17 && g%magicNumber < diddleTable[idx] && (g < 0xc3c3c3c3+4 || g >= 0xc3c3c3c3+62) {
		diddleAdd = 1
	}
	t += diddleAdd
	if g > u32MaxHalf {
		t++
	}
	t >>= 1
	if g == 0xffffffff {
		t++
	}

	rng.seed = float64(t)
	rng.next()

	h := ws & 3
	for h != 0 {
		rng.next()
		h--
	}
}

// RandomPos mirrors the Zig RandomPos struct for procedural random.
type RandomPos struct {
	x, y int32
}

func randomNextF(worldSeed uint32, pos *RandomPos, min, max float64) float64 {
	var rng NollaPrng
	rng.setRandomSeed(worldSeed, float64(pos.x), float64(pos.y))
	result := min + (max-min)*rng.next()
	pos.y++
	return result
}

func randomNextI(worldSeed uint32, pos *RandomPos, min, max int32) int32 {
	var rng NollaPrng
	rng.setRandomSeed(worldSeed, float64(pos.x), float64(pos.y))
	result := rng.random(min, max)
	pos.y++
	return result
}

// RNG is the stateful world seed RNG used by providers, mirroring rng.zig.
type RNG struct {
	worldSeed uint32
	rng       NollaPrng
}

func newRNG() *RNG {
	return &RNG{}
}

func (r *RNG) SetWorldSeed(seed uint32) {
	r.worldSeed = seed
}

func (r *RNG) GetWorldSeed() uint32 {
	return r.worldSeed
}

func (r *RNG) SetRandomSeed(x, y float64) {
	r.rng.setRandomSeed(r.worldSeed, x, y)
}

// RandomInt returns a random int in [min, max] inclusive.
func (r *RNG) RandomInt(min, max int32) int32 {
	return r.rng.random(min, max)
}

// Randomf returns the next float in [0, 1).
func (r *RNG) Randomf() float64 {
	return r.rng.next()
}

// RandomRounded rounds min/max to even and returns random int in range.
func (r *RNG) RandomRounded(min, max float64) int32 {
	return r.rng.random(roundHalfToEvenI32(min), roundHalfToEvenI32(max))
}

// RandomMax returns a random int in [0, roundHalfToEven(max)].
func (r *RNG) RandomMax(max float64) int32 {
	return r.rng.random(0, roundHalfToEvenI32(max))
}

// ProceduralRandomf sets seed from (x,y), returns float in [min, max].
func (r *RNG) ProceduralRandomf(x, y, min, max float64) float64 {
	r.rng.setRandomSeed(r.worldSeed, x, y)
	return min + (max-min)*r.rng.next()
}

// ProceduralRandomi sets seed from (x,y), returns int in [min, max].
func (r *RNG) ProceduralRandomi(x, y, min, max float64) int32 {
	r.rng.setRandomSeed(r.worldSeed, x, y)
	return r.rng.random(roundHalfToEvenI32(min), roundHalfToEvenI32(max))
}

// SeededRandom returns a one-off random value from a specific seed and position.
func (r *RNG) SeededRandom(seed uint32, x, y float64) float64 {
	var local NollaPrng
	local.setRandomSeed(seed, x, y)
	return local.next()
}

// RoundHalfOfEven implements banker's rounding (round half to even).
func roundHalfToEvenF32(value float32) float32 {
	floorVal := float32(math.Floor(float64(value)))
	diff := value - floorVal
	if diff < 0.5 {
		return floorVal
	}
	if diff > 0.5 {
		return floorVal + 1.0
	}
	floorInt := int64(floorVal)
	if floorInt%2 == 0 {
		return floorVal
	}
	return floorVal + 1.0
}

func roundHalfToEvenI32(value float64) int32 {
	return int32(roundHalfToEvenF32(float32(value)))
}

// randomCreate mirrors random_create from random.ts.
func randomCreate(x, y int32) RandomPos {
	return RandomPos{x: x, y: y}
}

// randomNext mirrors random_next from random.ts (uses ProceduralRandomf pattern).
func (r *RNG) randomNext(pos *RandomPos, min, max float64) float64 {
	return randomNextF(r.worldSeed, pos, min, max)
}

// randomNextI mirrors random_nexti from random.ts.
func (r *RNG) randomNextI(pos *RandomPos, min, max int32) int32 {
	return randomNextI(r.worldSeed, pos, min, max)
}

// pickRandomFromTableBackwards mirrors pick_random_from_table_backwards.
// items must have a Chance() method.
type Chanced interface {
	GetChance() float64
}

func pickRandomFromTableBackwardsIdx(chances []float64, pos *RandomPos, worldSeed uint32) int {
	result := 0
	for i := len(chances) - 1; i >= 0; i-- {
		val := randomNextF(worldSeed, pos, 0.0, 1.0)
		if val <= chances[i] {
			result = i
			break
		}
	}
	return result
}

// pickRandomFromTableWeighted mirrors pick_random_from_table_weighted.
func pickRandomFromTableWeightedIdx(probabilities []float64, pos *RandomPos, worldSeed uint32) int {
	weightSum := 0.0
	for _, p := range probabilities {
		weightSum += p
	}
	val := randomNextF(worldSeed, pos, 0.0, weightSum)
	min := 0.0
	for i, p := range probabilities {
		max := min + p
		if val >= min && val <= max {
			return i
		}
		min = max
	}
	return 0
}
