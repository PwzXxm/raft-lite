package simulation

import (
	"testing"
)

func TestLeader(t *testing.T) {
	rf := RunLocally(5)
	rf.Wait(3)

	// TODO: complete the test

	rf.StopAll()
}

func TestEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Should not accepting size zero")
		}
	}()

	RunLocally(0)
}
