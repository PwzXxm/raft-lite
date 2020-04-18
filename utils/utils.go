package utils

import (
	"math/rand"
	"time"
)

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Random an integer within the range
func Random(a, b int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(b-a+1) + a
}
