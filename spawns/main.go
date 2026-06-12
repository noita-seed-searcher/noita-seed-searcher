package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	seed := flag.Uint("seed", 0, "World seed")
	ng := flag.Int("ng", 0, "New Game Plus count")
	pwMax := flag.Int("pw-max", 0, "Parallel world range (±N horizontal)")
	pwMaxV := flag.Int("pw-max-vertical", 0, "Parallel world range (±N vertical)")
	x := flag.Float64("x", 0, "X coordinate")
	y := flag.Float64("y", 0, "Y coordinate")
	mode := flag.String("mode", "chest", "Mode: chest, great-chest, wand, item, potion, pouch, list-spawns, score-biomes")
	spellSearch := flag.String("spell", "", "Filter list-spawns to spawns containing this spell (case-insensitive, substring)")
	weightsFile := flag.String("weights", "", "Path to weights JSON file for score-biomes mode")
	wandType := flag.String("wand-type", "wand_level_01", "Wand type for wand mode")
	biome := flag.String("biome", "coalmine", "Biome for item/potion mode")
	flag.Parse()

	ws := uint32(*seed)

	switch *mode {
	case "great-chest":
		result := GenerateGreatChest(ws, *ng, *x, *y, false)
		printChest(result)

	case "chest":
		result := GenerateChest(ws, *ng, *x, *y, false, false)
		printChest(result)

	case "spawn-chest":
		result := SpawnChest(ws, *ng, *x, *y, false, false, false)
		printChest(result)

	case "wand":
		wand := GenerateWand(ws, *ng, *wandType, *x, *y)
		if wand == nil {
			fmt.Fprintf(os.Stderr, "Unknown wand type: %s\n", *wandType)
			os.Exit(1)
		}
		printWand(wand)

	case "wand-altar":
		item := SpawnWand(ws, *ng, *x, *y, *biome, false)
		if item == nil {
			fmt.Printf("No wand at (%.0f, %.0f) in biome %s for seed %d\n", *x, *y, *biome, *seed)
		} else {
			printItem(item)
		}

	case "potion-altar":
		item := SpawnPotionAltar(ws, *ng, *x, *y, *biome, "normal", false)
		if item == nil {
			fmt.Printf("No potion spawns at (%.0f, %.0f) in biome %s for seed %d\n", *x, *y, *biome, *seed)
		} else {
			printItem(item)
		}

	case "item":
		item := SpawnItem(ws, *ng, *x, *y, *biome, false)
		if item == nil {
			fmt.Println("No item")
		} else {
			printItem(item)
		}

	case "potion":
		item := createPotion(ws, *ng, *x, *y, "normal", "normal")
		printItem(item)

	case "pouch":
		item := createPowderPouch(ws, *ng, *x, *y)
		printItem(item)

	case "list-spawns", "list-coalmine":
		spawns, err := listNaturalSpawns(ws, *ng, *pwMax, *pwMaxV)
		if err != nil {
			fmt.Fprintf(os.Stderr, "list-spawns: %v\n", err)
			os.Exit(1)
		}
		if *spellSearch != "" {
			needle := strings.ToUpper(*spellSearch)
			var filtered []*Spawn
			for _, s := range spawns {
				if spawnContainsSpell(s, needle) {
					filtered = append(filtered, s)
				}
			}
			spawns = filtered
		}
		printSpawnList(*seed, spawns)

	case "score-biomes":
		if *weightsFile == "" {
			fmt.Fprintln(os.Stderr, "score-biomes: -weights flag required")
			os.Exit(1)
		}
		wc, err := loadWeights(*weightsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "score-biomes: load weights: %v\n", err)
			os.Exit(1)
		}
		spawns, err := listNaturalSpawns(ws, *ng, *pwMax, *pwMaxV)
		if err != nil {
			fmt.Fprintf(os.Stderr, "score-biomes: %v\n", err)
			os.Exit(1)
		}
		printBiomeScores(*seed, spawns, wc)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		flag.Usage()
		os.Exit(1)
	}
}

func printWand(w *Wand) {
	shuffle := "shuffle"
	if w.ShuffleDeckWhenEmpty == 0 {
		shuffle = "no-shuffle"
	}
	rare := ""
	if w.IsRare == 1 {
		rare = " [RARE]"
	}
	fmt.Printf("Wand%s: %s  type=%s  level=%d\n", rare, w.Name, w.WandType, w.Level)
	fmt.Printf("  capacity=%g  apm=%g  reload=%g  fire_rate=%g  spread=%g  speed=%g\n",
		w.DeckCapacity, w.ActionsPerRound, w.ReloadTime, w.FireRateWait, w.SpreadDegrees, w.SpeedMultiplier)
	fmt.Printf("  mana=%g/%g  %s  sprite=%s\n", w.ManaMax, w.ManaChargeSpeed, shuffle, w.Sprite)
	if len(w.AlwaysCasts) > 0 {
		fmt.Printf("  always_cast: %s\n", strings.Join(w.AlwaysCasts, ", "))
	}
	if len(w.Cards) > 0 {
		fmt.Printf("  cards: %s\n", strings.Join(w.Cards, ", "))
	}
}

func printWandDetail(w *Wand) {
	shuffle := "shuffle"
	if w.ShuffleDeckWhenEmpty == 0 {
		shuffle = "no-shuffle"
	}
	rare := ""
	if w.IsRare == 1 {
		rare = " [RARE]"
	}
	fmt.Printf("    Wand%s: %s  type=%s  level=%d\n", rare, w.Name, w.WandType, w.Level)
	fmt.Printf("    capacity=%g  apm=%g  reload=%g  fire_rate=%g  spread=%g  speed=%g\n",
		w.DeckCapacity, w.ActionsPerRound, w.ReloadTime, w.FireRateWait, w.SpreadDegrees, w.SpeedMultiplier)
	fmt.Printf("    mana=%g/%g  %s  sprite=%s\n", w.ManaMax, w.ManaChargeSpeed, shuffle, w.Sprite)
	if len(w.AlwaysCasts) > 0 {
		fmt.Printf("    always_cast: %s\n", strings.Join(w.AlwaysCasts, ", "))
	}
	if len(w.Cards) > 0 {
		fmt.Printf("    cards: %s\n", strings.Join(w.Cards, ", "))
	}
}

func printItem(item *Item) {
	if item == nil {
		fmt.Println("(none)")
		return
	}
	if item.ItemType == "wand" && item.Wand != nil {
		printWand(item.Wand)
		return
	}
	if item.Material != "" {
		fmt.Printf("%s [%s]  @ (%.1f, %.1f)\n", item.ItemType, item.Material, item.X, item.Y)
	} else if item.Spell != "" {
		fmt.Printf("%s [%s]  @ (%.1f, %.1f)\n", item.ItemType, item.Spell, item.X, item.Y)
	} else if item.Amount > 0 {
		fmt.Printf("%s x%d  @ (%.1f, %.1f)\n", item.ItemType, item.Amount, item.X, item.Y)
	} else {
		fmt.Printf("%s  @ (%.1f, %.1f)\n", item.ItemType, item.X, item.Y)
	}
}

func pwSuffix(s *Spawn) string {
	if s.PW == 0 && s.PWV == 0 {
		return ""
	}
	if s.PWV == 0 {
		return fmt.Sprintf(" pw=%d", s.PW)
	}
	return fmt.Sprintf(" pw=%d pwv=%d", s.PW, s.PWV)
}

func itemContainsSpell(it *Item, needle string) bool {
	if it == nil {
		return false
	}
	if it.Spell != "" && strings.Contains(strings.ToUpper(it.Spell), needle) {
		return true
	}
	if it.Wand != nil {
		for _, c := range it.Wand.Cards {
			if strings.Contains(strings.ToUpper(c), needle) {
				return true
			}
		}
		for _, c := range it.Wand.AlwaysCasts {
			if strings.Contains(strings.ToUpper(c), needle) {
				return true
			}
		}
	}
	return false
}

func spawnContainsSpell(s *Spawn, needle string) bool {
	if itemContainsSpell(s.Item, needle) {
		return true
	}
	if s.Chest != nil {
		for _, it := range s.Chest.Items {
			if itemContainsSpell(it, needle) {
				return true
			}
		}
	}
	return false
}

func printSpawnList(seed uint, spawns []*Spawn) {
	fmt.Printf("Natural spawns for seed %d: %d item(s)\n", seed, len(spawns))
	for _, s := range spawns {
		switch {
		case s.Chest != nil:
			fmt.Printf("  [%s] %s%s @ (%.0f, %.0f) — %d item(s)\n", s.Kind, s.Biome, pwSuffix(s), s.X, s.Y, len(s.Chest.Items))
			for _, it := range s.Chest.Items {
				fmt.Printf("      - ")
				printItem(it)
			}
		case s.Item != nil:
			if s.Item.ItemType == "wand" && s.Item.Wand != nil {
				fmt.Printf("  [%s] %s%s @ (%.0f, %.0f)\n", s.Kind, s.Biome, pwSuffix(s), s.X, s.Y)
				printWandDetail(s.Item.Wand)
			} else {
				fmt.Printf("  [%s] %s%s ", s.Kind, s.Biome, pwSuffix(s))
				printItem(s.Item)
			}
		case s.Kind == "pixel_scene":
			fmt.Printf("  [%s:%s] %s%s @ (%.0f, %.0f) — %s\n", s.Kind, s.FuncName, s.Biome, pwSuffix(s), s.X, s.Y, s.Note)
		default:
			fmt.Printf("  [%s] %s%s @ (%.0f, %.0f)\n", s.Kind, s.Biome, pwSuffix(s), s.X, s.Y)
		}
	}
}

func printBiomeScores(seed uint, spawns []*Spawn, wc WeightConfig) {
	type entry struct {
		biome string
		score float64
	}
	totals := map[string]float64{}
	for _, s := range spawns {
		sc := wc.scoreSpawn(s)
		if sc != 0 {
			totals[s.Biome] += sc
		}
	}

	var rows []entry
	var total float64
	for biome, sc := range totals {
		rows = append(rows, entry{biome, sc})
		total += sc
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		return rows[i].biome < rows[j].biome
	})

	fmt.Printf("Biome scores for seed %d:\n", seed)
	for _, r := range rows {
		fmt.Printf("  %-30s %g\n", r.biome, r.score)
	}
	fmt.Printf("  %-30s %g\n", "(total)", total)
}

func printChest(result *ChestResult) {
	fmt.Printf("[%s] @ (%.1f, %.1f) — %d item(s)\n", result.Type, result.X, result.Y, len(result.Items))
	for _, item := range result.Items {
		count := ""
		if item.Count > 1 {
			count = fmt.Sprintf(" x%d", item.Count)
		}
		fmt.Printf("  [%s]%s ", item.ItemType, count)
		if item.ItemType == "wand" && item.Wand != nil {
			w := item.Wand
			shuffle := "shuffle"
			if w.ShuffleDeckWhenEmpty == 0 {
				shuffle = "no-shuffle"
			}
			rare := ""
			if w.IsRare == 1 {
				rare = "[RARE] "
			}
			fmt.Printf("%s%s cap=%.0f apm=%.0f reload=%.0f fire=%0.f %s\n", rare, w.Name, w.DeckCapacity, w.ActionsPerRound, w.ReloadTime, w.FireRateWait, shuffle)
			if len(w.AlwaysCasts) > 0 {
				fmt.Printf("         always_cast: %s\n", strings.Join(w.AlwaysCasts, ", "))
			}
			if len(w.Cards) > 0 {
				fmt.Printf("         cards: %s\n", strings.Join(w.Cards, ", "))
			}
		} else if item.Material != "" {
			fmt.Printf("%s\n", item.Material)
		} else if item.Spell != "" {
			fmt.Printf("%s\n", item.Spell)
		} else if item.Amount > 0 {
			fmt.Printf("x%d\n", item.Amount)
		} else {
			fmt.Println()
		}
	}
}
