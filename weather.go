package main

import "math"

// Weather ported from InfoProviders/Weather.ts.
// NOTE: Snow weather depends on real-world date, so we use the current month/day.

const (
	WeatherNone   = 0
	WeatherSnow   = 1
	WeatherLiquid = 2

	snowfallChance  = 1.0 / 12
	rainfallChance  = 1.0 / 15
)

type snowType struct {
	chance       float64
	rainMaterial string
	rainType     int
}

var snowTypes = []snowType{
	{1.0, "snow", WeatherSnow},
	{0.25, "slush", WeatherSnow},
}

type rainType struct {
	chance       float64
	rainMaterial string
	rainType     int
}

var rainTypes = []rainType{
	{1.0, "water", WeatherLiquid},
	{0.05, "water", WeatherLiquid},
	{0.001, "blood", WeatherLiquid},
	{0.0002, "acid", WeatherLiquid},
	{0.0001, "slime", WeatherLiquid},
}

// WeatherResult holds weather data for a seed.
type WeatherResult struct {
	RainType     int
	RainMaterial string
	Fog          float64
	Clouds       float64
}

// GetWeather computes weather for the current world seed and real-world time.
// month: 1-12, day: day of month (used for snowfall check).
func GetWeatherWithTime(rng *RNG, month, day, hour int) WeatherResult {
	rnd := randomCreate(7893434, 3458934)
	rndTime := randomCreate(int32(hour+day), int32(hour+day+1))

	snows1 := month >= 12
	snows2 := month <= 2
	snows := (snows1 || snows2) && rng.randomNext(&rndTime, 0.0, 1.0) <= snowfallChance
	rains := !snows && rng.randomNext(&rnd, 0.0, 1.0) <= rainfallChance

	result := WeatherResult{RainType: WeatherNone}

	if snows {
		result.RainType = WeatherSnow
		// pick_random_from_table_backwards for snow_types
		chances := make([]float64, len(snowTypes))
		for i, st := range snowTypes {
			chances[i] = st.chance
		}
		idx := pickRandomFromTableBackwardsIdx(chances, &rndTime, rng.worldSeed)
		result.RainMaterial = snowTypes[idx].rainMaterial
	} else if rains {
		result.RainType = WeatherLiquid
		chances := make([]float64, len(rainTypes))
		for i, rt := range rainTypes {
			chances[i] = rt.chance
		}
		idx := pickRandomFromTableBackwardsIdx(chances, &rnd, rng.worldSeed)
		result.RainMaterial = rainTypes[idx].rainMaterial
	}

	if result.RainType != WeatherNone {
		result.Fog = rng.randomNext(&rnd, 0.3, 0.85)
		result.Clouds = math.Max(result.Fog, rng.randomNext(&rnd, 0.0, 1.0))
	}

	return result
}

// GetWeather uses the current real-world time (month=1 so rarely snows by default).
// For seed searching, weather is typically searched with fixed params.
func GetWeather(rng *RNG) WeatherResult {
	// Use a neutral date that won't trigger snow (e.g., July)
	return GetWeatherWithTime(rng, 7, 1, 12)
}
