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
	votedFor    *rpccore.NodeID
	log         []LogEntry

	commitIndex int
	lastApplied int

	// leader only
	nextIndex  map[rpccore.NodeID]int
	matchIndex map[rpccore.NodeID]int

	rpcPeersIds []rpccore.NodeID
	node        rpccore.Node
	dead        bool
	logger      *logrus.Entry
}

func NewPeer(node rpccore.Node, peers []rpccore.NodeID, logger *logrus.Entry) *Peer {
	p := new(Peer)

	// initialisation
	// initialise leader only fields (nextIndex, matchIndex) when becoming leader
	p.currentTerm = 0
	p.votedFor = nil
	p.log = make([]LogEntry, 0)

	p.commitIndex = 0
	p.lastApplied = 0

	p.rpcPeersIds = make([]rpccore.NodeID, len(peers))
	copy(p.rpcPeersIds, peers)

	p.node = node
	node.RegisterRawRequestCallback(p.handleRpcCallAndLogError)

	p.dead = false
	p.logger = logger

	return p
}

// Start fire up this peer
// TODO: handle starting after shutdown
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
