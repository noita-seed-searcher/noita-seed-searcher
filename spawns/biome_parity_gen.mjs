// Generates biome-map parity vectors from the reference noita-telescope JS.
// generateBiomeData only needs shuffleTable from utils.js, but utils.js pulls in
// a heavy import chain (png_sanitizer -> https CDN) that breaks under node. So we
// copy the two real source files verbatim into ./parity_ref/ next to a minimal
// utils.js stub (shuffleTable only), then import the copy. Logic stays verbatim.
// Output: biome_vectors.json (consumed by biome_parity_test.go).
import { readFileSync, writeFileSync, mkdirSync, copyFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const HERE = dirname(fileURLToPath(import.meta.url));
const TELE = "/home/legrems/Documents/Games/noita/noita-telescope/js";
const REF = join(HERE, "parity_ref");

mkdirSync(REF, { recursive: true });
copyFileSync(join(TELE, "biome_generator.js"), join(REF, "biome_generator.js"));
copyFileSync(join(TELE, "nolla_prng.js"), join(REF, "nolla_prng.js"));
// Minimal utils.js stub: only shuffleTable, copied verbatim from utils.js:141.
writeFileSync(
  join(REF, "utils.js"),
  `export function shuffleTable(arr, prng) {
    for (let i = arr.length - 1; i >= 1; i--) {
        let j = prng.Random(0, i);
        let temp = arr[i];
        arr[i] = arr[j];
        arr[j] = temp;
    }
}
`
);

const { generateBiomeData } = await import("./parity_ref/biome_generator.js");

const base = JSON.parse(readFileSync(join(HERE, "biome_base.json")));

// Mirror app.js / Go selectBiomeBase: which base map + dims per (mode, ng).
function selectBase(gameMode, ng) {
  const isNGP = ng > 0;
  if (isNGP) return base.ngp;
  if (gameMode === "nightmare") return base.nightmare;
  return base.normal;
}

// 32-bit rolling hash (FNV-1a-ish), replicated in Go for heaven/hell checks.
function hash32(arr) {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < arr.length; i++) {
    h = (h ^ (arr[i] & 0xffffffff)) >>> 0;
    h = Math.imul(h, 16777619) >>> 0;
  }
  return h >>> 0;
}

const CASES = [
  { seed: 123456789, ng: 0, gameMode: "normal" },   // static NG0
  { seed: 42, ng: 0, gameMode: "nightmare" },        // procedural nightmare
  { seed: 123456789, ng: 1, gameMode: "normal" },    // NGP, ng%7
  { seed: 999999937, ng: 2, gameMode: "normal" },    // tower (ng%2)
  { seed: 7, ng: 3, gameMode: "normal" },            // shuffle (ng%3), hell=47
  { seed: 2000000000, ng: 5, gameMode: "normal" },   // doWalls (ng%5)
  { seed: 42, ng: 6, gameMode: "normal" },           // tower+shuffle (ng%6)
  { seed: 100, ng: 7, gameMode: "normal" },          // color replace (ng%7)
  { seed: 100, ng: 30, gameMode: "normal" },         // ng>=25, ng%2,3,5,6
  { seed: 4294967295, ng: 1, gameMode: "normal" },   // max uint32 seed
];

const out = [];
for (const c of CASES) {
  const b = selectBase(c.gameMode, c.ng);
  const res = generateBiomeData(c.seed, c.ng, c.gameMode, b.rgba, b.w, b.h);
  out.push({
    ...c,
    w: b.w,
    h: b.h,
    pixels: Array.from(res.pixels),
    orbs: res.orbs.map((o) => ({ x: o.x, y: o.y, name: o.name })),
    heavenHash: hash32(res.heavenPixels),
    hellHash: hash32(res.hellPixels),
  });
}

writeFileSync(join(HERE, "biome_vectors.json"), JSON.stringify(out));
console.log(`wrote biome_vectors.json: ${out.length} cases`);
