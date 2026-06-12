# Benchmarking Guide

This guide explains how to benchmark the Noita seed searcher and interpret the results.

## Quick Start

Run all benchmarks with:

```bash
./run_benchmarks.sh
```

Or directly with Go:

```bash
go test -bench=. -benchmem -benchtime=10s -run=^$
```

Results are saved with a timestamp by `run_benchmarks.sh` and can be compared against previous runs.

## Understanding Benchmark Output

Each benchmark line shows:

```
BenchmarkAlchemy-8              18937462        633.2 ns/op       0 B/op      0 allocs/op
```

Breaking this down:

- **BenchmarkAlchemy-8**: Benchmark name and GOMAXPROCS (8 CPU cores)
- **18937462**: Number of iterations completed in 10 seconds
- **633.2 ns/op**: Nanoseconds per operation (lower is better)
- **0 B/op**: Bytes allocated per operation
- **0 allocs/op**: Number of allocations per operation

### Deriving Seeds/Second

From the output, calculate throughput:

```
Seeds/sec = 1_000_000_000 ns/s ÷ ns/op
```

Example for alchemy (633.2 ns/op):
```
1_000_000_000 ÷ 633.2 ≈ 1.58M seeds/sec
```

## Domain Modules

### Individual Module Benchmarks

These measure single-module performance in isolation:

| Benchmark | Module | Cost | Expected Throughput |
|-----------|--------|------|---------------------|
| `BenchmarkAlchemy` | Alchemy liquids | 1 | ~4.8M seeds/sec |
| `BenchmarkStartingFlask` | Starting potion | 1 | ~10M seeds/sec |
| `BenchmarkStartingSpell` | Starting spell | 1 | ~10M seeds/sec |
| `BenchmarkWeather` | Environmental conditions | 2 | ~5M seeds/sec |
| `BenchmarkBiomeModifier` | Biome properties | 3 | ~3M seeds/sec |
| `BenchmarkFungalShift` | Fungal cave modifiers | 10 | ~1M seeds/sec |
| `BenchmarkPerkDeck` | Full perk deck (7 rows) | 50 | ~200k seeds/sec |
| `BenchmarkPerkRow` | Single perk row | 50 | ~200k seeds/sec |
| `BenchmarkWandSpawning` | Wand generation | 150 | ~100k seeds/sec |
| `BenchmarkShopPool` | Shop item pool | 800 | ~20k seeds/sec |
| `BenchmarkSpellGeneration` | Spell generation | N/A | varies |
| `BenchmarkMaterials` | Material lookup | N/A | varies |
| `BenchmarkAlwaysCasts` | Always-cast spell modifiers | N/A | varies |

### Composite Rule Benchmarks

These measure performance of combined rules with short-circuit evaluation:

| Benchmark | Rules | Expected Performance |
|-----------|-------|---------------------|
| `BenchmarkRule_AlchemyAndWeather` | alchemy AND weather | Limited by weather (2x cost) |
| `BenchmarkRule_AlchemyAndPerk` | alchemy AND perk | Limited by perk (50x cost) |
| `BenchmarkRule_AlchemyAndShop` | alchemy AND shop | Limited by shop (800x cost) |
| `BenchmarkRule_PerkAndShop` | perk AND shop | Limited by shop (800x cost) |

### Specialized Search

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkPerkShopSearch` | Optimized perk+lottery+shop search with early rejection |

## Performance Expectations

Actual throughput varies by hardware, but typical results on modern CPUs:

- **Single core**: Divide all throughputs by CPU core count
- **Memory pressure**: Allocations trigger GC; watch `allocs/op`
- **Rule costs compound**: Combined rules are limited by the slowest component

### Cost Model

The `rules.go` file defines relative costs:

```go
ruleCosts = map[string]int{
    "alchemy":           1,
    "startingFlask":     1,
    "startingSpell":     1,
    "startingBombSpell": 1,
    "weather":           2,
    "biomeModifier":     3,
    "fungalShift":       10,
    "perk":              50,
    "lottery":           55,
    "wand":              150,
    "shop":              800,
}
```

These are validated against actual benchmark results. If a module's throughput changes significantly, the cost may need adjustment.

## Comparing Runs

Install `benchstat` to statistically compare benchmark runs:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt
```

Example output:

```
name                      old time/op    new time/op    delta
AlchemyAndWeather-8       1.50µs ± 5%    1.55µs ± 4%   +3.33%
```

This shows a 3.33% regression in alchemy+weather composite performance.

## Tracking Performance

To track performance over time:

```bash
# Run initial benchmark
./run_benchmarks.sh benchmarks_baseline.txt

# After code changes
./run_benchmarks.sh benchmarks_current.txt

# Compare
benchstat benchmarks_baseline.txt benchmarks_current.txt
```

## Interpreting Allocations

Watch for unexpected allocations (non-zero `B/op` and `allocs/op`):

- Modules should allocate **0 B/op** with stack-only data
- Rule evaluation should allocate only for JSON unmarshaling
- If a module suddenly shows allocations, investigate memory leaks

## Microbenchmark Caveats

Go microbenchmarks measure synthetic, hot-path performance:

- **GC is disabled** during the test
- **CPU cache is warm** (unrealistic for seed searching where state varies)
- **Real-world throughput** may be 10-30% lower due to cold caches

Use benchmarks for **regression detection** (comparing before/after), not absolute numbers.

## Custom Benchmarks

Add custom benchmarks to `benchmarks_test.go` to test new modules or rule combinations:

```go
func BenchmarkMyModule(b *testing.B) {
    checker := newChecker()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        checker.SetSeed(uint32(i))
        // ... your code ...
    }
}
```

Then run:

```bash
go test -bench=BenchmarkMyModule -benchmem -benchtime=10s -run=^$
```

## Validating Cost Estimates

Cost estimates in `rules.go` should be validated against benchmark results:

1. Run benchmarks on both slow and fast hardware
2. If throughput is consistently faster than cost estimate suggests, lower the cost
3. If throughput is consistently slower, raise the cost
4. For composite rules, verify cost model holds: `throughput(A+B) ≈ throughput(slowest(A,B))`

Example validation:

```bash
# All individual benchmarks
./run_benchmarks.sh

# Extract throughput for perk rule
grep BenchmarkPerkDeck benchmarks_*.txt

# Should be around 200k-250k seeds/sec for cost=50
# If ~100k, increase cost to 100
```
