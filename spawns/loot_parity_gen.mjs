// Generates loot-content parity vectors from the reference noita-telescope JS.
// Verifies the pre-existing leaf loot generators (createPotion, createPowderPouch,
// spawnItem, spawnPotionAltar) — the dominant coalmine loot — against telescope.
// Imports VERBATIM potion_generation.js + its data deps (spawn_config, potion_config)
// into parity_ref/, stubbing unlocks.js (all unlocked, matching Go allSpellsUnlocked)
// and settings.js. Output: loot_vectors.json (consumed by loot_parity_test.go).
import { writeFileSync, mkdirSync, copyFileSync, readFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const HERE = dirname(fileURLToPath(import.meta.url));
const TELE = "/home/legrems/Documents/Games/noita/noita-telescope/js";
const REF = join(HERE, "parity_ref");

// potion_config.js fetches material_data.json at load (color lookups only, not
// the potion material arrays); node's fetch doesn't support file URLs, so polyfill.
globalThis.fetch = async () => ({
  json: async () => JSON.parse(readFileSync(join(TELE, "..", "data", "material_data.json"), "utf8")),
});

mkdirSync(REF, { recursive: true });
for (const f of ["potion_generation.js", "nolla_prng.js", "spawn_config.js", "potion_config.js"]) {
  copyFileSync(join(TELE, f), join(REF, f));
}
// All spells unlocked (matches Go allSpellsUnlocked = true).
writeFileSync(join(REF, "unlocks.js"), `export const unlockedSpells = new Proxy({}, { get: () => true });`);
writeFileSync(join(REF, "settings.js"), `export const appSettings = { date: { month: 6, day: 12 }, rngInfo: false, debugRngInfo: false };`);

const { createPotion, createPowderPouch, spawnItem, spawnPotionAltar } = await import("./parity_ref/potion_generation.js");

function sig(res) {
  if (!res) return "none";
  return `${res.item}|${res.material || ""}|${res.active ? "A" : ""}`;
}

const SEEDS = [1, 42, 123456789, 999999937, 786433, 2000000000, 555, 7];
const out = [];
for (const seed of SEEDS) {
  for (let xi = 0; xi < 12; xi++) {
    for (let yi = 0; yi < 12; yi++) {
      const x = -150 + xi * 173;
      const y = -50 + yi * 311; // crosses 1536 (mimic_potion branch)
      out.push({
        seed, x, y,
        potion: sig(createPotion(seed, 0, x, y, "normal", "normal")),
        pouch: sig(createPowderPouch(seed, 0, x, y)),
        item: sig(spawnItem(seed, 0, x, y, "coalmine", {}, "normal")),
        altar: sig(spawnPotionAltar(seed, 0, x, y, "coalmine", {}, "normal")),
      });
    }
  }
}

writeFileSync(join(HERE, "loot_vectors.json"), JSON.stringify(out));
const distinct = new Set();
for (const c of out) { distinct.add(c.potion); distinct.add(c.item); distinct.add(c.altar); }
console.log(`wrote loot_vectors.json: ${out.length} cases, ${distinct.size} distinct content signatures`);
