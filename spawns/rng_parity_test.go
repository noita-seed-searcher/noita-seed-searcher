package main

import (
	"encoding/json"
	"os"
	"testing"
)

// parityVector mirrors one record emitted by parity_gen.mjs (reference JS).
type parityVector struct {
	WS            uint32    `json:"ws"`
	X             float64   `json:"x"`
	Y             float64   `json:"y"`
	SeedAfterSet  float64   `json:"seedAfterSet"`
	Proc          float64   `json:"proc"`
	SeedAfterNext float64   `json:"seedAfterNext"`
	Pri           int       `json:"pri"`
	Seq           []float64 `json:"seq"`
	USeq          []uint32  `json:"useq"`
	SrfwsSeed     float64   `json:"srfwsSeed"`
	SrfwsSeq      []float64 `json:"srfwsSeq"`
}

// TestPRNGParity verifies spawns/rng.go matches the reference nolla_prng.js
// bit-for-bit across the vectors in parity_vectors.json. Regenerate with:
//
//	node parity_gen.mjs > parity_vectors.json
func TestPRNGParity(t *testing.T) {
	data, err := os.ReadFile("parity_vectors.json")
	if err != nil {
		t.Fatalf("read vectors (run `node parity_gen.mjs > parity_vectors.json`): %v", err)
	}
	var vecs []parityVector
	if err := json.Unmarshal(data, &vecs); err != nil {
		t.Fatalf("unmarshal vectors: %v", err)
	}
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}

	for _, v := range vecs {
		// ProceduralRandom: SetRandomSeed then next(). Check the integer
		// seed state after seeding (exact), the returned float, and the
		// integer seed after the draw.
		p := &NollaPrng{}
		p.setRandomSeed(v.WS, v.X, v.Y)
		if got := p.seed; got != v.SeedAfterSet {
			t.Errorf("ws=%d x=%g y=%g: seedAfterSet = %.0f, want %.0f", v.WS, v.X, v.Y, got, v.SeedAfterSet)
			continue
		}
		proc := p.next()
		if proc != v.Proc {
			t.Errorf("ws=%d x=%g y=%g: proc = %v, want %v", v.WS, v.X, v.Y, proc, v.Proc)
		}
		if p.seed != v.SeedAfterNext {
			t.Errorf("ws=%d x=%g y=%g: seedAfterNext = %.0f, want %.0f", v.WS, v.X, v.Y, p.seed, v.SeedAfterNext)
		}

		// ProceduralRandomi(ws,x,y,0,100) == setRandomSeed + random(0,100).
		pi := &NollaPrng{}
		pi.setRandomSeed(v.WS, v.X, v.Y)
		if got := pi.random(0, 100); got != v.Pri {
			t.Errorf("ws=%d x=%g y=%g: proceduralRandomi = %d, want %d", v.WS, v.X, v.Y, got, v.Pri)
		}

		// Fresh seed, then a sequence of next() draws.
		ps := &NollaPrng{}
		ps.setRandomSeed(v.WS, v.X, v.Y)
		for i, want := range v.Seq {
			if got := ps.next(); got != want {
				t.Errorf("ws=%d x=%g y=%g: seq[%d] = %v, want %v", v.WS, v.X, v.Y, i, got, want)
			}
		}

		// Fresh seed, then a sequence of nextU() draws.
		pu := &NollaPrng{}
		pu.setRandomSeed(v.WS, v.X, v.Y)
		for i, want := range v.USeq {
			if got := pu.nextU(); got != want {
				t.Errorf("ws=%d x=%g y=%g: useq[%d] = %d, want %d", v.WS, v.X, v.Y, i, got, want)
			}
		}

		// setRandomFromWorldSeed then 3×nextU + 3×next (the tiler reseed dance).
		pw := &NollaPrng{}
		pw.setRandomFromWorldSeed(float64(v.WS))
		if pw.seed != v.SrfwsSeed {
			t.Errorf("ws=%d: srfwsSeed = %.0f, want %.0f", v.WS, pw.seed, v.SrfwsSeed)
		}
		for i, want := range v.SrfwsSeq {
			var got float64
			if i < 3 {
				got = float64(pw.nextU())
			} else {
				got = pw.next()
			}
			if got != want {
				t.Errorf("ws=%d: srfwsSeq[%d] = %v, want %v", v.WS, i, got, want)
			}
		}
	}
}
