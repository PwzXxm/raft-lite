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

	// [false] for reset and [true] for trigger
	// don't use this one directly, ues [triggerTimeout] or [resetTimeout]
	timeoutLoopChan chan (bool)
	shutdown        bool

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
	// this channel can't be blocking, otherwise it will cause a dead lock
	p.timeoutLoopChan = make(chan bool, 1)

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

		execute := true
		select {
		case <-time.After(timeout * time.Millisecond):
		case execute = <-p.timeoutLoopChan:
		}
		p.mutex.Lock()
		// ignore this round if the state has been changed.
		if currentState == p.state && execute {
			// TODO: handle timeout here
			switch p.state {
			case Follower:
				if !p.heardFromLeader {
					// TODO: change to candidate?
				} else {
					p.heardFromLeader = false
				}
			}

		}
		if p.shutdown {
			break
		}
		p.mutex.Unlock()
	}
}

func (p *Peer) triggerTimeout() {
	p.timeoutLoopChan <- true
}

func (p *Peer) resetTimeout() {
	p.timeoutLoopChan <- false
}

func (p *Peer) changeState(state PeerState) {
	p.logger.Infof("Change from state: %v to state: %v.", p.state, state)
	p.state = state
	// TODO: init state here
	switch state {
	case Follower:
		p.heardFromLeader = false
	case Candidate:
	case Leader:
		p.nextIndex = make(map[rpccore.NodeID]int)
		p.matchIndex = make(map[rpccore.NodeID]int)
	}
	p.resetTimeout()
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
		p.resetTimeout()
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
