package utils

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Random an integer within the range
func Random(a, b int) int {
	return rand.Intn(b-a+1) + a
}

func RandomFloat(a, b float64) float64 {
	return a + rand.Float64()*(b-a)
}

func RandomTime(a, b time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(b-a+1)) + int64(a))
}

func RandomBool(prob float64) bool {
	return rand.Float64() < prob
}
