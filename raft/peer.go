package raft

import (
	"sync"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/utils"
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
	voteCount   int
	log         []LogEntry

	commitIndex int
	lastApplied int

	rpcPeersIds []rpccore.NodeID
	node        rpccore.Node
	shutdown    bool
	logger      *logrus.Entry

	// [false] for reset and [true] for trigger
	// don't use this one directly, use [triggerTimeout] or [resetTimeout]
	timeoutLoopChan chan (bool)

	// leader only
	nextIndex                    map[rpccore.NodeID]int
	matchIndex                   map[rpccore.NodeID]int
	appendingEntries             map[rpccore.NodeID]bool
	logIndexMajorityCheckChannel map[int]chan rpccore.NodeID

	// follower only
	heardFromLeader bool
}

func NewPeer(node rpccore.Node, peers []rpccore.NodeID, logger *logrus.Entry) *Peer {
	p := new(Peer)

	// initialisation
	// initialise leader only fields (nextIndex, matchIndex) when becoming leader
	p.currentTerm = 0
	p.votedFor = nil
	p.log = []LogEntry{{cmd: nil, term: 0}}

	p.commitIndex = 0
	p.lastApplied = 0

	p.rpcPeersIds = make([]rpccore.NodeID, len(peers))
	copy(p.rpcPeersIds, peers)

	p.node = node
	node.RegisterRawRequestCallback(p.handleRPCCallAndLogError)

	p.shutdown = true
	p.logger = logger
	// this channel can't be blocking, otherwise it will cause a dead lock
	p.timeoutLoopChan = make(chan bool, 1)

	p.appendingEntries = make(map[rpccore.NodeID]bool)

	p.changeState(Follower)

	return p
}

func (p *Peer) timeoutLoop() {
	for {
		// TODO: make [PeerState] atomic? we need to find out the cost of using
		// mutex https://golang.org/doc/diagnostics.html
		p.mutex.Lock()
		currentState := p.state
		p.mutex.Unlock()

		// timeout based on state
		var timeout time.Duration
		switch currentState {
		case Follower:
			timeout = time.Duration(utils.Random(200, 400))
		case Candidate:
			timeout = time.Duration(utils.Random(200, 400))
		case Leader:
			timeout = 50
		}

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
					p.changeState(Candidate)
				} else {
					p.heardFromLeader = false
				}
			case Leader:
				for peerID, appendingEntry := range p.appendingEntries {
					if !appendingEntry {
						go p.callAppendEntryRPC(peerID)
					}
				}
			}
		}
		shutdown := p.shutdown
		p.mutex.Unlock()
		if shutdown {
			break
		}
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
		p.voteCount = 1
		p.startElection()
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
		p.shutdown = false
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
	p.logger.Info("Start election.")
	voteID := p.node.NodeID()
	p.updateTerm(p.currentTerm + 1)
	p.votedFor = &voteID

	req := requestVoteReq{Term: p.currentTerm, CandidateID: p.node.NodeID(), LastLogIndex: len(p.log) - 1, LastLogTerm: p.log[len(p.log)-1].term}
	for _, peerID := range p.rpcPeersIds {
		go func(peerID rpccore.NodeID) {
			res := p.requestVote(peerID, req)
			if res != nil {
				p.handleRequestVoteRespond(*res)
			}
		}(peerID)
	}
}

func (p *Peer) updateTerm(term int) {
	if term > p.currentTerm {
		p.logger.Infof("Term is incremented from %v to %v.", p.currentTerm, term)
		p.currentTerm = term
		p.votedFor = nil
	}
}

func (p *Peer) GetTerm() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.currentTerm
}

func (p *Peer) GetState() PeerState {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.state
}
