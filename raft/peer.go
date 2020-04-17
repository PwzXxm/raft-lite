package raft

import (
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
	currentTerm int
	votedFor    int
	log         []LogEntry

	commitIndex int
	lastApplied int

	nextIndex  map[rpccore.NodeID]int
	matchIndex map[rpccore.NodeID]int

	rpcPeersIds []rpccore.NodeID
	node        rpccore.Node
	dead        bool
	logger      *logrus.Entry
}

func NewPeer(node rpccore.Node, peers []rpccore.NodeID) *Peer {
	p := new(Peer)

	// initialisation
	p.node = node
	node.RegisterRawRequestCallback(p.handleRpcCallAndLogError)

	return p
}

// Start fire up this peer
// may start after shutdown
func (p *Peer) Start() {
}

// ShutDown stop this peer from running
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
