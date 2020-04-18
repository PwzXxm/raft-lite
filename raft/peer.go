package raft

import (
	"sync"
	"time"

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
	state       PeerState
	mutex       sync.Mutex
	currentTerm int
	votedFor    *rpccore.NodeID
	log         []LogEntry

	commitIndex int
	lastApplied int

	rpcPeersIds []rpccore.NodeID
	node        rpccore.Node
	dead        bool
	logger      *logrus.Entry

	triggerTimeoutChan chan (int)
	shutdown           bool

	// leader only
	nextIndex  map[rpccore.NodeID]int
	matchIndex map[rpccore.NodeID]int

	// follower only
	heardFromLeader bool
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
	node.RegisterRawRequestCallback(p.handleRPCCallAndLogError)

	p.dead = false
	p.logger = logger

	p.changeState(Follower)

	p.shutdown = true

	return p
}

func (p *Peer) timeoutLoop() {
	for {
		// TODO: make [PeerState] atomic? we need to find out the cost of using
		// mutex https://golang.org/doc/diagnostics.html
		p.mutex.Lock()
		currentState := p.state
		p.mutex.Unlock()

		// TODO: get timeout based on state
		timeout := time.Duration(1000)
		select {
		case <-time.After(timeout * time.Millisecond):
		case _ = <-p.triggerTimeoutChan:
		}
		p.mutex.Lock()
		// ignore this round if the state has been changed.
		if currentState == p.state {
			// TODO: handle timeout here

		}
		if p.shutdown {
			break
		}
		p.mutex.Unlock()
	}
}

func (p *Peer) triggerTimeout() {
	p.triggerTimeoutChan <- 0
}

func (p *Peer) changeState(state PeerState) {
	p.state = state
	// TODO: init state here

	// restart the timeout
	p.triggerTimeout()
}

// Start fire up this peer
// TODO: handle starting after shutdown
func (p *Peer) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.shutdown {
		p.logger.Info("Starting peer.")
		go p.timeoutLoop()
	} else {
		p.logger.Warning("This peer is already running.")
	}
}

// ShutDown stop this peer from running
func (p *Peer) ShutDown() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.shutdown == false {
		p.shutdown = true
		p.logger.Info("Stopping peer.")
		p.triggerTimeout()
	} else {
		p.logger.Warning("This peer is already stopped.")
	}
}

func (p *Peer) startElection() {
}

func (p *Peer) runTimer() {
	// checkout Ticker
}

func (p *Peer) sendHeartBeats() {
	// send heartbeats to all peers
}
