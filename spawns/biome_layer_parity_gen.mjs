// Generates final-layer parity vectors from the reference noita-telescope JS.
// Imports VERBATIM biome_hacks.js + pathfinding.js + stbhw.js (copied into
// parity_ref/) with thin stubs for their heavy deps:
//   - utils.js  -> getWorldCenter (verbatim) + shuffleTable
//   - png_sanitizer.js -> loadPNG returns the coalmine overlay from a global
// The orchestration (generateRawTileBuffer full + per-region loop) and the pure
// helpers findBiomeRegions/calculateMapDimensions/applyMasking/blockOutRooms are
// copied verbatim from tile_generator.js / pixel_scene_generation.js below.
// Output: layer_vectors.json (consumed by hacks_parity_test.go).
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
// Stub png_sanitizer.js: feed the coalmine overlay from a global.
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

// Let preloadOverlays() (fired at biome_hacks load) finish.
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
// --- verbatim from pixel_scene_generation.js ---
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
// --- end verbatim ---

const MAX_PATHFINDING_ATTEMPTS = 99;
const BIOME_PATH_HEIGHT_LIMIT_CHUNKS = 4;
const RESTORE_BLOCKED_ROOMS = true;

// Full generateRawTileBuffer (tile_generator.js), pre-pathfinding portion.
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

// Per-region body of generateBiomeTiles (tile_generator.js).
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
  // (extension hack only for >1024; coalmine is 256-wide so skipped)

  const validChunks = new Set(region.map((p) => `${p[0]},${p[1]}`));
  const imgData = { data: new Uint8ClampedArray(rawResult.width * rawResult.mapH * 4) };
  applyMasking(rawResult.buffer, imgData, rawResult.width, bbox, validChunks, 4);

  return { buffer: rawResult.buffer, width: rawResult.width, height: rawResult.height, mapH: rawResult.mapH, path: finalPath, attempts };
}

function hashBytes(arr) {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < arr.length; i++) { h = (h ^ (arr[i] & 0xff)) >>> 0; h = Math.imul(h, 16777619) >>> 0; }
  return h >>> 0;
}

const base = JSON.parse(readFileSync(join(HERE, "biome_base.json")));
const wang = JSON.parse(readFileSync(join(HERE, "wang_base.json")));
const cm = wang.coalmine;
const ts = new stbhw.StbhwTileset();
stbhw.stbhw_build_tileset_from_image(ts, Uint8Array.from(cm.rgb), cm.w * 3, cm.w, cm.h);

const COALMINE_COLOR = 0xffd57917 >>> 0;
const nb = base.normal;
const pixels = Uint32Array.from(nb.argb);
const { regions, bboxes } = findBiomeRegions(pixels, nb.w, nb.h, COALMINE_COLOR);

const SEEDS = [123456789, 42, 999999937, 7, 2000000000, 1, 786433, 4294967295];
const out = { regions: bboxes, cases: [] };
for (const seed of SEEDS) {
  for (let ri = 0; ri < bboxes.length; ri++) {
    const layer = genTileLayer(bboxes[ri], regions[ri], ts, seed, 0, "coalmine", "normal", undefined);
    out.cases.push({
      seed, ng: 0, regionIdx: ri, bbox: bboxes[ri],
      width: layer.width, height: layer.height, mapH: layer.mapH,
      attempts: layer.attempts, pathLen: layer.path.length,
      bufferHash: hashBytes(layer.buffer),
    });
  }
}

writeFileSync(join(HERE, "layer_vectors.json"), JSON.stringify(out));
console.log(`wrote layer_vectors.json: ${out.cases.length} cases`);
for (const c of out.cases) console.log(`  seed=${c.seed} attempts=${c.attempts} pathLen=${c.pathLen} bufHash=${c.bufferHash}`);
