package simulation

import (
	"github.com/PwzXxm/raft-lite/raft"
)

type local struct {
    n int
    peers map[int]*raft.Peer

    // TODO: add rpc network
    // rpc network may simulate long delays and unreliable communications
    // add a random waiting time?
}

func RunLocally(n int) *local {
    rf := new(local)
    rf.n = n

    // fire up individual peer

    // connect them to the network

    return rf
}

func (rf *local) Stop() {
}

func (rf *local) Request(cmd interface{}) {
}

func (rf *local) ShutDownPeer(id int) {
}

func (rf *local) ConnectPeer(id int) {
}

func (rf *local) DisconnectPeer(id int) {
}