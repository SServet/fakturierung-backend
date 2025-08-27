package utils

import "math"

// Round2 rounds x to 2 decimal places (banking-style simple round).
func Round2(x float64) float64 {
	return math.Round(x*100) / 100
}
