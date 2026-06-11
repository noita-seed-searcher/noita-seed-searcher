package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Specialized search for:
//   Row 0: PERKS_LOTTERY + EDIT_WANDS_EVERYWHERE + (NO_MORE_SHUFFLE OR UNLIMITED_SPELLS)
//   Lottery survival: one of {EDIT_WANDS_EVERYWHERE, NO_MORE_SHUFFLE, UNLIMITED_SPELLS}
//     must NOT be rerolled by lottery in any row 1+ (lotteries=1)
//   First shop: (CHAINSAW or MANA_REDUCE) AND (TELEPORT_PROJECTILE or TELEPORT_PROJECTILE_SHORT)

const (
	perkLottery   = "PERKS_LOTTERY"
	perkTinker    = "EDIT_WANDS_EVERYWHERE"
	perkNoShuffle = "NO_MORE_SHUFFLE"
	perkUnlimited = "UNLIMITED_SPELLS"
)

// checkPerkShopSeed is the hot path — all three conditions are inlined.
func checkPerkShopSeed(rng *RNG) bool {
	// Step 1: cheap lottery pre-filter on row 0 slot positions (no perk deck needed).
	// With 1 lottery each slot has 50% survival. We need PERKS_LOTTERY to survive
	// plus at least one other, so at least 2 of 3 slots must survive.
	// Rejects ~50% of all seeds before the expensive deck generation.
	const perksOnLevel = 3
	var slotSurvived [perksOnLevel]bool
	survivedCount := 0
	for perkNum := 0; perkNum < perksOnLevel; perkNum++ {
		if !lotteryIsRerolledFn(rng, 0, perkNum, perksOnLevel, 1) {
			slotSurvived[perkNum] = true
			survivedCount++
		}
	}
	if survivedCount < 2 {
		return false
	}

	// Step 2: build perk deck and generate row 0 (expensive).
	it := NewPerkIterator(rng)
	row0 := it.NextRow()
	if row0 == nil {
		return false
	}

	// Step 3: row 0 must have all three required perks, and specifically:
	// PERKS_LOTTERY must be in a surviving slot, plus at least one of the others.
	hasLottery, hasTinker, hasThird := false, false, false
	for perkNum, p := range row0 {
		if !slotSurvived[perkNum] {
			continue
		}
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
	fmt.Fprintf(os.Stderr, "  Row 0: PERKS_LOTTERY + EDIT_WANDS_EVERYWHERE + (NO_MORE_SHUFFLE or UNLIMITED_SPELLS)\n")
	fmt.Fprintf(os.Stderr, "  Lottery: PERKS_LOTTERY + at least one other must survive lottery at row 0 (50%% chance each)\n")
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
