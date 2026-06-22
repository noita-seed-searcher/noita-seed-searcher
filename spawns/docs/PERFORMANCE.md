# Great-Chest Search — Performance Work

Optimization log for `-mode search-great-chest` (the parallel seed scanner that
generates biome tiles and reports seeds containing a `great_chest`).

All changes are **behaviour-preserving**: output was verified byte-for-byte
identical to the pre-optimization baseline (`f67c37c`) over 20,000 seeds × 13
tile-generating biomes, with zero generation errors, and the
tiles/scan/heart/biome/hacks/rng parity tests stay green.

## Headline result

Workload: 12,000 seeds, `-biomes coalmine,excavationsite`, 32-core machine.

| Stage | Time (12k seeds) | Cumulative speedup |
|-------|------------------|--------------------|
| Baseline (`f67c37c`) | ~4.5 s | 1.00× |
| Round 1 — `perf: global increase` | ~2.8 s | 1.61× |
| Round 2 — `perf: pool pathfinding scratch + bulk-copy tile draws` | ~2.4 s | 1.88× |
| Round 3 — `perf: O(1) spawn-color lookup` | ~2.3 s | **1.95×** |

## How the search works (context)

Per seed, for each target biome region:
1. **Generate** the herringbone Wang tile (`stbhwGenerateImage`) into an RGB buffer.
2. **Hacks** mutate the buffer for pathfinding: `blockOutRooms`, `applyMainBiomeHack`, `applyCoalmineHack` (overlay).
3. **Pathfind** a top→bottom walkable route with `findMinPath` (BFS); failure triggers a reroll (regenerate with a bumped PRNG).
4. **Post-process**: restore blocked rooms, `undoCoalmineHack`, coffee/random-color fills, large-region extension, masking.
5. **Scan** the final buffer (`prescanSpawnFunctions`) for spawn-function colours → world coordinates.
6. **Dispatch** only `great_chest`-capable spawn funcs and keep the hits.

The search is parallel across `GOMAXPROCS` workers in ordered batches; output stays
deterministic regardless of scheduling. For `ng==0` the base biome map is
seed-independent, so its regions/bounding boxes are precomputed **once** and shared.

## Round 1 — `perf: global increase`

- **Static region precompute (`search.go`).** For `ng==0`, the base biome map
  (and therefore each biome's regions + bounding boxes) is identical for every
  seed. Compute it once up front instead of decoding a PNG + scanning the map per
  seed. Removes a full decode + several large allocations + a map scan from every
  seed's work.
- **Corner-color grid pool (`stbhw.go`).** Each tile generation allocated a
  90,000-int32 (~360 KB) `cColor` grid *and* filled it with `-1` — then
  `stbhwGenerateImage` immediately re-filled it with `-1`, making the first fill
  pure waste. Now pulled from a `sync.Pool` (via `newStbhwGen`/`release`) with the
  redundant fill removed. Eliminated `newStbhwGen` (was ~11.7% cum) and most of
  the per-call zeroing.
- **`isBlockedColor` switch (`hacks.go`).** `blockOutRooms` did a
  `map[uint32]bool` lookup for nearly every pixel. Replaced the 11-entry map with
  a `switch` (compare chain). Removed `mapaccess1_fast32` (was ~7.5% flat).
- **Scanner string-lookup inline (`scan.go`).** `prescanSpawnFunctions` already
  fetches the biome's `fns` once, but called `getSpawnFunctionIndex` per pixel,
  which redid the `biomeSpawnFunctionMap[biome]` *string-map* lookup every time.
  Inlined the scan against the already-fetched slice.
- Added a gated `-cpuprofile` flag (`main.go`) for future profiling.

## Round 2 — `perf: pool pathfinding scratch + bulk-copy tile draws`

- **Generation-stamped BFS scratch (`hacks.go`, `findMinPath`).** This was the
  largest remaining allocator: every reroll of every region allocated fresh
  `visited` + `parents` arrays (`width*height` each), filled `parents` with `-1`
  (O(n)), and grew the BFS queue via repeated `memmove`. Replaced with a
  `sync.Pool`-recycled scratch where "visited" is a **generation stamp**
  (`seen[i] == gen`): bumping `gen` gives a fresh visited set with **no O(n)
  reset**, `parents` needs no init (only read for stamped nodes), and the queue's
  backing array is reused (no regrowth). Arrays are only zeroed on the rare `gen`
  wraparound.
- **Slice pre-sizing.** `rooms` in `blockOutRooms` and `detected` in
  `prescanSpawnFunctions` are pre-allocated to kill `growslice` churn.
- **Bulk-copy tile draws (`stbhw.go`, `drawHTile`/`drawVTile`).** Unclipped tiles
  (the common interior case) have contiguous rows, so each row is now a single
  `copy()` (fast `memmove`) instead of per-pixel bounds-checked byte writes;
  clipped tiles fall back to the original path. Cut the two draw functions from
  ~20% combined to ~4%.

Effect on the profile: `growslice` dropped out of the top entries, `memclr`
roughly halved (~6.2% → ~2.6%), `memmove` ~2.6% → ~0.6%.

## Round 3 — `perf: O(1) spawn-color lookup`

- **Membership bitset (`scan.go`, `prescanSpawnFunctions`).** A biome's `fns`
  table holds ~50 entries (coalmine ≈ 49, excavationsite ≈ 54), and the scanner
  linear-scanned all of them for every non-black/white pixel — almost all of
  which are terrain, so they scanned the whole list and matched nothing.
  Replaced with a precomputed **2²⁴-bit membership set** over `0xRRGGBB` colours
  (O(1) reject for the common case) plus a first-match `map[uint32]int` for the
  rare hits, built once per biome and cached. Preserves
  `getSpawnFunctionIndex`'s first-match semantics exactly.

  Cut `prescanSpawnFunctions` CPU from ~17.8% flat to ~5.2% flat. The modest
  wall-clock gain at 32 cores is the bandwidth ceiling (below).

## Scaling characteristics

Worker-scaling test (6,000 seeds, coalmine+excavationsite, 32 logical CPUs):

| `GOMAXPROCS` | Time | Speedup vs 1 |
|--------------|------|--------------|
| 1  | 12.0 s | 1.0× |
| 2  | 6.1 s  | 2.0× |
| 4  | 3.2 s  | 3.75× |
| 8  | 1.8 s  | 6.7× |
| 12 | 1.45 s | 8.3× |

Near-linear to ~4 cores, good to ~8, then **diminishing returns** — the per-pixel
full-buffer passes (`blockOutRooms`, stbhw generation, the coalmine overlay hack,
`findMinPath`) saturate memory bandwidth at high core counts. Implication: at low
core counts the work is compute-bound (a faster language would help per core); at
high core counts it is bandwidth-bound (reducing memory *traffic* matters more
than reducing instructions).

## Current hot path (post round 3)

Roughly, by cumulative cost:

- `blockOutRooms` — full-buffer per-pixel scan + room flood
- `stbhwGenerateImage` — the Wang tile generation (`chooseTile`, draws)
- `findMinPath` — BFS pathfinding (+ rerolls)
- `applyOverlayHack` — the coalmine overlay, run **twice** (apply before
  pathfinding, undo after)

These are core algorithm work, not incidental overhead.

## Remaining levers (not yet done)

Lowest-risk first, all in Go:

1. **Cut the double overlay pass.** `applyCoalmineHack` then `undoCoalmineHack`
   each scan the whole overlay; combine or localize to the border region.
2. **Reduce full-buffer passes.** `blockOutRooms`, the postprocess fills, and
   `prescanSpawnFunctions` each traverse the buffer; fusing where stages permit
   cuts memory traffic (the bandwidth bottleneck).
3. **Shrink buffers / tighten layout** to lower bytes-touched per seed.

A native (Rust/C) rewrite of the generation core would help per-core throughput
(~1.5–2×) but is bandwidth-capped past ~8 cores on a 32-core box (realistically
~1.3–1.7× overall), at the cost of re-porting ~5k parity-verified lines and
re-validating against the JS reference. See [`CUDA_PLAN.md`](./CUDA_PLAN.md) for
the GPU option.

## Reproducing the measurements

```sh
cd spawns
go build -o noita-spawn-gen .

# timing
time ./noita-spawn-gen -mode search-great-chest \
    -seed-start 1 -seed-end 12000 -biomes coalmine,excavationsite >/dev/null

# CPU profile
./noita-spawn-gen -mode search-great-chest \
    -seed-start 1 -seed-end 12000 -biomes coalmine,excavationsite \
    -cpuprofile cpu.prof >/dev/null
go tool pprof -top -nodecount=20 cpu.prof

# parity / correctness
go test ./...   # tiles/scan/heart/biome/hacks/rng parity green
```

> Note: the pre-existing `TestLootContentParity` / `TestWandChestContentParity`
> (potion divergence) failures are unrelated to the search path and fail
> identically on the untouched tree.
