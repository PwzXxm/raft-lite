package raft

import "github.com/PwzXxm/raft-lite/rpccore"

type PeerState int

const (
	Follower PeerState = iota
	Candidate
	Leader
)

type LogEntry struct {
	cmd  interface{}
	term int
}

type Peer struct {
	currentTerm int
	votedFor    int
	log         []LogEntry

	commitIndex int
	lastApplied int

	nextIndex  map[int]int
	matchIndex map[int]int

	id       int
	peersIds []rpccore.NodeID
	node     *rpccore.Node
	dead     bool
}

func NewPeer(node *rpccore.Node, peers []rpccore.NodeID) *Peer {
	p := new(Peer)

	// initialisation

	return p
}

// start fire up a new peer in the network
// may start after shutdown
func (p *Peer) Start(id int) {
}

// shutDown stop this peer from running
func (p *Peer) ShutDown() {
}

func (p *Peer) startElection() {
}

func (p *Peer) runTimer() {
	// checkout Ticker
}

func (p *Peer) sendHeartBeats() {
	// send heartbeats to all peers
}
