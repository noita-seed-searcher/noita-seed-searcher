// Generates PRNG parity test vectors from the reference noita-telescope JS.
// Output: JSON array on stdout, consumed by rng_parity_test.go.
import { NollaPrng } from "/home/legrems/Documents/Games/noita/noita-telescope/js/nolla_prng.js";

// Build a set of (ws, x, y) cases covering normal + edge inputs.
const seeds = [1, 42, 123456789, 2147483646, 0x93262e6f, 999999937, 7, 2000000000];
const baseCoords = [
  [0, 0], [1, 0], [-1, 0], [2, 2], [100, 50], [315, 119],
  [512, 14], [-512, -14], [3584, 1000], [12345, -6789],
  [0, 102400], [0, 102399], [0, 200000], [-0.5, 0.5],
  [1.0, 1.0], [-1.0, -1.0], [0.25, -0.25], [35840, 24576],
];

function caseFor(ws, x, y) {
  // Edge-case probe: also force x_ == 0 (so r == 0) for this ws.
  const p = new NollaPrng(0);
  p.Seed = 0; // reset; constructor already called Next once, normalize
  // ProceduralRandom: SetRandomSeed then Next.
  const p2 = new NollaPrng(0);
  p2.SetRandomSeed(ws >>> 0, x, y);
  const seedAfterSet = p2.Seed; // integer state after seeding
  const proc = p2.Next();        // ProceduralRandom return value
  const seedAfterNext = p2.Seed;

  // ProceduralRandomi(ws,x,y,0,100)
  const p3 = new NollaPrng(0);
  const pri = p3.ProceduralRandomi(ws >>> 0, x, y, 0, 100);

  // A short Next() sequence from the seeded state (re-seed fresh).
  const p4 = new NollaPrng(0);
  p4.SetRandomSeed(ws >>> 0, x, y);
  const seq = [];
  for (let i = 0; i < 4; i++) seq.push(p4.Next());

  // NextU() sequence from the seeded state (used by the Wang-tile generator).
  const p5 = new NollaPrng(0);
  p5.SetRandomSeed(ws >>> 0, x, y);
  const useq = [];
  for (let i = 0; i < 4; i++) useq.push(p5.NextU());

  // SetRandomFromWorldSeed(ws) then mixed draws (the tiler's reseed dance).
  const p6 = new NollaPrng(0);
  p6.SetRandomFromWorldSeed(ws);
  const srfwsSeed = p6.Seed;
  const srfwsSeq = [];
  for (let i = 0; i < 3; i++) srfwsSeq.push(p6.NextU());
  for (let i = 0; i < 3; i++) srfwsSeq.push(p6.Next());

  return {
    ws: ws >>> 0, x, y,
    seedAfterSet,
    proc,
    seedAfterNext,
    pri,
    seq,
    useq,
    srfwsSeed,
    srfwsSeq,
  };
}

const out = [];
for (const ws of seeds) {
  for (const [x, y] of baseCoords) {
    out.push(caseFor(ws, x, y));
  }
  // r==0 edge case: x_ = x + b == 0  =>  x = -b
  const b = (ws ^ 0x93262e6f) >>> 0 & 0xfff;
  out.push(caseFor(ws, -b, 0));
  out.push(caseFor(ws, -b, 5));
}

process.stdout.write(JSON.stringify(out, null, 0));
