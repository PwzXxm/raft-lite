package raft

import (
	"sync"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/sirupsen/logrus"
)

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
	mutex       sync.Mutex
	currentTerm int
	votedFor    int
	log         []LogEntry

	commitIndex int
	lastApplied int

	nextIndex  map[int]int
	matchIndex map[int]int

	id       int
	peersIds []rpccore.NodeID
	node     rpccore.Node
	dead     bool

	logger *logrus.Entry
}

func NewPeer(node rpccore.Node, peers []rpccore.NodeID) *Peer {
	p := new(Peer)

	// initialisation
	p.node = node
	node.RegisterRawRequestCallback(p.handleRPCCallAndLogError)

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
