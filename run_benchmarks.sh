#!/bin/bash
# Benchmark script to measure performance of each domain module
# Usage: ./run_benchmarks.sh [output_file]
#
# Results are saved to a file for comparison with previous runs.
# Requires Go to be installed.

set -e

OUTPUT_FILE="${1:-benchmarks_$(date +%Y%m%d_%H%M%S).txt}"

echo "Running benchmarks..."
echo "Output: $OUTPUT_FILE"
echo ""

go test -bench=. -benchmem -benchtime=10s -run=^$ 2>&1 | tee "$OUTPUT_FILE"

echo ""
echo "Benchmarks complete. Results saved to: $OUTPUT_FILE"
echo ""
echo "Cost estimates (from rules.go):"
echo "  alchemy: 1, startingFlask: 1, startingSpell: 1, startingBombSpell: 1"
echo "  weather: 2, biomeModifier: 3, fungalShift: 10"
echo "  perk: 50, lottery: 55, wand: 150, shop: 800"
echo ""
echo "To compare with a previous run:"
echo "  benchstat $OUTPUT_FILE <previous_file>"
