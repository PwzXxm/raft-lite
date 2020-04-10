package main

import (
	"github.com/PwzXxm/raft-lite/raft"
)

func main()  {
    size := 5

    rf := raft.Run(size)
    defer rf.Stop()
}