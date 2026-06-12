package main

import (
	"testing"
)

// Benchmarks for each domain module to measure performance and validate cost estimates.
// Run with: go test -bench=. -benchmem -benchtime=10s
//
// Cost estimates (from rules.go):
// alchemy: 1, startingFlask: 1, startingSpell: 1, startingBombSpell: 1
// weather: 2, biomeModifier: 3, fungalShift: 10
// perk: 50, lottery: 55, wand: 150, shop: 800

func BenchmarkAlchemy(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetAlchemyResult(uint32(i))
	}
}

func BenchmarkStartingFlask(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetStartingFlask(checker.rng)
	}
}

func BenchmarkStartingSpell(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetStartingSpell(checker.rng)
	}
}

func BenchmarkStartingBombSpell(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetStartingBombSpell(checker.rng)
	}
}

func BenchmarkWeather(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetWeather(checker.rng)
	}
}

func BenchmarkBiomeModifier(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetBiomeModifiers(checker.rng)
	}
}

func BenchmarkFungalShift(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = PickFungal(uint32(i), 10)
	}
}

func BenchmarkPerkDeck(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetPerks(checker.rng)
	}
}

func BenchmarkPerkRow(b *testing.B) {
	rng := newRNG()
	it := NewPerkIterator(rng)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rng.SetWorldSeed(uint32(i))
		it = NewPerkIterator(rng)
		_ = it.NextRow()
	}
}

func BenchmarkWandSpawning(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = ProvideWand(checker.rng, 100, 100, 10, 1, false, false)
	}
}

func BenchmarkShopLevel(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = ProvideShopLevel(checker.rng, 1, false)
	}
}

func BenchmarkSpellGeneration(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = GetRandomAction(checker.rng, 100, 100, 1, 0)
	}
}

func BenchmarkAlwaysCasts(b *testing.B) {
	checker := newChecker()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = ProvideAlwaysCast(checker.rng, 1, 1, 7, 0)
	}
}

// Comparative benchmarks: combined rules
func BenchmarkRule_AlchemyAndWeather(b *testing.B) {
	checker := newChecker()
	alchemyRule := &RuleNode{
		Type: "alchemy",
		Val:  []byte(`{"LC":["water"],"AP":[]}`),
	}
	weatherRule := &RuleNode{
		Type: "weather",
		Val:  []byte(`{"id":"clear"}`),
	}
	rootRule := &RuleNode{
		Type:  RuleTypeAND,
		Rules: []*RuleNode{alchemyRule, weatherRule},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = checker.Check(rootRule)
	}
}

func BenchmarkRule_AlchemyAndPerk(b *testing.B) {
	checker := newChecker()
	alchemyRule := &RuleNode{
		Type: "alchemy",
		Val:  []byte(`{"LC":["water"],"AP":[]}`),
	}
	perkRule := &RuleNode{
		Type: "perk",
		Val:  []byte(`{"names":["PERKS_LOTTERY"],"match":"all"}`),
	}
	rootRule := &RuleNode{
		Type:  RuleTypeAND,
		Rules: []*RuleNode{alchemyRule, perkRule},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = checker.Check(rootRule)
	}
}

func BenchmarkRule_AlchemyAndShop(b *testing.B) {
	checker := newChecker()
	alchemyRule := &RuleNode{
		Type: "alchemy",
		Val:  []byte(`{"LC":["water"],"AP":[]}`),
	}
	shopRule := &RuleNode{
		Type: "shop",
		Val:  []byte(`{"poolId":1,"items":["WAND_FIREBALL"],"match":"any"}`),
	}
	rootRule := &RuleNode{
		Type:  RuleTypeAND,
		Rules: []*RuleNode{alchemyRule, shopRule},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = checker.Check(rootRule)
	}
}

func BenchmarkRule_PerkAndShop(b *testing.B) {
	checker := newChecker()
	perkRule := &RuleNode{
		Type: "perk",
		Val:  []byte(`{"names":["PERKS_LOTTERY"],"match":"all"}`),
	}
	shopRule := &RuleNode{
		Type: "shop",
		Val:  []byte(`{"poolId":1,"items":["WAND_FIREBALL"],"match":"any"}`),
	}
	rootRule := &RuleNode{
		Type:  RuleTypeAND,
		Rules: []*RuleNode{perkRule, shopRule},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		checker.SetSeed(uint32(i))
		_ = checker.Check(rootRule)
	}
}

// PerkShop optimized search
func BenchmarkPerkShopSearch(b *testing.B) {
	rng := newRNG()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rng.SetWorldSeed(uint32(i))
		_ = checkPerkShopSeed(rng)
	}
}
