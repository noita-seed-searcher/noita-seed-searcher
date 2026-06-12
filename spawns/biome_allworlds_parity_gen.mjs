// Generates parity vectors for all biomes across multiple seeds.
// Uses the reference noita-telescope JS to enumerate spawn points in every biome.
// Output: allworlds_vectors.json (consumed by allworlds_parity_test.go).
import { readFileSync, writeFileSync, mkdirSync, copyFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const HERE = dirname(fileURLToPath(import.meta.url));
const TELE = "/home/legrems/Documents/Games/noita/noita-telescope/js";
const REF = join(HERE, "parity_ref");

mkdirSync(REF, { recursive: true });
copyFileSync(join(TELE, "stbhw.js"), join(REF, "stbhw.js"));
copyFileSync(join(TELE, "nolla_prng.js"), join(REF, "nolla_prng.js"));
copyFileSync(join(TELE, "biome_hacks.js"), join(REF, "biome_hacks.js"));
copyFileSync(join(TELE, "pathfinding.js"), join(REF, "pathfinding.js"));

// Stub utils.js: getWorldCenter (verbatim) + shuffleTable.
writeFileSync(join(REF, "utils.js"), `
export function getWorldCenter(isNGP, gameMode='normal') {
    if (gameMode === 'nightmare') return 32;
    return isNGP ? 32 : 35;
}
export function shuffleTable(arr, prng) {
    for (let i = arr.length - 1; i >= 1; i--) {
        let j = prng.Random(0, i);
        let t = arr[i]; arr[i] = arr[j]; arr[j] = t;
    }
}
`);

// Stub png_sanitizer.js: return null for all biomes except coalmine.
// For coalmine, feed overlay_base.json's coalmine entry.
writeFileSync(join(REF, "png_sanitizer.js"), `
export async function loadPNG(url) { return globalThis.__COALMINE_OVERLAY__ || null; }
`);

const overlayJson = JSON.parse(readFileSync(join(HERE, "overlay_base.json")));
const ov = overlayJson.coalmine;
globalThis.__COALMINE_OVERLAY__ = { data: Uint8ClampedArray.from(ov.rgba), width: ov.w, height: ov.h };

const stbhw = await import("./parity_ref/stbhw.js");
const { NollaPrng } = await import("./parity_ref/nolla_prng.js");
const hacks = await import("./parity_ref/biome_hacks.js");
const { findMinPath } = await import("./parity_ref/pathfinding.js");
const { getWorldCenter } = await import("./parity_ref/utils.js");

await new Promise((r) => setTimeout(r, 50));

// --- verbatim from tile_generator.js ---
function calculateMapDimensions(bbox) {
  const [minX, minY, maxX, maxY] = bbox;
  let totalWidth = 0;
  for (let x = minX; x <= maxX; x++) { totalWidth += 51; if (x % 5 === 4) totalWidth += 1; }
  let totalHeight = 0;
  for (let y = minY; y <= maxY; y++) { totalHeight += 51; if (y % 5 === 4) totalHeight += 1; }
  return { width: totalWidth, height: totalHeight };
}

function findBiomeRegions(pixels, width, height, targetColor) {
  const visited = new Uint8Array(width * height);
  const regions = [], bboxes = [];
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = y * width + x;
      if (visited[idx] === 0 && pixels[idx] === targetColor) {
        const regionPoints = [], queue = [[x, y]];
        visited[idx] = 1;
        let minX = width, maxX = 0, minY = height, maxY = 0;
        while (queue.length > 0) {
          const [cx, cy] = queue.shift();
          regionPoints.push([cx, cy]);
          if (cx < minX) minX = cx; if (cx > maxX) maxX = cx;
          if (cy < minY) minY = cy; if (cy > maxY) maxY = cy;
          for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) {
            const nx = cx + dx, ny = cy + dy;
            if (nx >= 0 && nx < width && ny >= 0 && ny < height) {
              const nIdx = ny * width + nx;
              if (visited[nIdx] === 0 && pixels[nIdx] === targetColor) { visited[nIdx] = 1; queue.push([nx, ny]); }
            }
          }
        }
        let valid = true;
        for (let i = 0; i < regions.length; i++) {
          const [rMinX, rMinY, rMaxX, rMaxY] = bboxes[i];
          if (minX >= rMinX && maxX <= rMaxX && minY >= rMinY && maxY <= rMaxY) {
            for (const p of regionPoints) regions[i].push(p);
            valid = false;
          }
        }
        if (valid) { regions.push(regionPoints); bboxes.push([minX, minY, maxX, maxY]); }
      }
    }
  }
  return { regions, bboxes };
}

function applyMasking(pixels, imgData, mapW, bbox, validChunks, offsetY = 4) {
  const [minCX, minCY, maxCX, maxCY] = bbox;
  let tx = 0;
  for (let cx = minCX; cx <= maxCX; cx++) {
    let cw = 51; if (cx % 5 === 4) cw += 1;
    let ty = 0;
    for (let cy = minCY; cy <= maxCY; cy++) {
      let ch = 51; if (cy % 5 === 4) ch += 1;
      if (validChunks.has(`${cx},${cy}`)) {
        for (let y = 0; y < ch; y++) for (let x = 0; x < cw; x++) {
          const srcIdx = ((ty + y + offsetY) * mapW + (tx + x)) * 3;
          const dstIdx = ((ty + y) * mapW + (tx + x)) * 4;
          const r = pixels[srcIdx], g = pixels[srcIdx + 1], b = pixels[srcIdx + 2];
          imgData.data[dstIdx] = r; imgData.data[dstIdx + 1] = g; imgData.data[dstIdx + 2] = b;
          imgData.data[dstIdx + 3] = (r <= 1 && g <= 1 && b <= 1) ? 0 : 255;
        }
      } else {
        for (let y = 0; y < ch; y++) for (let x = 0; x < cw; x++) {
          const srcIdx = ((ty + y + offsetY) * mapW + (tx + x)) * 3;
          pixels[srcIdx] = 0; pixels[srcIdx + 1] = 0; pixels[srcIdx + 2] = 0;
        }
      }
      ty += ch;
    }
    tx += cw;
  }
}

const BLOCKED_COLORS = [0x00ac6e, 0x70d79e, 0x70d79f, 0x70d7a0, 0x70d7a1, 0x7868ff, 0xc35700, 0xff0080, 0xff00ff, 0xff0aff, 0x00AC64];
function blockOutRooms(pixels, width, height) {
  let rooms = [];
  for (let y = 4; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = (y * width + x) * 3;
      const color = (pixels[idx] << 16) | (pixels[idx + 1] << 8) | pixels[idx + 2];
      if (color === 0x000000 || color === 0xffffff) continue;
      if (!BLOCKED_COLORS.includes(color)) continue;
      let startX = x + 1, startY = y + 1, endX = x + 1, endY = y + 1, foundEnd = false;
      while (!foundEnd && endX < width) {
        if (endX >= width) break;
        const tempIdx = (startY * width + endX) * 3;
        const tempColor = (pixels[tempIdx] << 16) | (pixels[tempIdx + 1] << 8) | pixels[tempIdx + 2];
        if (tempColor === 0x000000 || tempColor === 0x323232) { endX++; continue; }
        endX--; foundEnd = true;
      }
      if (endX >= width) endX = width - 1;
      foundEnd = false;
      while (!foundEnd && endY < height) {
        if (endY >= height) break;
        const tempIdx = (endY * width + startX) * 3;
        const tempColor = (pixels[tempIdx] << 16) | (pixels[tempIdx + 1] << 8) | pixels[tempIdx + 2];
        if (tempColor === 0x000000 || tempColor === 0x323232) { endY++; continue; }
        endY--; foundEnd = true;
      }
      if (endY >= height) endY = height - 1;
      if (endX > startX && endY > startY) {
        for (let by = startY; by <= endY; by++) for (let bx = startX; bx <= endX; bx++) {
          const bIdx = (by * width + bx) * 3;
          pixels[bIdx] = 0xff; pixels[bIdx + 1] = 0x01; pixels[bIdx + 2] = 0xff;
        }
      }
      rooms.push({ color, startX, startY, endX, endY });
    }
  }
  return rooms;
}

const MAX_PATHFINDING_ATTEMPTS = 99;
const BIOME_PATH_HEIGHT_LIMIT_CHUNKS = 4;
const RESTORE_BLOCKED_ROOMS = true;

function genRawTile(bbox, ts, worldSeed, ngPlus, extraRerolls, biomeName, gameMode) {
  const minX = bbox[0], minY = bbox[1];
  const dims = calculateMapDimensions(bbox);
  const mapW = dims.width, mapH = dims.height;
  const outH = mapH + 4;
  if (ts.h_tiles.length === 0) return null;

  const prng = new NollaPrng(0);
  stbhw.stbhw_set_prng(prng);
  prng.SetRandomFromWorldSeed(worldSeed + ngPlus);
  prng.Next();
  const iters = mapW + (worldSeed + ngPlus) - 11 * Math.floor(mapW / 11) - 12 * Math.floor((worldSeed + ngPlus) / 12);
  for (let i = 0; i < iters; i++) prng.Next();
  for (let i = 0; i < extraRerolls; i++) prng.Next();
  prng.Seed = prng.NextU();
  prng.Next();

  const rawBuffer = new Uint8Array(mapW * outH * 3);
  const ti = stbhw.stbhw_generate_image(ts, rawBuffer, mapW * 3, mapW, outH);
  if (!ti) return null;

  let pixelSceneRooms = [];
  if (biomeName === "coalmine" || biomeName === "excavationsite") pixelSceneRooms = blockOutRooms(rawBuffer, mapW, outH);
  if (bbox[0] <= getWorldCenter(ngPlus > 0, gameMode) && bbox[2] >= getWorldCenter(ngPlus > 0, gameMode)) {
    hacks.applyMainBiomeHack(bbox[0], rawBuffer, mapW, outH, biomeName, ngPlus > 0, gameMode);
  }
  if ((biomeName === "coalmine" || biomeName === "solid_wall_tower_1") && gameMode !== "nightmare") {
    hacks.applyCoalmineHack(rawBuffer, mapW, outH, "coalmine");
  }
  return { buffer: rawBuffer, width: mapW, height: outH, minX, minY, mapH, pixelSceneRooms };
}

function genTileLayer(bbox, region, ts, worldSeed, ngPlus, biomeName, gameMode, randomColors) {
  let valid = false, currentRerolls = 0, attempts = 0, rawResult = null, finalPath = null;
  while (!valid && attempts < MAX_PATHFINDING_ATTEMPTS) {
    rawResult = genRawTile(bbox, ts, worldSeed, ngPlus, currentRerolls, biomeName, gameMode);
    if (!rawResult) break;
    let path = (1 + bbox[3] - bbox[1] > BIOME_PATH_HEIGHT_LIMIT_CHUNKS) ? [] : findMinPath(bbox, rawResult.buffer, rawResult.width, rawResult.height, biomeName, ngPlus > 0, gameMode);
    if (path) { valid = true; finalPath = path; } else { currentRerolls++; attempts++; }
  }
  if (attempts === MAX_PATHFINDING_ATTEMPTS) {
    rawResult = genRawTile(bbox, ts, worldSeed, ngPlus, currentRerolls, biomeName, gameMode);
    valid = true; finalPath = [];
  }
  if (!(valid && rawResult)) return null;

  if (rawResult.pixelSceneRooms && RESTORE_BLOCKED_ROOMS) {
    for (const r of rawResult.pixelSceneRooms) {
      for (let y = r.startY; y <= r.endY; y++) for (let x = r.startX; x <= r.endX; x++) {
        const idx = (y * rawResult.width + x) * 3;
        rawResult.buffer[idx] = 0; rawResult.buffer[idx + 1] = 0; rawResult.buffer[idx + 2] = 0;
      }
    }
  }
  if ((biomeName === "coalmine" || biomeName === "solid_wall_tower_1") && gameMode !== "nightmare") {
    hacks.undoCoalmineHack(rawResult.buffer, rawResult.width, rawResult.height, "coalmine");
  }
  hacks.applyPostprocessingHacks(rawResult.buffer, rawResult.width, rawResult.height, worldSeed, ngPlus, finalPath, randomColors);

  const validChunks = new Set(region.map((p) => `${p[0]},${p[1]}`));
  const imgData = { data: new Uint8ClampedArray(rawResult.width * rawResult.mapH * 4) };
  applyMasking(rawResult.buffer, imgData, rawResult.width, bbox, validChunks, 4);

  return { buffer: rawResult.buffer, width: rawResult.width, height: rawResult.height, mapH: rawResult.mapH, path: finalPath, attempts, minX: rawResult.minX, minY: rawResult.minY, biomeName };
}

// --- verbatim spawn-function tables from spawn_function_config.js + biomeSpawnFunctionMap ---
const DEFAULT_SPAWNS = [
  { color: 0xff0000, funcName: "spawn_small_enemies" }, { color: 0x800000, funcName: "spawn_big_enemies" },
  { color: 0x00ff00, funcName: "spawn_items" }, { color: 0xc88d1a, funcName: "spawn_props" },
  { color: 0xc88000, funcName: "spawn_props2" }, { color: 0xc80040, funcName: "spawn_props3" },
  { color: 0xffff00, funcName: "spawn_lamp" }, { color: 0xff0aff, funcName: "load_pixel_scene" },
  { color: 0xFF0080, funcName: "load_pixel_scene2" }, { color: 0xFF8000, funcName: "spawn_unique_enemy" },
  { color: 0xc84040, funcName: "spawn_unique_enemy2" }, { color: 0x804040, funcName: "spawn_unique_enemy3" },
  { color: 0x96C850, funcName: "spawn_ghostlamp" }, { color: 0x60A064, funcName: "spawn_candles" },
  { color: 0x50a000, funcName: "spawn_potion_altar" }, { color: 0xbca0f0, funcName: "spawn_potions" },
  { color: 0x00FF5A, funcName: "spawn_apparition" }, { color: 0x78FFFF, funcName: "spawn_heart" },
  { color: 0x50A0F0, funcName: "spawn_wands" }, { color: 0xbf26a6, funcName: "spawn_portal" },
  { color: 0x04A977, funcName: "spawn_end_portal" }, { color: 0xffd171, funcName: "spawn_orb" },
  { color: 0xffd181, funcName: "spawn_perk" }, { color: 0xffff81, funcName: "spawn_all_perks" },
  { color: 0xc7eb28, funcName: "spawn_wand_trap" }, { color: 0xE8FF80, funcName: "spawn_wand_trap_ignite" },
  { color: 0x2768DE, funcName: "spawn_wand_trap_electricity_source" }, { color: 0x2768DF, funcName: "spawn_wand_trap_electricity" },
  { color: 0x6b4f9b, funcName: "spawn_moon" }, { color: 0xd7b3e8, funcName: "spawn_collapse" },
];

const COALMINE_SPAWNS = [
  { color: 0x0000ff, funcName: "spawn_nest" }, { color: 0xB40000, funcName: "spawn_fungi" },
  { color: 0x969678, funcName: "load_structures" }, { color: 0x967878, funcName: "load_large_structures" },
  { color: 0x967896, funcName: "load_i_structures" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0xC35700, funcName: "load_oiltank" }, { color: 0x55AF4B, funcName: "load_altar" },
  { color: 0x23B9C3, funcName: "spawn_altar_torch" }, { color: 0x55AF8C, funcName: "spawn_skulls" },
  { color: 0x55FF8C, funcName: "spawn_chest" }, { color: 0x4e175e, funcName: "load_oiltank_alt" },
  { color: 0x33934c, funcName: "spawn_shopitem" }, { color: 0x50fafa, funcName: "spawn_trapwand" },
  { color: 0xf12ab5, funcName: "spawn_bbqbox" }, { color: 0x005cfd, funcName: "spawn_swing_puzzle_box" },
  { color: 0x00b5fc, funcName: "spawn_swing_puzzle_target" }, { color: 0x93ca00, funcName: "spawn_oiltank_puzzle" },
  { color: 0xb97300, funcName: "spawn_receptacle_oil" },
];

const COALMINE_ALT_SPAWNS = [
  { color: 0x0000ff, funcName: "spawn_nest" }, { color: 0xB40000, funcName: "spawn_fungi" },
  { color: 0x969678, funcName: "load_structures" }, { color: 0x967878, funcName: "load_large_structures" },
  { color: 0x80FF5A, funcName: "spawn_vines" }, { color: 0x33934c, funcName: "spawn_shopitem" },
];

const EXCAVATIONSITE_SPAWNS = [
  { color: 0x0000ff, funcName: "spawn_nest" }, { color: 0xFF50FF, funcName: "spawn_hanger" },
  { color: 0x00AC64, funcName: "load_pixel_scene4" }, { color: 0x00ac6e, funcName: "load_pixel_scene4_alt" },
  { color: 0x0050FF, funcName: "spawn_wheel" }, { color: 0x0150FF, funcName: "spawn_wheel_small" },
  { color: 0x0250FF, funcName: "spawn_wheel_tiny" }, { color: 0x2d2eac, funcName: "spawn_rock" },
  { color: 0x0A50FF, funcName: "spawn_physicsstructure" }, { color: 0xc999ff, funcName: "spawn_hanging_prop" },
  { color: 0x7868ff, funcName: "load_puzzleroom" }, { color: 0x70d79e, funcName: "load_gunpowderpool_01" },
  { color: 0x70d79f, funcName: "load_gunpowderpool_02" }, { color: 0x70d7a0, funcName: "load_gunpowderpool_03" },
  { color: 0x70d7a1, funcName: "load_gunpowderpool_04" }, { color: 0x33934c, funcName: "spawn_shopitem" },
  { color: 0xb09016, funcName: "spawn_meditation_cube" }, { color: 0x00855c, funcName: "spawn_receptacle" },
  { color: 0xb1ff99, funcName: "spawn_tower_short" }, { color: 0x5c8550, funcName: "spawn_tower_tall" },
  { color: 0x227fff, funcName: "spawn_beam_low" }, { color: 0x8228ff, funcName: "spawn_beam_low_flipped" },
  { color: 0x0098ba, funcName: "spawn_beam_steep" }, { color: 0x7600a9, funcName: "spawn_beam_steep_flipped" },
];

const SNOWCAVE_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x00AC33, funcName: "load_pixel_scene3" },
  { color: 0x00AC64, funcName: "load_pixel_scene4" }, { color: 0x4691c7, funcName: "load_puzzle_capsule" },
  { color: 0x3691d7, funcName: "load_puzzle_capsule_b" }, { color: 0x55AF4B, funcName: "load_altar" },
  { color: 0x23B9C3, funcName: "spawn_altar_torch" }, { color: 0x55AF8C, funcName: "spawn_skulls" },
  { color: 0xF516E3, funcName: "spawn_scavenger_party" }, { color: 0xFFC84E, funcName: "spawn_acid" },
  { color: 0x7285c4, funcName: "load_acidtank_right" }, { color: 0x9472c4, funcName: "load_acidtank_left" },
  { color: 0x504600, funcName: "spawn_stones" }, { color: 0xc800ff, funcName: "load_pixel_scene_alt" },
  { color: 0x33934c, funcName: "spawn_shopitem" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0x434040, funcName: "spawn_burning_barrel" }, { color: 0xb4a00a, funcName: "spawn_fish" },
  { color: 0xaa42ff, funcName: "spawn_electricity_trap" }, { color: 0x366178, funcName: "spawn_buried_eye_teleporter" },
  { color: 0x876543, funcName: "spawn_statue_hand" }, { color: 0x00855c, funcName: "spawn_receptacle" },
];

const SNOWCASTLE_SPAWNS = [
  { color: 0xC8C800, funcName: "spawn_lamp2" }, { color: 0x01a1fa, funcName: "spawn_turret" },
  { color: 0x80FF5A, funcName: "spawn_vines" }, { color: 0xc78f20, funcName: "spawn_barricade" },
  { color: 0xc022f5, funcName: "spawn_forcefield_generator" }, { color: 0xa3d900, funcName: "spawn_brimstone" },
  { color: 0x00d982, funcName: "spawn_vasta_or_vihta" }, { color: 0x932020, funcName: "spawn_cook" },
  { color: 0x614630, funcName: "load_panel_01" }, { color: 0x614635, funcName: "load_panel_02" },
  { color: 0x61463e, funcName: "load_panel_03" }, { color: 0x614638, funcName: "load_panel_04" },
  { color: 0x614646, funcName: "load_panel_07" }, { color: 0x614650, funcName: "load_panel_08" },
  { color: 0x614658, funcName: "load_panel_09" }, { color: 0xc133ff, funcName: "load_chamfer_top_r" },
  { color: 0x8b33ff, funcName: "load_chamfer_top_l" }, { color: 0x8824b3, funcName: "load_chamfer_bottom_r" },
  { color: 0x5f23ad, funcName: "load_chamfer_bottom_l" }, { color: 0x73ffa7, funcName: "load_chamfer_inner_top_r" },
  { color: 0xd5ff7f, funcName: "load_chamfer_inner_top_l" }, { color: 0x387d51, funcName: "load_chamfer_inner_bottom_r" },
  { color: 0x97b55b, funcName: "load_chamfer_inner_bottom_l" }, { color: 0x44609c, funcName: "load_pillar_filler" },
  { color: 0x44449c, funcName: "load_pillar_filler_tall" }, { color: 0xb03058, funcName: "load_pod_large" },
  { color: 0xb05830, funcName: "load_pod_small_l" }, { color: 0xb09030, funcName: "load_pod_small_r" },
  { color: 0xffa659, funcName: "load_furniture" }, { color: 0xfec390, funcName: "load_furniture_bunk" },
  { color: 0x4c63e0, funcName: "spawn_root_grower" }, { color: 0x4cacab, funcName: "spawn_forge_check" },
  { color: 0x2a78ff, funcName: "spawn_drill_laser" },
];

const RAINFOREST_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x400000, funcName: "spawn_scavengers" },
  { color: 0x400080, funcName: "spawn_large_enemies" }, { color: 0xC8C800, funcName: "spawn_lamp2" },
  { color: 0x00AC64, funcName: "load_pixel_scene4" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0x943030, funcName: "spawn_dragonspot" }, { color: 0x4c63e0, funcName: "spawn_root_grower" },
  { color: 0x806326, funcName: "spawn_tree" },
];

const RAINFOREST_OPEN_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x400000, funcName: "spawn_scavengers" },
  { color: 0x400080, funcName: "spawn_large_enemies" }, { color: 0xC8C800, funcName: "spawn_lamp2" },
  { color: 0x00AC64, funcName: "load_pixel_scene4" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0x943030, funcName: "spawn_dragonspot" }, { color: 0x4c63e0, funcName: "spawn_root_grower" },
  { color: 0x806326, funcName: "spawn_tree" },
];

const RAINFOREST_DARK_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x400000, funcName: "spawn_scavengers" },
  { color: 0x400080, funcName: "spawn_large_enemies" }, { color: 0xC8C800, funcName: "spawn_lamp2" },
  { color: 0x00AC64, funcName: "load_pixel_scene4" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0x943030, funcName: "spawn_dragonspot" }, { color: 0x4c63e0, funcName: "spawn_root_grower" },
  { color: 0x806326, funcName: "spawn_tree" },
];

const VAULT_SPAWNS = [
  { color: 0x692e94, funcName: "load_pixel_scene_wide" }, { color: 0x822e5b, funcName: "load_pixel_scene_tall" },
  { color: 0x00AC64, funcName: "load_warning_strip" }, { color: 0x01a1fa, funcName: "spawn_turret" },
  { color: 0x80FF5A, funcName: "spawn_vines" }, { color: 0x504B64, funcName: "spawn_machines" },
  { color: 0xc999ff, funcName: "spawn_hanging_prop" }, { color: 0xBE8246, funcName: "spawn_pipes_hor" },
  { color: 0xBE8264, funcName: "spawn_pipes_turn_right" }, { color: 0xBE8282, funcName: "spawn_pipes_turn_left" },
  { color: 0xBE82A0, funcName: "spawn_pipes_ver" }, { color: 0xBE82BE, funcName: "spawn_pipes_cross" },
  { color: 0x2E8246, funcName: "spawn_pipes_big_hor" }, { color: 0x2E8264, funcName: "spawn_pipes_big_turn_right" },
  { color: 0x2E8282, funcName: "spawn_pipes_big_turn_left" }, { color: 0x2E82A0, funcName: "spawn_pipes_big_ver" },
  { color: 0x5c73da, funcName: "spawn_stains" }, { color: 0x5c73db, funcName: "spawn_stains_ceiling" },
  { color: 0xc78f20, funcName: "spawn_barricade" }, { color: 0x4a107d, funcName: "load_pillar" },
  { color: 0x7b59ab, funcName: "load_pillar_base" }, { color: 0x40ffce, funcName: "load_catwalk" },
  { color: 0xbf4c86, funcName: "spawn_apparatus" }, { color: 0xaa42ff, funcName: "spawn_electricity_trap" },
  { color: 0x33934c, funcName: "spawn_shopitem" }, { color: 0xacf14b, funcName: "spawn_laser_trap" },
  { color: 0xa45aff, funcName: "spawn_lab_puzzle" },
];

const VAULT_FROZEN_SPAWNS = [
  { color: 0x400000, funcName: "spawn_robots" }, { color: 0x00AC64, funcName: "load_pixel_scene4" },
  { color: 0x01a1fa, funcName: "spawn_turret" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0x504B64, funcName: "spawn_machines" }, { color: 0xBE8246, funcName: "spawn_pipes_hor" },
  { color: 0xBE8264, funcName: "spawn_pipes_turn_right" }, { color: 0xBE8282, funcName: "spawn_pipes_turn_left" },
  { color: 0xBE82A0, funcName: "spawn_pipes_ver" }, { color: 0xBE82BE, funcName: "spawn_pipes_cross" },
  { color: 0xc78f20, funcName: "spawn_barricade" },
];

const CRYPT_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x808000, funcName: "spawn_statues" },
  { color: 0x00AC33, funcName: "load_pixel_scene3" }, { color: 0x00AC64, funcName: "load_pixel_scene4" },
  { color: 0x97ab00, funcName: "load_pixel_scene5" }, { color: 0xc9d959, funcName: "load_pixel_scene5b" },
  { color: 0xC8C800, funcName: "spawn_lamp2" }, { color: 0x400080, funcName: "spawn_large_enemies" },
  { color: 0xC8001A, funcName: "spawn_ghost_crystal" }, { color: 0x82FF5A, funcName: "spawn_crawlers" },
  { color: 0x647D7D, funcName: "spawn_pressureplates" }, { color: 0x649B7D, funcName: "spawn_doors" },
  { color: 0xA07864, funcName: "spawn_scavengers" }, { color: 0xFFCD2A, funcName: "spawn_scorpions" },
  { color: 0x2D1E5A, funcName: "spawn_bones" }, { color: 0x782060, funcName: "load_beam" },
  { color: 0x783060, funcName: "load_background_scene" }, { color: 0x378ec4, funcName: "load_small_background_scene" },
  { color: 0x786460, funcName: "load_cavein" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0x535988, funcName: "spawn_statue_back" }, { color: 0x33934c, funcName: "spawn_shopitem" },
];

const FUNGICAVE_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x400000, funcName: "spawn_robots" },
  { color: 0x0000ff, funcName: "spawn_nest" }, { color: 0x30b3b0, funcName: "spawn_physics_fungus" },
];

const FUNGIFOREST_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x0000ff, funcName: "spawn_nest" },
  { color: 0x30b3b0, funcName: "spawn_physics_fungus" }, { color: 0x30b3f0, funcName: "spawn_physics_acid_fungus" },
  { color: 0x80FF5A, funcName: "spawn_vines" }, { color: 0x6a8d79, funcName: "spawn_fungitrap" },
];

const RAINFOREST_DARK2_SPAWNS = RAINFOREST_DARK_SPAWNS;

const WIZARDCAVE_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x808000, funcName: "spawn_statues" },
  { color: 0x00AC33, funcName: "load_pixel_scene3" }, { color: 0x00AC64, funcName: "load_pixel_scene4" },
  { color: 0x97ab00, funcName: "load_pixel_scene5" }, { color: 0xc9d959, funcName: "load_pixel_scene5b" },
  { color: 0xC8C800, funcName: "spawn_lamp2" }, { color: 0x400080, funcName: "spawn_large_enemies" },
  { color: 0xC8001A, funcName: "spawn_ghost_crystal" }, { color: 0x82FF5A, funcName: "spawn_crawlers" },
  { color: 0x647D7D, funcName: "spawn_pressureplates" }, { color: 0x649B7D, funcName: "spawn_doors" },
  { color: 0xA07864, funcName: "spawn_scavengers" }, { color: 0xFFCD2A, funcName: "spawn_scorpions" },
  { color: 0x2D1E5A, funcName: "spawn_bones" }, { color: 0x782060, funcName: "load_beam" },
  { color: 0x783060, funcName: "load_background_scene" }, { color: 0x378ec4, funcName: "load_small_background_scene" },
  { color: 0x786460, funcName: "load_cavein" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0x535988, funcName: "spawn_statue_back" }, { color: 0x33934c, funcName: "spawn_shopitem" },
];

const LIQUIDCAVE_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x00AC64, funcName: "load_background_panel_big" },
  { color: 0x967878, funcName: "spawn_lasergun" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0xc88dab, funcName: "spawn_statues" },
];

const ROBOBASE_SPAWNS = [
  { color: 0x00AC64, funcName: "load_warning_strip" }, { color: 0x01a1fa, funcName: "spawn_turret" },
  { color: 0x80FF5A, funcName: "spawn_vines" }, { color: 0xc999ff, funcName: "spawn_hanging_prop" },
  { color: 0xc78f20, funcName: "spawn_barricade" }, { color: 0x33934c, funcName: "spawn_shopitem" },
  { color: 0x39a760, funcName: "spawn_lasergate_ver" },
];

const MEAT_SPAWNS = [
  { color: 0xffeedd, funcName: "init" }, { color: 0x55AF8C, funcName: "spawn_skulls" },
  { color: 0x4c63e1, funcName: "spawn_cyst" }, { color: 0x80FF5A, funcName: "spawn_vines" },
  { color: 0xd97f7f, funcName: "spawn_mouth" }, { color: 0xc999ff, funcName: "spawn_hanging_prop" },
];

const TOWER_SPAWNS = [
  { color: 0x0000ff, funcName: "spawn_nest" }, { color: 0xB40000, funcName: "spawn_fungi" },
  { color: 0x80FF5A, funcName: "spawn_vines" }, { color: 0x55AF8C, funcName: "spawn_skulls" },
  { color: 0x55FF8C, funcName: "spawn_chest" }, { color: 0x33934c, funcName: "spawn_shopitem" },
];

const WATCHTOWER_SPAWNS = [
  { color: 0xaaff00, funcName: "spawn_small_enemies2" }, { color: 0xffaa00, funcName: "spawn_big_enemies2" },
];

const TEMPLES_COMMON_SPAWNS = [
  { color: 0x805000, funcName: "spawn_cloud_trap" }, { color: 0x397780, funcName: "load_floor_rubble" },
  { color: 0x00ffa0, funcName: "load_floor_rubble_l" }, { color: 0x1ca7ff, funcName: "load_floor_rubble_r" },
  { color: 0xffeed1, funcName: "spawn_puzzle_watchtower" }, { color: 0xffeeda, funcName: "spawn_puzzle_barren" },
  { color: 0xffeedb, funcName: "spawn_puzzle_potion_mimics" }, { color: 0xffeedc, funcName: "spawn_puzzle_darkness" },
  { color: 0xffeedd, funcName: "spawn_boss" }, { color: 0xffeede, funcName: "spawn_potion_mimic_empty" },
  { color: 0xffeedf, funcName: "spawn_potion_mimic" }, { color: 0xffeed0, funcName: "spawn_fish_many" },
  { color: 0xffeed2, funcName: "spawn_boss_phase2_marker" }, { color: 0xffeed3, funcName: "spawn_book_barren" },
  { color: 0xffeed4, funcName: "spawn_potion_beer" }, { color: 0xffeed5, funcName: "spawn_potion_milk" },
  { color: 0xffeed6, funcName: "spawn_scorpion" }, { color: 0xffaaaa, funcName: "spawn_sign_left" },
  { color: 0xffaadd, funcName: "spawn_sign_right" },
];

const BIOME_SPAWN_FUNCTION_MAP = {
  "coalmine": [...DEFAULT_SPAWNS, ...COALMINE_SPAWNS],
  "coalmine_alt": [...DEFAULT_SPAWNS, ...COALMINE_ALT_SPAWNS],
  "excavationsite": [...DEFAULT_SPAWNS, ...EXCAVATIONSITE_SPAWNS],
  "snowcave": [...DEFAULT_SPAWNS, ...SNOWCAVE_SPAWNS],
  "snowcastle": [...DEFAULT_SPAWNS, ...SNOWCASTLE_SPAWNS],
  "rainforest": [...DEFAULT_SPAWNS, ...RAINFOREST_SPAWNS],
  "rainforest_open": [...DEFAULT_SPAWNS, ...RAINFOREST_OPEN_SPAWNS],
  "rainforest_dark": [...DEFAULT_SPAWNS, ...RAINFOREST_DARK_SPAWNS],
  "vault": [...DEFAULT_SPAWNS, ...VAULT_SPAWNS],
  "vault_frozen": [...DEFAULT_SPAWNS, ...VAULT_FROZEN_SPAWNS],
  "crypt": [...DEFAULT_SPAWNS, ...CRYPT_SPAWNS],
  "fungicave": [...DEFAULT_SPAWNS, ...FUNGICAVE_SPAWNS],
  "fungiforest": [...DEFAULT_SPAWNS, ...FUNGIFOREST_SPAWNS],
  "wizardcave": [...DEFAULT_SPAWNS, ...WIZARDCAVE_SPAWNS],
  "liquidcave": [...DEFAULT_SPAWNS, ...LIQUIDCAVE_SPAWNS],
  "robobase": [...DEFAULT_SPAWNS, ...ROBOBASE_SPAWNS],
  "meat": [...DEFAULT_SPAWNS, ...MEAT_SPAWNS],
  "solid_wall_tower_1": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_2": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_3": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_4": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_5": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_6": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_7": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_8": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "solid_wall_tower_9": [...DEFAULT_SPAWNS, ...TOWER_SPAWNS],
  "biome_watchtower": [...DEFAULT_SPAWNS, ...TEMPLES_COMMON_SPAWNS, ...WATCHTOWER_SPAWNS],
  "biome_potion_mimics": [...DEFAULT_SPAWNS, ...TEMPLES_COMMON_SPAWNS],
  "biome_darkness": [...DEFAULT_SPAWNS, ...TEMPLES_COMMON_SPAWNS],
  "biome_boss_sky": [...DEFAULT_SPAWNS, ...TEMPLES_COMMON_SPAWNS],
  "biome_barren": [...DEFAULT_SPAWNS, ...TEMPLES_COMMON_SPAWNS],
};

function getSpawnFunctionIndex(biomeName, color) {
  const fns = BIOME_SPAWN_FUNCTION_MAP[biomeName];
  if (!fns) return null;
  for (let i = 0; i < fns.length; i++) if (fns[i].color === color) return i;
  return null;
}

// --- verbatim tileToWorldCoordinates (utils.js); constants inlined ---
const WORLD_CHUNK_CENTER_X = 35, WORLD_CHUNK_CENTER_X_NGP = 32, WORLD_CHUNK_CENTER_Y = 14;
const CHUNK_SIZE = 512, TILE_SIZE = 10, TILE_OFFSET_X = 5, TILE_OFFSET_Y = -13;
function tileToWorldCoordinates(chunkBaseX, chunkBaseY, tileX, tileY, pw = 0, pwVertical = 0, isNGP = false, gameMode = "normal") {
  const world_chunk_center_x = (isNGP || gameMode === "nightmare") ? WORLD_CHUNK_CENTER_X_NGP : WORLD_CHUNK_CENTER_X;
  const worldSize = (isNGP || gameMode === "nightmare") ? 64 * 512 - 8 : 70 * 512;
  let smallChunkSize = Math.floor(CHUNK_SIZE / TILE_SIZE);
  let div5offX = 5 * CHUNK_SIZE * Math.floor((chunkBaseX - world_chunk_center_x) / 5);
  let mod5offX = (((chunkBaseX - world_chunk_center_x) % 5 + 5) % 5);
  let worldBaseX = div5offX + mod5offX * smallChunkSize * TILE_SIZE;
  let worldX_alt = -TILE_SIZE + worldBaseX + tileX * TILE_SIZE + TILE_OFFSET_X;
  let div5offY = 5 * CHUNK_SIZE * Math.floor((chunkBaseY - WORLD_CHUNK_CENTER_Y) / 5);
  let mod5offY = (((chunkBaseY - WORLD_CHUNK_CENTER_Y) % 5 + 5) % 5);
  let worldBaseY = div5offY + mod5offY * smallChunkSize * TILE_SIZE;
  if (mod5offY > 0) worldBaseY += TILE_SIZE;
  let worldY_alt = -TILE_SIZE + worldBaseY + tileY * TILE_SIZE + TILE_OFFSET_Y;
  if (isNGP || gameMode === "nightmare") { if (mod5offX >= 3) worldX_alt += TILE_SIZE; }
  worldY_alt += TILE_SIZE;
  if (isNGP || gameMode === "nightmare") worldX_alt -= 4;
  worldX_alt += pw * worldSize;
  worldY_alt += pwVertical * 24570;
  return { x: worldX_alt, y: worldY_alt };
}

// --- verbatim prescanSpawnFunctions (poi_scanner.js) for one layer ---
function prescanLayer(layer, isNGP, gameMode) {
  const detected = [];
  const sourceBiome = layer.biomeName;
  const width = layer.width, height = layer.mapH;
  const fns = BIOME_SPAWN_FUNCTION_MAP[sourceBiome] || [];
  if (fns.length === 0) return detected;
  for (let y = 4; y < height + 4; y++) {
    for (let x = 0; x < width; x++) {
      const srcIdx = (y * width + x) * 3;
      const r = layer.buffer[srcIdx], g = layer.buffer[srcIdx + 1], b = layer.buffer[srcIdx + 2];
      const colorInt = (r << 16) | (g << 8) | b;
      if (colorInt === 0x000000 || colorInt === 0xffffff) continue;
      const index = getSpawnFunctionIndex(sourceBiome, colorInt);
      if (index !== null) {
        const coords = tileToWorldCoordinates(layer.minX, layer.minY, x, y - 4, 0, 0, isNGP, gameMode);
        detected.push({ funcName: fns[index].funcName, index, x: coords.x, y: coords.y });
      }
    }
  }
  return detected;
}

function hashScan(list) {
  let h = 2166136261 >>> 0;
  for (const s of list) {
    for (const v of [s.index, s.x | 0, s.y | 0]) {
      h = (h ^ (v & 0xffffffff)) >>> 0;
      h = Math.imul(h, 16777619) >>> 0;
    }
  }
  return h >>> 0;
}

// PNG decoder using zlib.inflateSync
import { inflateSync } from "zlib";

function decodePNG(filepath) {
  const buf = readFileSync(filepath);
  let off = 8; // skip 8-byte signature
  let width, height, colorType;
  const idatParts = [];
  while (off < buf.length) {
    const len = buf.readUInt32BE(off); off += 4;
    const type = buf.subarray(off, off + 4).toString("ascii"); off += 4;
    const chunk = buf.subarray(off, off + len); off += len + 4; // +4 = CRC
    if (type === "IHDR") {
      width = chunk.readUInt32BE(0);
      height = chunk.readUInt32BE(4);
      colorType = chunk[9]; // 2=RGB, 6=RGBA
    } else if (type === "IDAT") {
      idatParts.push(chunk);
    } else if (type === "IEND") break;
  }
  const raw = inflateSync(Buffer.concat(idatParts));
  const ch = colorType === 6 ? 4 : 3;
  const rowLen = width * ch;
  const recon = Buffer.alloc(height * rowLen);
  for (let y = 0; y < height; y++) {
    const filt = raw[y * (1 + rowLen)];
    const src = raw.subarray(y * (1 + rowLen) + 1, (y + 1) * (1 + rowLen));
    const dst = recon.subarray(y * rowLen, (y + 1) * rowLen);
    const prv = y > 0 ? recon.subarray((y - 1) * rowLen, y * rowLen) : null;
    for (let i = 0; i < rowLen; i++) {
      const rb = src[i];
      const a = i >= ch ? dst[i - ch] : 0;
      const b = prv ? prv[i] : 0;
      const d = (i >= ch && prv) ? prv[i - ch] : 0;
      let val;
      switch (filt) {
        case 0: val = rb; break;
        case 1: val = rb + a; break;
        case 2: val = rb + b; break;
        case 3: val = rb + Math.floor((a + b) / 2); break;
        case 4: { const p=a+b-d,pa=Math.abs(p-a),pb=Math.abs(p-b),pc=Math.abs(p-d); val=rb+(pa<=pb&&pa<=pc?a:pb<=pc?b:d); break; }
        default: val = rb;
      }
      dst[i] = val & 0xff;
    }
  }
  // Extract RGB only (drop alpha)
  const rgb = new Uint8Array(width * height * 3);
  for (let i = 0; i < width * height; i++) {
    rgb[i*3]=recon[i*ch]; rgb[i*3+1]=recon[i*ch+1]; rgb[i*3+2]=recon[i*ch+2];
  }
  return { rgb, w: width, h: height };
}

// Build BIOME_CONFIG from biome_config.go: name, color, wangFile, randomColors
const BIOME_CONFIG = [
  { biomeName: "coalmine", color: 0xffd57917, wangFile: "data/wang_tiles/coalmine.png", randomColors: null },
  { biomeName: "coalmine_alt", color: 0xffD56517, wangFile: "data/wang_tiles/coalmine_alt.png", randomColors: null },
  { biomeName: "excavationsite", color: 0xff124445, wangFile: "data/wang_tiles/excavationsite.png", randomColors: null },
  { biomeName: "snowcave", color: 0xff1775d5, wangFile: "data/wang_tiles/snowcave.png", randomColors: null },
  { biomeName: "snowcastle", color: 0xff0046FF, wangFile: "data/wang_tiles/snowcastle.png", randomColors: null },
  { biomeName: "rainforest", color: 0xff808000, wangFile: "data/wang_tiles/rainforest.png", randomColors: null },
  { biomeName: "rainforest_open", color: 0xffA08400, wangFile: "data/wang_tiles/rainforest_open.png", randomColors: null },
  { biomeName: "vault", color: 0xff008000, wangFile: "data/wang_tiles/vault.png", randomColors: null },
  { biomeName: "crypt", color: 0xff786C42, wangFile: "data/wang_tiles/crypt.png", randomColors: null },
  { biomeName: "fungicave", color: 0xffe861f0, wangFile: "data/wang_tiles/fungicave.png", randomColors: null },
  { biomeName: "fungiforest", color: 0xffa861ff, wangFile: "data/wang_tiles/fungiforest.png", randomColors: null },
  { biomeName: "rainforest_dark", color: 0xff375c00, wangFile: "data/wang_tiles/rainforest_dark.png", randomColors: null },
  { biomeName: "wizardcave", color: 0xff726186, wangFile: "data/wang_tiles/wizardcave.png", randomColors: null },
  { biomeName: "liquidcave", color: 0xff89a04b, wangFile: "data/wang_tiles/liquidcave.png", randomColors: { 0x01CFEE: [0xF86868, 0x7FCEEA, 0xA3569F, 0xC23055, 0x0BFFE5] } },
  { biomeName: "robobase", color: 0xff4e5267, wangFile: "data/wang_tiles/robobase.png", randomColors: null },
  { biomeName: "vault_frozen", color: 0xff0080a8, wangFile: "data/wang_tiles/vault_frozen.png", randomColors: null },
  { biomeName: "meat", color: 0xff572828, wangFile: "data/wang_tiles/meat.png", randomColors: null },
  { biomeName: "solid_wall_tower_1", color: 0xff3d3e37, wangFile: "data/wang_tiles/coalmine.png", randomColors: null },
  { biomeName: "solid_wall_tower_2", color: 0xff3d3e38, wangFile: "data/wang_tiles/excavationsite.png", randomColors: null },
  { biomeName: "solid_wall_tower_3", color: 0xff3d3e39, wangFile: "data/wang_tiles/snowcave.png", randomColors: null },
  { biomeName: "solid_wall_tower_4", color: 0xff3d3e3a, wangFile: "data/wang_tiles/snowcastle.png", randomColors: null },
  { biomeName: "solid_wall_tower_5", color: 0xff3d3e3b, wangFile: "data/wang_tiles/fungicave.png", randomColors: null },
  { biomeName: "solid_wall_tower_6", color: 0xff3d3e3c, wangFile: "data/wang_tiles/rainforest.png", randomColors: null },
  { biomeName: "solid_wall_tower_7", color: 0xff3d3e3d, wangFile: "data/wang_tiles/vault.png", randomColors: null },
  { biomeName: "solid_wall_tower_8", color: 0xff3d3e3e, wangFile: "data/wang_tiles/crypt.png", randomColors: null },
  { biomeName: "solid_wall_tower_9", color: 0xff3d3e3f, wangFile: "data/wang_tiles/the_end.png", randomColors: null },
  { biomeName: "biome_watchtower", color: 0xffb70000, wangFile: "data/wang_tiles/static/watchtower_fg.png", randomColors: null },
  { biomeName: "biome_potion_mimics", color: 0xffff00fe, wangFile: "data/wang_tiles/static/potion_mimics_fg.png", randomColors: null },
  { biomeName: "biome_darkness", color: 0xffff00fd, wangFile: "data/wang_tiles/static/darkness_fg.png", randomColors: null },
  { biomeName: "biome_boss_sky", color: 0xffff00fc, wangFile: "data/wang_tiles/static/boss_fg.png", randomColors: null },
  { biomeName: "biome_barren", color: 0xffff00fb, wangFile: "data/wang_tiles/static/barren_fg.png", randomColors: null },
];

// Load biome_base.json
const base = JSON.parse(readFileSync(join(HERE, "biome_base.json")));
const nb = base.normal;
const pixels = Uint32Array.from(nb.argb);

// Pre-build tileset cache for each unique wangFile
const tilesetCache = {};
const wangFilesUsed = new Set(BIOME_CONFIG.filter(e => e.wangFile).map(e => e.wangFile));
for (const wangFile of wangFilesUsed) {
  try {
    const wangPath = join(HERE, wangFile);
    const decoded = decodePNG(wangPath);
    const ts = new stbhw.StbhwTileset();
    stbhw.stbhw_build_tileset_from_image(ts, decoded.rgb, decoded.w * 3, decoded.w, decoded.h);
    if (ts.h_tiles.length > 0) {
      tilesetCache[wangFile] = ts;
    } else {
      console.warn(`WARNING: tileset ${wangFile} built but h_tiles empty, skipping biomes using it`);
    }
  } catch (err) {
    console.warn(`WARNING: failed to load/build tileset ${wangFile}: ${err.message}`);
  }
}

const SEEDS = [42, 1, 999999937];
const out = { cases: [] };

for (const seed of SEEDS) {
  for (const entry of BIOME_CONFIG) {
    if (!entry.wangFile || !tilesetCache[entry.wangFile]) {
      continue; // Skip if no tileset
    }
    const ts = tilesetCache[entry.wangFile];

    const { regions, bboxes } = findBiomeRegions(pixels, nb.w, nb.h, entry.color);
    if (regions.length === 0) {
      continue; // No regions for this biome in the base map
    }

    for (let ri = 0; ri < bboxes.length; ri++) {
      const layer = genTileLayer(bboxes[ri], regions[ri], ts, seed, 0, entry.biomeName, "normal", entry.randomColors);
      if (!layer) continue;

      const detected = prescanLayer(layer, false, "normal");
      const count = detected.length;
      const hash = hashScan(detected);

      out.cases.push({
        seed,
        biome: entry.biomeName,
        count,
        hash,
        sample: detected.slice(0, 3),
      });
    }
  }
}

writeFileSync(join(HERE, "allworlds_vectors.json"), JSON.stringify(out));
console.log(`wrote allworlds_vectors.json: ${out.cases.length} cases`);
for (let i = 0; i < Math.min(out.cases.length, 10); i++) {
  const c = out.cases[i];
  console.log(`  seed=${c.seed} biome=${c.biome} spawns=${c.count} hash=${c.hash}`);
}
if (out.cases.length > 10) {
  console.log(`  ... and ${out.cases.length - 10} more cases`);
}
