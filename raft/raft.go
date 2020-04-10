package raft

type raft struct {
    n int
    peers map[int]*Peer

    // TODO: add rpc network
    // rpc network may simulate long delays and unreliable communications
    // add a random waiting time?
}

func Run(n int) *raft {
    rf := new(raft)
    rf.n = n

    // fire up individual peer

    // connect them to the network

    return rf
}

func (rf *raft) Stop() {
}

func (rf *raft) Request(cmd interface{}) {
}

func (rf *raft) ShutDownPeer(id int) {
}

func (rf *raft) ConnectPeer(id int) {
}

func (rf *raft) DisconnectPeer(id int) {
}