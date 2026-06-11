# Noita Seed Searcher

A high-performance parallel seed searcher for the game [Noita](https://noitagame.com/). It efficiently searches through the entire seed space to find seeds matching custom criteria like specific perks, items, spells, shops, and more.

## Overview

Noita uses a seeded RNG for all procedural generation. This tool replicates the game's RNG algorithm and rule evaluation system to search for seeds with desired properties before actually playing them.

The searcher supports:
- **Flexible rule definitions** (JSON-based) for complex AND/OR/NOT logic
- **Specialized perk+shop search** with optimized fast-path for common scenarios
- **Parallel processing** across all CPU cores
- **Real-time progress reporting** with ETA estimation

## Architecture

### Core Components

#### RNG (`rng.go`)
Implements Noita's Linear Congruential Generator (LCG) PRNG, ported from `nolla_prng.zig`. The RNG state is deterministic based on the world seed, allowing reproducible generation of all procedural content.

**Key constants:**
- `lcgModulus = 0x7fffffff`
- `lcgMultiplier = 0x41a7`
- `lcgDivisor = 0x1f31d`

#### Rule Evaluation (`rules.go`)
Implements a tree-based rule evaluation system supporting logical operations:
- **Leaf rules**: Atomic checks (alchemy, perks, wands, shops, etc.)
- **Composite rules**: AND, OR, NOT logic with short-circuit evaluation

Rules are **cost-sorted before evaluation** — cheaper rules are checked first so expensive rules (shops: 800 cost, wands: 150) are skipped when cheaper rules (alchemy: 1) fail. This optimization dramatically improves throughput.

#### Game Domain Modules

Each module simulates a specific aspect of Noita's procedural generation:

- **Alchemy** (`alchemy.go`): Liquids and potions
- **Perks** (`perk.go`): Perk deck generation and selection
- **Wands** (`wand.go`): Wand spawning and properties
- **Shops** (`shop.go`): Shop pool generation
- **Spells** (`spells.go`): Spell generation
- **Starting items** (`starting_items.go`): Initial equipment
- **Biome modifiers** (`biome_modifier.go`): Biome properties
- **Weather** (`weather.go`): Environmental conditions
- **Fungal shifts** (`fungal.go`): Fungal cave effects
- **Materials** (`materials.go`): Material properties
- **Always casts** (`alwayscasts.go`): Always-cast spell modifiers

#### Search Implementations

- **Generic Rule Search** (`main.go`): Main entry point supporting arbitrary rule trees
- **Perk+Shop Search** (`perkshop_search.go`): Optimized path for the common case:
  - Row 0 with 3 specific perks not replaced by lottery
  - First shop with specific potion+teleport perk combinations
  - Uses precomputed coordinates to reject 87.5% of seeds without full deck build

#### Verification (`verify_test.go`)
Test suite ensuring the searcher's RNG and rule evaluation match actual game behavior.

### Execution Model

1. **Divide seeds**: Split the search range into chunks, one per worker goroutine
2. **Per-seed evaluation**:
   - Set RNG to seed state
   - Evaluate rule tree against RNG
   - Output seed if rule passes
3. **Progress tracking**: Parallel stats goroutine reports throughput and ETA every 5 seconds
4. **Result collection**: Matching seeds printed to stdout, one per line

## Building

```bash
go build -o noita-seed-search
```

Requires Go 1.21+.

## Usage

### Generic Rule Search

Search using a custom rule JSON:

```bash
./noita-seed-search -rules <json|file> [-from N] [-to N] [-workers N] [-stats]
```

**Options:**
- `-rules`: Rule definition as JSON string or path to JSON file (required)
- `-from`: Starting seed (default: 1)
- `-to`: Ending seed, exclusive (default: 1073741823, ~2^30)
- `-workers`: Number of parallel goroutines (default: CPU count)
- `-stats`: Print progress every 5 seconds (default: true)

**Example: Find seeds with water alchemy.**

```bash
./noita-seed-search -rules '{
  "id":"root",
  "type":"and",
  "rules":[
    {
      "id":"r1",
      "type":"alchemy",
      "val":{"LC":["water"],"AP":[]}
    }
  ]
}'
```

**Example: Save rules to a file and search:**

```bash
cat > rules.json <<'EOF'
{
  "id": "root",
  "type": "and",
  "rules": [
    {
      "id": "r1",
      "type": "perk",
      "val": {
        "names": ["PERKS_LOTTERY"],
        "match": "all"
      }
    },
    {
      "id": "r2",
      "type": "shop",
      "val": {
        "poolId": 1,
        "items": ["WAND_FIREBALL"],
        "match": "any"
      }
    }
  ]
}
EOF

./noita-seed-search -rules rules.json -from 1 -to 10000000
```

### Perk + Shop Search (Optimized)

```bash
./noita-seed-search -perkshop [-from N] [-to N] [-workers N] [-stats]
```

This specialized search looks for:
- Row 0 (first perk choice) containing:
  - `PERKS_LOTTERY` (double perk picks)
  - `EDIT_WANDS_EVERYWHERE` (tinker wands anywhere)
  - One of: `NO_MORE_SHUFFLE` or `UNLIMITED_SPELLS`
  - None replaced by lottery (survives 50% chance)
- First shop containing one of:
  - `CHAINSAW` + (`TELEPORT_PROJECTILE` or `TELEPORT_PROJECTILE_SHORT`)
  - OR `MANA_REDUCE` + same teleport perks

**Example:**

```bash
./noita-seed-search -perkshop -from 1 -to 100000000 -workers 8
```

## Rule Format (JSON)

Rules are defined as a tree of nodes. Each node has:

- `id`: Unique identifier (string, any value)
- `type`: Node type — one of:
  - **Logic**: `"and"`, `"or"`, `"not"`, `"rules"`
  - **Leaf**: `"alchemy"`, `"perk"`, `"wand"`, `"shop"`, `"spell"`, `"startingSpell"`, `"startingBombSpell"`, `"startingFlask"`, `"weather"`, `"biomeModifier"`, `"fungalShift"`, etc.
- `rules`: Array of child nodes (only for logic types)
- `val`: Payload for leaf rules (type-specific, passed as JSON)
- `params`: Optional parameters (unused in current version)

### Example Complex Rule

Find seeds with both water *and* lava:

```json
{
  "id": "root",
  "type": "and",
  "rules": [
    {
      "id": "water",
      "type": "alchemy",
      "val": {"LC": ["water"], "AP": []}
    },
    {
      "id": "lava",
      "type": "alchemy",
      "val": {"LC": ["lava"], "AP": []}
    }
  ]
}
```

Find seeds with lottery *or* tinker in row 0:

```json
{
  "id": "root",
  "type": "or",
  "rules": [
    {
      "id": "lottery",
      "type": "perk",
      "val": {"names": ["PERKS_LOTTERY"], "match": "all"}
    },
    {
      "id": "tinker",
      "type": "perk",
      "val": {"names": ["EDIT_WANDS_EVERYWHERE"], "match": "all"}
    }
  ]
}
```

## Performance

The searcher achieves:
- **~4.8M seeds/sec** with alchemy-only rules
- **~90k seeds/sec** with perk rules
- **~5k seeds/sec** with shop rules

Performance scales linearly with CPU cores. A typical search across 100M seeds takes 1-5 minutes on modern hardware.

## Output

Matching seeds are printed to stdout, one per line:

```
1234567
2345678
3456789
```

Redirect to a file:

```bash
./noita-seed-search -rules ... > matching_seeds.txt
```

Progress is printed to stderr and does not interfere with results.

## Testing

Run the test suite:

```bash
go test -v
```

Tests verify RNG output matches the game's implementation and that rule evaluation is correct.

## License

This project is a reverse-engineering effort for the purposes of seed searching in Noita.
