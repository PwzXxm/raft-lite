package main

import (
	"github.com/PwzXxm/raft-lite/simulation"
)

func main() {
	size := 5

	rf := simulation.RunLocally(size)
	defer rf.Stop()
}
