// Generates tile-generation parity vectors from the reference noita-telescope JS.
// Uses the verbatim stbhw.js (copied into parity_ref/), the PIL-decoded base
// biome map (biome_base.json) and coalmine Wang tile (wang_base.json), so the
// comparison isolates: region detection, the reseed dance, and stbhw_generate_image
// (the PRE-HACK raw buffer — biome_hacks/pathfinding/masking are a later stage).
// findBiomeRegions + calculateMapDimensions are copied verbatim from
// tile_generator.js (both are pure; the rest of that module pulls heavy imports).
// Output: tile_vectors.json (consumed by tiles_parity_test.go).
import { readFileSync, writeFileSync, mkdirSync, copyFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const HERE = dirname(fileURLToPath(import.meta.url));
const TELE = "/home/legrems/Documents/Games/noita/noita-telescope/js";
const REF = join(HERE, "parity_ref");

mkdirSync(REF, { recursive: true });
copyFileSync(join(TELE, "stbhw.js"), join(REF, "stbhw.js"));
copyFileSync(join(TELE, "nolla_prng.js"), join(REF, "nolla_prng.js"));

const { NollaPrng } = await import("./parity_ref/nolla_prng.js");
const stbhw = await import("./parity_ref/stbhw.js");

// --- verbatim from tile_generator.js ---
function calculateMapDimensions(bbox) {
  const [minX, minY, maxX, maxY] = bbox;
  let totalWidth = 0;
  for (let x = minX; x <= maxX; x++) {
    totalWidth += 51;
    if (x % 5 === 4) totalWidth += 1;
  }
  let totalHeight = 0;
  for (let y = minY; y <= maxY; y++) {
    totalHeight += 51;
    if (y % 5 === 4) totalHeight += 1;
  }
  return { width: totalWidth, height: totalHeight };
}

function findBiomeRegions(pixels, width, height, targetColor) {
  const visited = new Uint8Array(width * height);
  const regions = [];
  const bboxes = [];
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = y * width + x;
      if (visited[idx] === 0 && pixels[idx] === targetColor) {
        const regionPoints = [];
        const queue = [[x, y]];
        visited[idx] = 1;
        let minX = width, maxX = 0, minY = height, maxY = 0;
        while (queue.length > 0) {
          const [cx, cy] = queue.shift();
          regionPoints.push([cx, cy]);
          if (cx < minX) minX = cx; if (cx > maxX) maxX = cx;
          if (cy < minY) minY = cy; if (cy > maxY) maxY = cy;
          const neighbors = [[1, 0], [-1, 0], [0, 1], [0, -1]];
          for (const [dx, dy] of neighbors) {
            const nx = cx + dx, ny = cy + dy;
            if (nx >= 0 && nx < width && ny >= 0 && ny < height) {
              const nIdx = ny * width + nx;
              if (visited[nIdx] === 0 && pixels[nIdx] === targetColor) {
                visited[nIdx] = 1;
                queue.push([nx, ny]);
              }
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
        if (valid) {
          regions.push(regionPoints);
          bboxes.push([minX, minY, maxX, maxY]);
        }
      }
    }
  }
  return { regions, bboxes };
}
// --- end verbatim ---

// Pre-hack portion of generateRawTileBuffer (tile_generator.js).
function genRawBuffer(bbox, ts, worldSeed, ngPlus, extraRerolls) {
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
  return { buffer: rawBuffer, width: mapW, height: outH, tileIndices: ti.tileIndices, xmax: ti.xmax, ymax: ti.ymax };
}

function hashBytes(arr) {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < arr.length; i++) {
    h = (h ^ (arr[i] & 0xff)) >>> 0;
    h = Math.imul(h, 16777619) >>> 0;
  }
  return h >>> 0;
}
function hashI32(arr) {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < arr.length; i++) {
    h = (h ^ (arr[i] & 0xffffffff)) >>> 0;
    h = Math.imul(h, 16777619) >>> 0;
  }
  return h >>> 0;
}

const base = JSON.parse(readFileSync(join(HERE, "biome_base.json")));
const wang = JSON.parse(readFileSync(join(HERE, "wang_base.json")));

// Build coalmine tileset from the PIL-decoded RGB.
const cm = wang.coalmine;
const ts = new stbhw.StbhwTileset();
stbhw.stbhw_build_tileset_from_image(ts, Uint8Array.from(cm.rgb), cm.w * 3, cm.w, cm.h);

// Find coalmine regions on the static NG0 map.
const COALMINE_COLOR = 0xffd57917 >>> 0;
const nb = base.normal;
const pixels = Uint32Array.from(nb.argb);
const { regions, bboxes } = findBiomeRegions(pixels, nb.w, nb.h, COALMINE_COLOR);

const out = {
  tileset: {
    isCorner: ts.is_corner,
    numColor: ts.num_color,
    shortSideLen: ts.short_side_len,
    numVaryX: ts.num_vary_x,
    numVaryY: ts.num_vary_y,
    numHTiles: ts.num_h_tiles,
    numVTiles: ts.num_v_tiles,
  },
  regions: bboxes.map((b, i) => ({ bbox: b, numPoints: regions[i].length })),
  cases: [],
};

const SEEDS = [123456789, 42, 999999937, 7, 2000000000, 1];
for (const seed of SEEDS) {
  for (let ri = 0; ri < bboxes.length; ri++) {
    const r = genRawBuffer(bboxes[ri], ts, seed, 0, 0);
    out.cases.push({
      seed, ng: 0, regionIdx: ri, bbox: bboxes[ri],
      width: r.width, height: r.height, xmax: r.xmax, ymax: r.ymax,
      bufferHash: hashBytes(r.buffer),
      tileIndicesHash: hashI32(r.tileIndices),
    });
  }
}

writeFileSync(join(HERE, "tile_vectors.json"), JSON.stringify(out));
console.log(`wrote tile_vectors.json: ${out.regions.length} coalmine region(s), ${out.cases.length} cases`);
console.log(`  tileset: corner=${ts.is_corner} ssl=${ts.short_side_len} h=${ts.num_h_tiles} v=${ts.num_v_tiles} vary=${ts.num_vary_x}x${ts.num_vary_y}`);
console.log(`  regions: ${JSON.stringify(bboxes)}`);
