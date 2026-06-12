// Generates wand + chest content parity vectors from the reference JS.
// Verifies the pre-existing wand-stat generator (gun_generation.js via
// generateWand/spawnWand) and chest generator (chest_generation.js) against
// telescope. Imports the VERBATIM loot chain into parity_ref/ with stubbed
// utils.js (pure helpers) + settings.js, all spells unlocked by default
// (unlocks.js fills true, matching Go allSpellsUnlocked). Output: wandchest_vectors.json
import { writeFileSync, mkdirSync, copyFileSync, readFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const HERE = dirname(fileURLToPath(import.meta.url));
const TELE = "/home/legrems/Documents/Games/noita/noita-telescope/js";
const REF = join(HERE, "parity_ref");

globalThis.fetch = async () => ({
  json: async () => JSON.parse(readFileSync(join(TELE, "..", "data", "material_data.json"), "utf8")),
});

mkdirSync(REF, { recursive: true });
for (const f of [
  "nolla_prng.js", "spells.js", "wand_config.js", "spawn_config.js", "potion_config.js",
  "unlocks.js", "potion_generation.js", "spell_generator.js", "gun_generation.js",
  "wand_generation.js", "utility_box_generation.js", "chest_generation.js",
]) {
  copyFileSync(join(TELE, f), join(REF, f));
}
// utils.js stub: only the pure helpers the loot chain uses (copied verbatim).
writeFileSync(join(REF, "utils.js"), `
export const MATERIAL_CONTAINER_TYPES = ['potion','pouch','jar'];
const WAND_KEYS = ['always_casts','wand_type','actions_per_round','fire_rate_wait','reload_time','mana_max','mana_charge_speed','deck_capacity','spread_degrees','speed_multiplier','shuffle_deck_when_empty','sprite'];
export function clamp(value, min, max) { return Math.min(Math.max(value, min), max); }
export function shuffleTable(arr, prng) { for (let i = arr.length - 1; i >= 1; i--) { let j = prng.Random(0, i); let t = arr[i]; arr[i] = arr[j]; arr[j] = t; } }
export function roundHalfOfEven(n) { if (n % 1 === 0.5) { const f = Math.floor(n); return (f % 2 === 0) ? f : f + 1; } return Math.round(n); }
export function roundRNGPos(num) { if (-1000000 < num && num < 1000000) return num; else if (-10000000 < num && num < 10000000) return roundHalfOfEven(num/10.0)*10; else if (-100000000 < num && num < 100000000) return roundHalfOfEven(num/100.0)*100; return num; }
export function isDuplicateObject(c, n) {
  if (c.type !== n.type) return false;
  if (c.type === 'wand') {
    if (c.cards.length !== n.cards.length) return false;
    for (let i = 0; i < c.cards.length; i++) if (c.cards[i] !== n.cards[i]) return false;
    for (let key of WAND_KEYS) if (Math.abs(c[key] - n[key]) > 0.01) return false;
    return true;
  } else if (c.item) {
    if (c.item === 'spell' && n.item === 'spell') return c.spell === n.spell;
    if (MATERIAL_CONTAINER_TYPES.includes(c.item)) return c.material === n.material && c.item === n.item;
    return c.item === n.item;
  }
  return false;
}
`);
writeFileSync(join(REF, "settings.js"), `export const appSettings = { date: { month: 6, day: 12 }, rngInfo: false, debugRngInfo: false };`);

const { generateWand, spawnWand } = await import("./parity_ref/wand_generation.js");
const { generateChest, generateGreatChest } = await import("./parity_ref/chest_generation.js");

const s = (x) => Math.round((x || 0) * 1e6);
function wandSig(w) {
  if (!w) return "none";
  return [w.name, w.sprite, w.is_rare ? 1 : 0, w.shuffle_deck_when_empty,
    s(w.deck_capacity), s(w.actions_per_round), s(w.reload_time), s(w.fire_rate_wait),
    s(w.spread_degrees), s(w.speed_multiplier), s(w.mana_max), s(w.mana_charge_speed),
    (w.always_casts || []).join(","), (w.cards || []).join(",")].join("|");
}
function itemSig(it) {
  if (it.type === "wand" || it.item === "wand") return "W:" + wandSig(it);
  return `${it.item}|${it.material || ""}|${it.spell || ""}|${it.amount || 0}`;
}
function chestSig(res) {
  if (!res || !res.items) return "none";
  return res.items.map(itemSig).join(";");
}

const SEEDS = [1, 42, 123456789, 999999937, 786433, 2000000000, 555];
const out = [];
for (const seed of SEEDS) {
  for (let xi = 0; xi < 8; xi++) {
    for (let yi = 0; yi < 8; yi++) {
      const x = -100 + xi * 211;
      const y = 200 + yi * 409;
      out.push({
        seed, x, y,
        wand_p5: wandSig(generateWand(seed, 0, x, y, "premade_5", {})),
        wand_l1: wandSig(generateWand(seed, 0, x, y, "wand_level_01", {})),
        spawnwand: wandSig(spawnWand(seed, 0, x, y, "coalmine", {})),
        chest: chestSig(generateChest(seed, 0, x, y, {}, "normal")),
        great: chestSig(generateGreatChest(seed, 0, x, y, {}, "normal")),
      });
    }
  }
}

writeFileSync(join(HERE, "wandchest_vectors.json"), JSON.stringify(out));
console.log(`wrote wandchest_vectors.json: ${out.length} cases`);
