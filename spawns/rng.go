package main

import (
	"math"
)

// NollaPrng is Noita's LCG-based PRNG, ported from nolla_prng.js.
type NollaPrng struct {
	seed float64
}

func newPrng(seed float64) *NollaPrng {
	p := &NollaPrng{seed: seed}
	p.next()
	return p
}

func (p *NollaPrng) next() float64 {
	s := int32(p.seed)
	v4 := int32(16807)*s - int32(2147483647)*int32(s/127773)
	if v4 <= 0 {
		v4 += 2147483647
	}
	p.seed = float64(v4)
	return p.seed / 2147483647.0
}

// random returns int in [a, b] inclusive.
func (p *NollaPrng) random(a, b int) int {
	return a + int(float64(b+1-a)*p.next())
}

func truncF64ToU32(v float64) uint32 {
	return uint32(int64(v))
}

func rngHelper(v float64) uint32 {
	if v == 0 {
		return 2
	}
	bits := math.Float64bits(v)
	finite := ((bits>>32)&0x7fffffff) < 0x7ff00000
	inRange := v >= -9.223372036854776e18 && v < 9.223372036854776e18
	if !finite || !inRange {
		return 0
	}
	return truncF64ToU32(v)
}

func rngHelper2(a, b, ws uint32) uint32 {
	u2 := ((a - b - ws) ^ (ws >> 13))
	u1 := ((b - u2 - ws) ^ (u2 << 8))
	u3 := ((ws - u2 - u1) ^ (u1 >> 13))
	u2 = ((u2 - u1 - u3) ^ (u3 >> 12))
	u1 = ((u1 - u2 - u3) ^ (u2 << 16))
	u3 = ((u3 - u2 - u1) ^ (u1 >> 5))
	u2 = ((u2 - u1 - u3) ^ (u3 >> 3))
	u1 = ((u1 - u2 - u3) ^ (u2 << 10))
	return ((u3 - u2 - u1) ^ (u1 >> 15))
}

func (p *NollaPrng) setRandomSeed(ws uint32, x, y float64) {
	a := ws ^ 0x93262e6f
	b := a & 0xfff
	c := (a >> 12) & 0xfff
	xAdj := x + float64(b)
	yAdj := y + float64(c)

	r := xAdj * 134217727.0
	e := rngHelper(r)

	absX := math.Abs(xAdj)
	absY := math.Abs(yAdj)
	if absY >= 102400.0 || absX <= 1.0 {
		r = yAdj * 134217727.0
	} else {
		yWork := yAdj*3483.328 + float64(e)
		r = yAdj * yWork
	}
	f := rngHelper(r)
	g := rngHelper2(e, f, ws)

	diddle := [17]uint32{0, 4, 6, 25, 12, 39, 52, 9, 21, 64, 78, 92, 104, 118, 18, 32, 44}
	const magic uint32 = 252645135

	t := g
	if g < 0x80000000 {
		t++
	}
	if g == 0 {
		t++
	}
	t -= g / magic
	idx := g / magic
	if idx < 17 && g%magic < diddle[idx] && (g < 0xc3c3c3c3+4 || g >= 0xc3c3c3c3+62) {
		t++
	}
	if g > 0x80000000 {
		t++
	}
	t >>= 1
	if g == 0xffffffff {
		t++
	}

	p.seed = float64(t)
	p.next()

	h := ws & 3
	for h > 0 {
		p.next()
		h--
	}
}

// proceduralRandom matches ProceduralRandom(ws, x, y) from JS.
func (p *NollaPrng) proceduralRandom(ws uint32, x, y float64) float64 {
	p.setRandomSeed(ws, x, y)
	return p.next()
}

func (p *NollaPrng) getDistribution(mean, sharpness, baseline float64) float64 {
	const pi = 3.1415
	for i := 0; i < 100; i++ {
		r1 := p.next()
		r2 := p.next()
		div := math.Abs(r1 - mean)
		if r2 < (1.0-div)*baseline {
			return r1
		}
		if div < 0.5 {
			v11 := math.Sin(((0.5 - mean) + r1) * pi)
			v12 := math.Pow(v11, sharpness)
			if v12 > r2 {
				return r1
			}
		}
	}
	return p.next()
}

// randomDistribution matches RandomDistribution(min, max, mean, sharpness) from JS.
func (p *NollaPrng) randomDistribution(min, max, mean int, sharpness float64) int {
	if sharpness == 0 {
		return p.random(min, max)
	}
	adjMean := float64(mean-min) / float64(max-min)
	v7 := p.getDistribution(adjMean, sharpness, 0.005)
	d := jsRound(v7 * float64(max-min))
	return min + d
}

// randomDistributionF matches RandomDistributionF(min, max, mean, sharpness) from JS.
func (p *NollaPrng) randomDistributionF(min, max, mean, sharpness float64) float64 {
	if sharpness == 0 {
		return min + (max-min)*p.next()
	}
	adjMean := (mean - min) / (max - min)
	v7 := p.getDistribution(adjMean, sharpness, 0.005)
	return min + v7*(max-min)
}

// jsRound matches JavaScript's Math.round (ties round toward +inf).
func jsRound(x float64) int {
	return int(math.Floor(x + 0.5))
}

// roundHalfToEven matches roundHalfOfEven from JS utils.
func roundHalfToEven(n float64) int {
	frac := n - math.Floor(n)
	if frac == 0.5 {
		floor := int(math.Floor(n))
		if floor%2 == 0 {
			return floor
		}
		return floor + 1
	}
	return jsRound(n)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func shuffleTable(arr []string, p *NollaPrng) {
	for i := len(arr) - 1; i >= 1; i-- {
		j := p.random(0, i)
		arr[i], arr[j] = arr[j], arr[i]
	}
}

func roundRNGPos(x float64) float64 {
	if x > -1000000 && x < 1000000 {
		return x
	}
	if x > -10000000 && x < 10000000 {
		return float64(roundHalfToEven(x/10.0)) * 10
	}
	if x > -100000000 && x < 100000000 {
		return float64(roundHalfToEven(x/100.0)) * 100
	}
	return x
}
