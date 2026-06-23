# True orb in a greater chest

This is the chance a Noita **greater chest** give you a **true orb**, and what it mean once you try to find a seed with *more than one* of them. Everything here come from `GenerateGreatChest` in `chest.go`, plus an empirical count from the searcher.

Be carefull, same scope as the rest of the docs: first two zones (coalmine + excavationsite) of the **main** world, no parallel worlds.

## Where the orb come from

The orb is not part of the normal loot loop. It only show up in the "very special (~impossible)" branch at the top of the chest:

```go
// Very special (~impossible)
if p.random(0, 100000) >= 100000 {   // hit only on the value 100000  → 1/100001
    count = 0
    if p.random(0, 1000) == 999 {     // → 1/1001
        items = append(... "true_orb" ...)
    } else {
        items = append(... "sampo" ...)
    }
}
```

With `random(a,b) = a + int((b+1-a)·next())` the ranges are inclusive, so the two gates are `1/100001` and `1/1001`:

$$P(\text{true orb} \mid \text{great chest}) = \frac{1}{100001}\cdot\frac{1}{1001} = \frac{1}{100{,}101{,}001} \approx 1.0\times10^{-8}$$

So about **1 in 100.1 million greater chests**.

Be carefull about one thing: when this branch fire it set `count = 0` and append exactly **one** item (the orb *or* a sampo, never both). A single greater chest hold **at most one** true orb. So "3 orbs out of one chest" can't happen, you need **3 different greater chests** in the same seed each rolling the orb.

## What the searcher really measure

Over **400,000,000** seeds the searcher found **13** true orbs in greater chests (first two zones, main world). That give:

$$\lambda_{2\text{-zone}} = \frac{13}{400{,}000{,}000} = 3.25\times10^{-8} \text{ orb-chests per seed}$$

## Orbs per seed (first two zones)

Each greater chest roll on its own, and a chest hold at most one orb, therefor the orb count in a seed is Poisson with `λ = 3.25e-8`:

| event in one seed | probability | exp. seeds over 2³¹ |
|---|---:|---:|
| ≥1 orb-chest | ≈ 3.25e-8 | ~70 |
| ≥2 in one seed | λ²/2 ≈ 5.3e-16 | 1.1e-6 |
| ≥3 in one seed | λ³/6 ≈ 5.7e-24 | 1.2e-14 |

So ~70 seeds in the whole `2³¹` space have an orb-chest (this is just rescaling the 13/400M). But **2 in one seed** is already a ~1-in-a-million long shot *across the entire keyspace*, and **3 in one seed** is `1.2e-14`, it basically can't happen.

## Scaling to the whole main world

The orb chance per chest is fixed, so going to the whole world is only a question of *how many more greater chests* a full run have. The first two zones are only the top of the descent. Below them you still go through fungal caverns, snowy depths, hiisi base, underground jungle, the vault, temple of the art, the laboratory, then the deep/hell biomes under the lab, plus all the surface and side stuff (forest, lakes, desert, snowy wasteland, pyramid). That's a dozen-plus chest-bearing biomes, and the lower ones are *bigger* than the mines. The early zones are chest-dense so it's not a clean linear count, but a rough estimate is the whole main world hold around **20×** the greater chests of the first two zones.

Be carefull, this `M ≈ 20` is a rough estimate (probably somewhere ~15–30×), not something measured. The clean way is to count the greater chests directly once the all-biomes searcher can scan the whole main world, then `M` stop being a guess. Everything below scale with it (`λ ∝ M`).

$$\lambda_{\text{main}} = 3.25\times10^{-8} \times 20 = 6.5\times10^{-7} \text{ orb-chests/seed}$$

| event | λ (orbs/seed) | exp. seeds over 2³¹ |
|---|---:|---:|
| ≥1 orb-chest | 6.5e-7 | ~1,400 |
| ≥2 in one seed | — | 4.5e-4 |
| ≥3 in one seed | — | 9.8e-11 |

So even with the whole main world, ≥2 is still ~2,000× short of expecting a single seed, and ≥3 stay hopeless.

## How many parallel worlds you would "need"

Now the fun part. A parallel world is a full copy, so each one add another `λ_main`. With `W` worlds total, `λ_T = W · λ_main`. If you ask "how many worlds so that at least one seed like that plausibly exist somewhere in 2³¹", you set the expected count to **1**:

$$N_{\text{seeds}}\cdot\frac{\lambda_T^{\,k}}{k!} = 1 \;\Rightarrow\; \lambda_T = \left(\frac{k!}{2^{31}}\right)^{1/k}, \qquad W = \frac{\lambda_T}{\lambda_{\text{main}}}$$

| target | required λ_T | total worlds W | parallel worlds needed |
|---|---:|---:|---:|
| ≥2 in one seed | 3.05e-5 | ~47 | **~46** |
| ≥3 in one seed | 1.41e-3 | ~2,168 | **~2,167** |

So to make a **2-orb seed** merely *exist* in the keyspace you would need ~**46 parallel worlds** loaded and scanned, and for a **3-orb seed** ~**2,167**.

Now here is the interesting part: Noita can actually handle ~46 parallel worlds! So the **≥2 case is reachable**, it's not a fantasy. If you scan the main world plus its 46 parallel worlds, you expect about **1 seed** in the whole `2³¹` keyspace with two true-orb greater chests (maybe a small handfull, depend on the real `M`).

The **≥3 case stay out of reach** though: even with those 46 worlds the expected count is ~1e-5 over the whole keyspace, you would need ~2,167 worlds, and that is way past what the worldgen survive. The jump from 2 to 3 is ~46×, because the threshold grow as `(k!)^{1/k}` while you fight a `10⁻⁷`-scale per-world rate.
