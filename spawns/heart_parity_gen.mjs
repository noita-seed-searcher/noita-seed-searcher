// Generates spawnHeart routing parity vectors from the reference JS.
// Imports VERBATIM heart_generation.js (copied into parity_ref/) with stubbed
// chest_generation.js (generateChest/generateGreatChest -> markers) and
// settings.js (non-Valentine date), so we isolate the heart ROUTING decision
// (heart / chest / great_chest / mimic / chest_leggy / none) from chest contents
// (which the existing Go generators already produce).
// Output: heart_vectors.json (consumed by heart_parity_test.go).
import { writeFileSync, mkdirSync, copyFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const HERE = dirname(fileURLToPath(import.meta.url));
const TELE = "/home/legrems/Documents/Games/noita/noita-telescope/js";
const REF = join(HERE, "parity_ref");

mkdirSync(REF, { recursive: true });
copyFileSync(join(TELE, "heart_generation.js"), join(REF, "heart_generation.js"));
copyFileSync(join(TELE, "nolla_prng.js"), join(REF, "nolla_prng.js"));
writeFileSync(join(REF, "chest_generation.js"), `
export function generateChest() { return { type: 'chest' }; }
export function generateGreatChest() { return { type: 'great_chest' }; }
`);
writeFileSync(join(REF, "settings.js"), `export const appSettings = { date: { month: 6, day: 12 } };`);

const { spawnHeart } = await import("./parity_ref/heart_generation.js");

function kindOf(res) {
  if (!res) return "none";
  if (res.type === "chest") return "chest";
  if (res.type === "great_chest") return "great_chest";
  if (res.type === "item") return res.item; // heart / mimic / chest_leggy
  return res.type || "unknown";
}

const SEEDS = [1, 42, 123456789, 999999937, 786433, 2000000000];
const out = [];
for (const seed of SEEDS) {
  // Sweep coordinates to exercise all branches (y<1536 and y>=1536, negatives).
  for (let xi = 0; xi < 20; xi++) {
    for (let yi = 0; yi < 20; yi++) {
      const x = -200 + xi * 137;
      const y = -100 + yi * 211; // crosses the 1536 threshold
      out.push({ seed, ng: 0, x, y, kind: kindOf(spawnHeart(seed, 0, x, y, "coalmine", {}, "normal")) });
    }
  }
}

writeFileSync(join(HERE, "heart_vectors.json"), JSON.stringify(out));
const counts = {};
for (const c of out) counts[c.kind] = (counts[c.kind] || 0) + 1;
console.log(`wrote heart_vectors.json: ${out.length} cases`, JSON.stringify(counts));
