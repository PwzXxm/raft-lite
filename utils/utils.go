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
