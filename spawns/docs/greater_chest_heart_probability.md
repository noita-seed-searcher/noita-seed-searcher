# Greater-chest heart-count distribution

This is the probability that a Noita **greater chest** give you exactly *X* hearts, once you take into account the in-chest reroll (`count += 2` / `count += 3`).

The numbers come from `GenerateGreatChest` in `chest.go`. The theoretical values are the exact PRNG branch probabilities (we suppose `random()` is uniform), and the empirical values are counted directly from `gsc_200-400m.out` (every greater chest in **coalmine + excavationsite**, seeds 200,000,000–400,000,000, that's **828,950** chests).

Be carefull, this don't include the parallel worlds! Our searcher only look at the first two level (coalmine + excavationsite) of the **main** world, so everything here is for the main world only.

## Model

On each draw (one `count--` iteration), the chest roll a `random(1,100)`:

| outcome | rnd range | prob | effect |
|---|---|---|---|
| heart reward | 40–60 | 0.21 | +1 heart, terminal |
| other item (potion/gold/wand/stone) | 1–39 | 0.39 | terminal, no heart |
| reroll `count += 2` | 61–98 | 0.38 | +2 new slots |
| reroll `count += 3` | 99–100 | 0.02 | +3 new slots |

So it's a subcritical branching process (the mean offspring is `2·0.38 + 3·0.02 = 0.82 < 1`, therefor it always stop). The per-slot heart PGF is `G(s) = a·s + b + 0.38·G(s)² + 0.02·G(s)³`.

The heart branch itself split into `heart` (89%), `heart_bigger` (10%), `full_heal` (1%), so you have two way to read "heart":

- **All heart pickups** (incl. full_heal): `a = 0.21`, mean ≈ 1.1667
- **HP-raising only** (`heart`+`heart_bigger`, matches the searcher tag): `a = 0.2079`, mean ≈ 1.1550

## Cross-check vs `gsc_200-400m.out`

So on a big sample of **828,950** greater chests (seeds 200M–400M), the empirical mean is **1.1548**, against **1.1550** for the model (HP-raising). That's basically dead on, it's a really good sign. This 200M-seed run replace the older 40,991-chest cross-check (from `gsc_20-30m.out`, mean 1.1479): with 20× more chests the noise mostly vanish and every bin line up nice with the model.

| X | empirical count | empirical P | model P (HP-raising) |
|---:|---:|---:|---:|
| 0 | 400841 | 0.483553 | 0.483005 |
| 1 | 278573 | 0.336055 | 0.335909 |
| 2 | 61497 | 0.074187 | 0.074561 |
| 3 | 28154 | 0.033963 | 0.034325 |
| 4 | 16296 | 0.019659 | 0.019727 |
| 5 | 10475 | 0.012636 | 0.012697 |
| 6 | 7247 | 0.008742 | 0.008755 |
| 7 | 5274 | 0.006362 | 0.006324 |
| 8 | 3883 | 0.004684 | 0.004724 |
| 9 | 3012 | 0.003634 | 0.003619 |
| 10 | 2364 | 0.002852 | 0.002828 |
| 11 | 1898 | 0.002290 | 0.002246 |
| 12 | 1545 | 0.001864 | 0.001806 |
| 13 | 1270 | 0.001532 | 0.001469 |
| 14 | 1048 | 0.001264 | 0.001206 |
| 15 | 858 | 0.001035 | 0.000998 |

The cumulative tail match too: P(≥1) is **0.5164** empirical vs **0.5170** model, P(≥5) **0.05258** vs **0.05247**, P(≥10) **0.01653** vs **0.01635**. The deepest chest in this window reach **X = 109** hearts (HP-raising), but be carefull, past there you don't really see anything, it's just the model extrapolating.

## Scaling to the seed space (1 … 2³¹)

You can scale this two way, both for the **HP-raising** reading:

- **Naive** `P × 2³¹` (2,147,483,648): here you suppose *one greater chest per seed*. It's an upper bound, a bit optimistic.

- **Realistic** `P × N_chests`: this one is based on the rate the searcher really measure, **0.004099 greater chests/seed** in coalmine+excavationsite. That give you **N_chests ≈ 8,803,000** greater chests over the whole seed space. This is the count that match what your searcher actually scan.

**Where the count of qualifying chests fall below 1 (HP-raising, cumulative ≥ X):**

| framing | last X with ≥1 | next X |
|---|---:|---:|
| naive `P(≥X)·2³¹` | **173** (1.03) | 174 (0.94) |
| realistic `P(≥X)·N_chests` | **114** (1.08) | 115 (0.98) |

So in practice, the **biggest heart count you can realistically hope to find** in a first-two-zones greater chest is around **114** hearts somewhere in the whole seed space. Be carefull, the naive one-chest-per-seed reading push this up to **173**, but this is a bit of a lie, since our searcher don't scan one chest per seed!

## Full distribution, X = 0…100

Here `est ≥X HP (naive)` = P(≥X)·2³¹, and `est ≥X HP (real)` = P(≥X)·N_chests (the one you want to trust).

| X | P(=X) all | P(≥X) all | P(=X) HP | P(≥X) HP | est =X HP naive | est ≥X HP naive | est ≥X HP real |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 0 | 4.796e-01 | 1.000e+00 | 4.830e-01 | 1.000e+00 | 1,037,246,370 | 2,147,483,648 | 8,803,000 |
| 1 | 3.378e-01 | 5.204e-01 | 3.359e-01 | 5.170e-01 | 721,358,439 | 1,110,237,278 | 4,551,103 |
| 2 | 7.503e-02 | 1.826e-01 | 7.456e-02 | 1.811e-01 | 160,118,514 | 388,878,839 | 1,594,099 |
| 3 | 3.457e-02 | 1.076e-01 | 3.433e-02 | 1.065e-01 | 73,712,593 | 228,760,325 | 937,738.1 |
| 4 | 1.988e-02 | 7.300e-02 | 1.973e-02 | 7.220e-02 | 42,364,154 | 155,047,732 | 635,574.2 |
| 5 | 1.281e-02 | 5.311e-02 | 1.270e-02 | 5.247e-02 | 27,265,666 | 112,683,578 | 461,914.4 |
| 6 | 8.838e-03 | 4.031e-02 | 8.755e-03 | 3.978e-02 | 18,800,826 | 85,417,912 | 350,146.5 |
| 7 | 6.389e-03 | 3.147e-02 | 6.324e-03 | 3.102e-02 | 13,581,008 | 66,617,085 | 273,077.9 |
| 8 | 4.777e-03 | 2.508e-02 | 4.724e-03 | 2.470e-02 | 10,144,739 | 53,036,077 | 217,406.4 |
| 9 | 3.662e-03 | 2.030e-02 | 3.619e-03 | 1.997e-02 | 7,772,168 | 42,891,339 | 175,820.9 |
| 10 | 2.864e-03 | 1.664e-02 | 2.828e-03 | 1.635e-02 | 6,073,537 | 35,119,171 | 143,961.1 |
| 11 | 2.276e-03 | 1.378e-02 | 2.246e-03 | 1.353e-02 | 4,822,275 | 29,045,633 | 119,064.3 |
| 12 | 1.832e-03 | 1.150e-02 | 1.806e-03 | 1.128e-02 | 3,879,166 | 24,223,359 | 99,296.8 |
| 13 | 1.491e-03 | 9.667e-03 | 1.469e-03 | 9.474e-03 | 3,154,791 | 20,344,193 | 83,395.2 |
| 14 | 1.225e-03 | 8.175e-03 | 1.206e-03 | 8.004e-03 | 2,589,582 | 17,189,402 | 70,463.1 |
| 15 | 1.015e-03 | 6.950e-03 | 9.977e-04 | 6.799e-03 | 2,142,639 | 14,599,820 | 59,847.8 |
| 16 | 8.460e-04 | 5.936e-03 | 8.313e-04 | 5.801e-03 | 1,785,144 | 12,457,182 | 51,064.7 |
| 17 | 7.097e-04 | 5.090e-03 | 6.968e-04 | 4.970e-03 | 1,496,347 | 10,672,037 | 43,747.0 |
| 18 | 5.986e-04 | 4.380e-03 | 5.872e-04 | 4.273e-03 | 1,261,013 | 9,175,690 | 37,613.1 |
| 19 | 5.072e-04 | 3.781e-03 | 4.972e-04 | 3.686e-03 | 1,067,775 | 7,914,677 | 32,444.0 |
| 20 | 4.317e-04 | 3.274e-03 | 4.228e-04 | 3.188e-03 | 908,023.8 | 6,846,902 | 28,066.9 |
| 21 | 3.688e-04 | 2.843e-03 | 3.610e-04 | 2.766e-03 | 775,154.0 | 5,938,878 | 24,344.7 |
| 22 | 3.162e-04 | 2.474e-03 | 3.092e-04 | 2.405e-03 | 664,040.5 | 5,163,724 | 21,167.2 |
| 23 | 2.720e-04 | 2.157e-03 | 2.657e-04 | 2.095e-03 | 570,664.0 | 4,499,684 | 18,445.2 |
| 24 | 2.346e-04 | 1.886e-03 | 2.290e-04 | 1.830e-03 | 491,843.5 | 3,929,020 | 16,105.9 |
| 25 | 2.029e-04 | 1.651e-03 | 1.979e-04 | 1.601e-03 | 425,040.1 | 3,437,176 | 14,089.7 |
| 26 | 1.759e-04 | 1.448e-03 | 1.715e-04 | 1.403e-03 | 368,211.9 | 3,012,136 | 12,347.4 |
| 27 | 1.529e-04 | 1.272e-03 | 1.489e-04 | 1.231e-03 | 319,704.8 | 2,643,924 | 10,838.0 |
| 28 | 1.331e-04 | 1.119e-03 | 1.295e-04 | 1.082e-03 | 278,171.1 | 2,324,219 | 9,527.5 |
| 29 | 1.161e-04 | 9.861e-04 | 1.129e-04 | 9.528e-04 | 242,505.5 | 2,046,048 | 8,387.2 |
| 30 | 1.015e-04 | 8.700e-04 | 9.863e-05 | 8.398e-04 | 211,797.2 | 1,803,543 | 7,393.1 |
| 31 | 8.888e-05 | 7.685e-04 | 8.628e-05 | 7.412e-04 | 185,291.4 | 1,591,745 | 6,524.9 |
| 32 | 7.794e-05 | 6.796e-04 | 7.560e-05 | 6.549e-04 | 162,360.4 | 1,406,454 | 5,765.4 |
| 33 | 6.845e-05 | 6.017e-04 | 6.635e-05 | 5.793e-04 | 142,479.2 | 1,244,094 | 5,099.8 |
| 34 | 6.020e-05 | 5.332e-04 | 5.830e-05 | 5.130e-04 | 125,207.6 | 1,101,614 | 4,515.8 |
| 35 | 5.302e-05 | 4.730e-04 | 5.130e-05 | 4.547e-04 | 110,174.8 | 976,406.8 | 4,002.5 |
| 36 | 4.675e-05 | 4.200e-04 | 4.520e-05 | 4.034e-04 | 97,067.4 | 866,232.1 | 3,550.9 |
| 37 | 4.127e-05 | 3.733e-04 | 3.987e-05 | 3.582e-04 | 85,619.9 | 769,164.7 | 3,153.0 |
| 38 | 3.647e-05 | 3.320e-04 | 3.521e-05 | 3.183e-04 | 75,606.4 | 683,544.7 | 2,802.0 |
| 39 | 3.227e-05 | 2.955e-04 | 3.112e-05 | 2.831e-04 | 66,834.4 | 607,938.3 | 2,492.1 |
| 40 | 2.857e-05 | 2.632e-04 | 2.754e-05 | 2.520e-04 | 59,139.2 | 541,103.9 | 2,218.1 |
| 41 | 2.533e-05 | 2.347e-04 | 2.439e-05 | 2.244e-04 | 52,379.7 | 481,964.7 | 1,975.7 |
| 42 | 2.247e-05 | 2.093e-04 | 2.162e-05 | 2.000e-04 | 46,434.8 | 429,585.0 | 1,761.0 |
| 43 | 1.995e-05 | 1.869e-04 | 1.919e-05 | 1.784e-04 | 41,200.1 | 383,150.1 | 1,570.6 |
| 44 | 1.773e-05 | 1.669e-04 | 1.704e-05 | 1.592e-04 | 36,585.5 | 341,950.1 | 1,401.7 |
| 45 | 1.577e-05 | 1.492e-04 | 1.514e-05 | 1.422e-04 | 32,513.2 | 305,364.6 | 1,251.8 |
| 46 | 1.404e-05 | 1.334e-04 | 1.347e-05 | 1.271e-04 | 28,915.9 | 272,851.3 | 1,118.5 |
| 47 | 1.250e-05 | 1.194e-04 | 1.198e-05 | 1.136e-04 | 25,735.1 | 243,935.4 | 999.9 |
| 48 | 1.115e-05 | 1.069e-04 | 1.067e-05 | 1.016e-04 | 22,919.8 | 218,200.4 | 894.5 |
| 49 | 9.941e-06 | 9.572e-05 | 9.512e-06 | 9.093e-05 | 20,426.0 | 195,280.5 | 800.5 |
| 50 | 8.872e-06 | 8.578e-05 | 8.482e-06 | 8.142e-05 | 18,215.0 | 174,854.5 | 716.8 |
| 51 | 7.923e-06 | 7.691e-05 | 7.568e-06 | 7.294e-05 | 16,253.2 | 156,639.5 | 642.1 |
| 52 | 7.080e-06 | 6.899e-05 | 6.757e-06 | 6.537e-05 | 14,511.2 | 140,386.3 | 575.5 |
| 53 | 6.329e-06 | 6.191e-05 | 6.036e-06 | 5.862e-05 | 12,963.1 | 125,875.1 | 516.0 |
| 54 | 5.662e-06 | 5.558e-05 | 5.395e-06 | 5.258e-05 | 11,586.4 | 112,912.1 | 462.9 |
| 55 | 5.067e-06 | 4.992e-05 | 4.825e-06 | 4.718e-05 | 10,361.3 | 101,325.7 | 415.4 |
| 56 | 4.537e-06 | 4.485e-05 | 4.317e-06 | 4.236e-05 | 9,270.4 | 90,964.4 | 372.9 |
| 57 | 4.065e-06 | 4.031e-05 | 3.864e-06 | 3.804e-05 | 8,298.3 | 81,694.1 | 334.9 |
| 58 | 3.643e-06 | 3.625e-05 | 3.461e-06 | 3.418e-05 | 7,431.6 | 73,395.8 | 300.9 |
| 59 | 3.267e-06 | 3.260e-05 | 3.101e-06 | 3.072e-05 | 6,658.5 | 65,964.1 | 270.4 |
| 60 | 2.931e-06 | 2.934e-05 | 2.779e-06 | 2.762e-05 | 5,968.3 | 59,305.7 | 243.1 |
| 61 | 2.630e-06 | 2.641e-05 | 2.492e-06 | 2.484e-05 | 5,352.0 | 53,337.3 | 218.6 |
| 62 | 2.361e-06 | 2.378e-05 | 2.236e-06 | 2.234e-05 | 4,801.2 | 47,985.3 | 196.7 |
| 63 | 2.121e-06 | 2.142e-05 | 2.006e-06 | 2.011e-05 | 4,308.9 | 43,184.1 | 177.0 |
| 64 | 1.906e-06 | 1.929e-05 | 1.801e-06 | 1.810e-05 | 3,868.4 | 38,875.2 | 159.4 |
| 65 | 1.713e-06 | 1.739e-05 | 1.618e-06 | 1.630e-05 | 3,474.3 | 35,006.8 | 143.5 |
| 66 | 1.540e-06 | 1.568e-05 | 1.454e-06 | 1.468e-05 | 3,121.5 | 31,532.4 | 129.3 |
| 67 | 1.385e-06 | 1.414e-05 | 1.306e-06 | 1.323e-05 | 2,805.5 | 28,410.9 | 116.5 |
| 68 | 1.247e-06 | 1.275e-05 | 1.175e-06 | 1.192e-05 | 2,522.3 | 25,605.5 | 105.0 |
| 69 | 1.122e-06 | 1.150e-05 | 1.056e-06 | 1.075e-05 | 2,268.4 | 23,083.2 | 94.6 |
| 70 | 1.010e-06 | 1.038e-05 | 9.503e-07 | 9.693e-06 | 2,040.7 | 20,814.8 | 85.3 |
| 71 | 9.098e-07 | 9.372e-06 | 8.552e-07 | 8.742e-06 | 1,836.5 | 18,774.1 | 77.0 |
| 72 | 8.197e-07 | 8.462e-06 | 7.698e-07 | 7.887e-06 | 1,653.2 | 16,937.6 | 69.4 |
| 73 | 7.386e-07 | 7.642e-06 | 6.932e-07 | 7.117e-06 | 1,488.6 | 15,284.4 | 62.7 |
| 74 | 6.658e-07 | 6.904e-06 | 6.243e-07 | 6.424e-06 | 1,340.8 | 13,795.9 | 56.6 |
| 75 | 6.004e-07 | 6.238e-06 | 5.625e-07 | 5.800e-06 | 1,208.0 | 12,455.1 | 51.1 |
| 76 | 5.415e-07 | 5.638e-06 | 5.069e-07 | 5.237e-06 | 1,088.6 | 11,247.1 | 46.1 |
| 77 | 4.885e-07 | 5.096e-06 | 4.570e-07 | 4.730e-06 | 981.3 | 10,158.5 | 41.6 |
| 78 | 4.408e-07 | 4.608e-06 | 4.120e-07 | 4.273e-06 | 884.8 | 9,177.2 | 37.6 |
| 79 | 3.979e-07 | 4.167e-06 | 3.716e-07 | 3.861e-06 | 798.0 | 8,292.4 | 34.0 |
| 80 | 3.592e-07 | 3.769e-06 | 3.352e-07 | 3.490e-06 | 719.9 | 7,494.4 | 30.7 |
| 81 | 3.244e-07 | 3.410e-06 | 3.025e-07 | 3.155e-06 | 649.5 | 6,774.5 | 27.8 |
| 82 | 2.930e-07 | 3.085e-06 | 2.730e-07 | 2.852e-06 | 586.2 | 6,125.0 | 25.1 |
| 83 | 2.647e-07 | 2.792e-06 | 2.464e-07 | 2.579e-06 | 529.2 | 5,538.8 | 22.7 |
| 84 | 2.392e-07 | 2.528e-06 | 2.225e-07 | 2.333e-06 | 477.8 | 5,009.6 | 20.5 |
| 85 | 2.162e-07 | 2.288e-06 | 2.009e-07 | 2.110e-06 | 431.5 | 4,531.7 | 18.6 |
| 86 | 1.955e-07 | 2.072e-06 | 1.815e-07 | 1.909e-06 | 389.8 | 4,100.2 | 16.8 |
| 87 | 1.767e-07 | 1.877e-06 | 1.640e-07 | 1.728e-06 | 352.2 | 3,710.4 | 15.2 |
| 88 | 1.598e-07 | 1.700e-06 | 1.482e-07 | 1.564e-06 | 318.3 | 3,358.2 | 13.8 |
| 89 | 1.446e-07 | 1.540e-06 | 1.339e-07 | 1.416e-06 | 287.6 | 3,040.0 | 12.5 |
| 90 | 1.308e-07 | 1.396e-06 | 1.211e-07 | 1.282e-06 | 260.0 | 2,752.3 | 11.3 |
| 91 | 1.184e-07 | 1.265e-06 | 1.095e-07 | 1.161e-06 | 235.1 | 2,492.3 | 10.2 |
| 92 | 1.071e-07 | 1.146e-06 | 9.901e-08 | 1.051e-06 | 212.6 | 2,257.2 | 9.3 |
| 93 | 9.699e-08 | 1.039e-06 | 8.956e-08 | 9.521e-07 | 192.3 | 2,044.6 | 8.4 |
| 94 | 8.781e-08 | 9.422e-07 | 8.102e-08 | 8.625e-07 | 174.0 | 1,852.2 | 7.6 |
| 95 | 7.952e-08 | 8.544e-07 | 7.331e-08 | 7.815e-07 | 157.4 | 1,678.2 | 6.9 |
| 96 | 7.202e-08 | 7.749e-07 | 6.634e-08 | 7.082e-07 | 142.5 | 1,520.8 | 6.2 |
| 97 | 6.524e-08 | 7.029e-07 | 6.005e-08 | 6.418e-07 | 129.0 | 1,378.3 | 5.7 |
| 98 | 5.911e-08 | 6.376e-07 | 5.436e-08 | 5.818e-07 | 116.7 | 1,249.4 | 5.1 |
| 99 | 5.356e-08 | 5.785e-07 | 4.922e-08 | 5.274e-07 | 105.7 | 1,132.6 | 4.6 |
| 100 | 4.854e-08 | 5.250e-07 | 4.457e-08 | 4.782e-07 | 95.7 | 1,026.9 | 4.2 |

> Past X≈3 the tail is more or less geometric, with a ratio ≈0.78. Be carefull with a few things: we suppose the PRNG is uniform; the realistic column suppose the greater chests are independent between seeds (true for the rare tail events); and empirically we only saw X≤109 in the 200M-seed window, so the deeper-tail values are extrapolation, you don't really see them!

## What else you can find in a great chest

The hearts are not the only interesting thing in a great chest, so here is the empirical chance for some other stuff. Everything is counted over the **126,680** great chests pooled from the four dumps (`gsc_0-10m.out` is partial, it only reach seed ~490k, but pooling the chests is still fine since each chest is independent of the others). Same scope as the rest of this doc: first two zones of the **main** world, no parallel worlds.

Two columns: `chance / great chest` is the one you want once you already found a great chest, and `chance / seed` use the measured rate of **0.004153 great chests/seed** to give you the odds per seed.

Be carefull, there is **two** hp_regen potions and they are not the same! The common `magic_liquid_hp_regeneration_unstable` is *not* the good one (it heal but it's unstable). The real good one is the stable `magic_liquid_hp_regeneration`, and it's way rarer, around 1 in 688 chests.

| item | chests with it | chance / great chest | ≈ 1 in N chests | chance / seed (main, 2 zones) | ≈ 1 in N seeds |
|---|---:|---:|---:|---:|---:|
| HP-regen potion (unstable, the common one) | 3334 | 2.6318% | 38 | 0.01093% | 9,149 |
| HP-regen potion (stable — the *good* one) | 184 | 0.1452% | 688 | 0.00060% | 165,783 |
| Purifying powder potion | 3185 | 2.5142% | 40 | 0.01044% | 9,577 |
| Berserkium potion | 1122 | 0.8857% | 113 | 0.00368% | 27,187 |
| Polymorphine potion | 1097 | 0.8660% | 115 | 0.00360% | 27,807 |
| Invisiblium potion | 1045 | 0.8249% | 121 | 0.00343% | 29,190 |
| Midas potion (turn to gold) | 149 | 0.1176% | 850 | 0.00049% | 204,725 |
| Void liquid potion | 156 | 0.1231% | 812 | 0.00051% | 195,538 |
| Monster powder (`monster_powder_test`) potion | 145 | 0.1145% | 874 | 0.00048% | 210,372 |
| Vuoksikivi (tablet) | 15562 | 12.2845% | 8 | 0.05102% | 1,960 |
| Bigger heart (`heart_bigger`) | 11868 | 9.3685% | 11 | 0.03891% | 2,570 |
| Full-heal heart | 1376 | 1.0862% | 92 | 0.00451% | 22,169 |
| Kakkakikkare | 685 | 0.5407% | 185 | 0.00225% | 44,531 |
| Sampo | 164 | 0.1295% | 772 | 0.00054% | 186,000 |
| True orb (`true_orb`) | 1 | 0.0008% | 126,680 | 3.28e-08 | 30,503,985 |

The `sampo` and `true_orb` both come from the same "very special" branch in `GenerateGreatChest` (the `random(0,100000) >= 100000` roll, then `random(0,1000) == 999` pick the orb instead of the sampo). On paper, with a uniform PRNG, that branch should fire only 1 time in ~100001 chests. But empirically the sampo show up **1 in 772**, way more often! The reason is that this branch is seeded by the chest position `(x, y)`, and the Noita position-RNG is biased for some coordinates — you can see the same chest position (like `(855, 897)`) give a sampo on plenty of different seeds. So this is a good real example of why the "uniform `random()`" supposition is not always true, be carefull with it. The `true_orb` need the extra `== 999` roll on top, so it stay a meme: we only saw it **once** in 126,680 chests.

For the potions, remember a great chest give them by triple, so when you hit the potion branch you often get 2 of the same plus one extra.
