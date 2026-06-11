package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Specialized search for:
//   Row 0: PERKS_LOTTERY + EDIT_WANDS_EVERYWHERE + (NO_MORE_SHUFFLE or UNLIMITED_SPELLS),
//     all three NOT lottery-replaced at level 0 (1 ticket).
//   First shop: (CHAINSAW or MANA_REDUCE) AND (TELEPORT_PROJECTILE or TELEPORT_PROJECTILE_SHORT)

const (
	perkLottery   = "PERKS_LOTTERY"
	perkTinker    = "EDIT_WANDS_EVERYWHERE"
	perkNoShuffle = "NO_MORE_SHUFFLE"
	perkUnlimited = "UNLIMITED_SPELLS"
)

// checkPerkShopSeed is the hot path — all conditions are inlined.
func checkPerkShopSeed(rng *RNG) bool {
	// Step 1: all 3 level-0 slots must survive the lottery check (1 ticket → probability=50).
	// Uses precomputed coordinates — no deck build needed.
	// Rejects ~87.5% of seeds (1 - 0.5³) before the expensive deck build.
	for i := range row0PerkX {
		rng.SetRandomSeed(row0PerkX[i], row0PerkY)
		if rng.RandomInt(1, 100) <= 50 {
			return false
		}
	}

	// Step 2: build deck and verify row 0 has the three target perks.
	it := NewPerkIterator(rng)
	row0 := it.NextRow()
	if row0 == nil {
		return false
	}
	hasLottery, hasTinker, hasThird := false, false, false
	for _, p := range row0 {
		switch p {
		case perkLottery:
			hasLottery = true
		case perkTinker:
			hasTinker = true
		case perkNoShuffle, perkUnlimited:
			hasThird = true
		}
	}
	if !hasLottery || !hasTinker || !hasThird {
		return false
	}

	// Step 3: first shop (level 0) must have both:
	//   - CHAINSAW or MANA_REDUCE
	//   - TELEPORT_PROJECTILE or TELEPORT_PROJECTILE_SHORT
	shop := ProvideShopLevel(rng, 0, false)
	hasChainsaw, hasTeleport := false, false
	if shop.Type == ShopTypeItem {
		for _, item := range shop.Items {
			switch item.SpellID {
			case "CHAINSAW", "MANA_REDUCE":
				hasChainsaw = true
			case "TELEPORT_PROJECTILE", "TELEPORT_PROJECTILE_SHORT":
				hasTeleport = true
			}
		}
	} else {
		for _, w := range shop.Wands {
			for _, card := range w.Wand.Cards.Cards {
				switch card {
				case "CHAINSAW", "MANA_REDUCE":
					hasChainsaw = true
				case "TELEPORT_PROJECTILE", "TELEPORT_PROJECTILE_SHORT":
					hasTeleport = true
				}
			}
		}
	}
	return hasChainsaw && hasTeleport
}

// RunPerkShopSearch runs the specialized three-stage search.
func RunPerkShopSearch(from, to uint32, workers int, printStats bool) {
	total := int64(to - from)
	fmt.Fprintf(os.Stderr, "PerkShop search: seeds %d..%d with %d workers\n", from, to, workers)
	fmt.Fprintf(os.Stderr, "  Row 0: PERKS_LOTTERY + EDIT_WANDS_EVERYWHERE + (NO_MORE_SHUFFLE or UNLIMITED_SPELLS), all not lottery-replaced (1 ticket)\n")
	fmt.Fprintf(os.Stderr, "  Shop 0: (CHAINSAW or MANA_REDUCE) AND (TELEPORT_PROJECTILE or TELEPORT_PROJECTILE_SHORT)\n")

	var (
		found      int64
		checked    int64
		foundSeeds = make(chan uint32, 1000)
		wg         sync.WaitGroup
	)

	startTime := time.Now()

	if printStats {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				c := atomic.LoadInt64(&checked)
				f := atomic.LoadInt64(&found)
				elapsed := time.Since(startTime).Seconds()
				rate := float64(c) / elapsed
				pct := float64(c) / float64(total) * 100
				remaining := float64(total-c) / rate
				fmt.Fprintf(os.Stderr, "\r[%.1f%%] checked %d / %d | found %d | %.0f seeds/s | ~%.0fs left    ",
					pct, c, total, f, rate, remaining)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		for seed := range foundSeeds {
			fmt.Println(seed)
			atomic.AddInt64(&found, 1)
		}
		close(done)
	}()

	chunkSize := int64(total) / int64(workers)
	if chunkSize < 1 {
		chunkSize = 1
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		wStart := from + uint32(int64(w)*chunkSize)
		wEnd := from + uint32(int64(w+1)*chunkSize)
		if w == workers-1 {
			wEnd = to
		}

		go func(start, end uint32) {
			defer wg.Done()
			rng := newRNG()
			for seed := start; seed < end; seed++ {
				rng.SetWorldSeed(seed)
				if checkPerkShopSeed(rng) {
					foundSeeds <- seed
				}
				atomic.AddInt64(&checked, 1)
			}
		}(wStart, wEnd)
	}

	wg.Wait()
	close(foundSeeds)
	<-done

	elapsed := time.Since(startTime)
	f := atomic.LoadInt64(&found)
	c := atomic.LoadInt64(&checked)
	fmt.Fprintf(os.Stderr, "\nDone: checked %d seeds in %s, found %d matches (%.0f seeds/s)\n",
		c, elapsed.Round(time.Millisecond), f, float64(c)/elapsed.Seconds())
}
