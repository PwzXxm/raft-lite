/*
 * Project: raft-lite
 * ---------------------
 * Authors:
 *   Minjian Chen 813534
 *   Shijie Liu   813277
 *   Weizhi Xu    752454
 *   Wenqing Xue  813044
 *   Zijun Chen   813190
 */

package raft

import (
	"sync"
	"time"

	"github.com/PwzXxm/raft-lite/pstorage"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/sirupsen/logrus"
)

// PeerState state type of peer
type PeerState int

// PeerState: Follower (0), Candidate (1), Leader (2)
const (
	Follower PeerState = iota
	Candidate
	Leader
)

// log entry structure
type LogEntry struct {
	Cmd  interface{}
	Term int
}

// snapshot structure
type Snapshot struct {
	LastIncludedIndex    int
	LastIncludedTerm     int
	StateMachineSnapshot []byte
}

// peer structure
type Peer struct {
	state       PeerState       // peer state
	mutex       sync.Mutex      // mutex, for accessing data across multiple goroutines
	currentTerm int             // latest term (increases monotonically)
	votedFor    *rpccore.NodeID // candidateID that peer votes in this term, nil if none
	voteCount   int             // count for other peers vote me in this term
	log         []LogEntry      // log entries
	commitIndex int             // index of highest log entry known to be committed (increases monotonically)

	rpcPeersIds []rpccore.NodeID // array of other peer IDs, excludes self
	node        rpccore.Node     // node
	shutdown    bool             // bool value to check whether this peer starts
	startBefore bool             // bool value to check whether this peer was started before
	logger      *logrus.Entry    // logger

	// snapshot
	snapshot          *Snapshot
	snapshotThreshold int

	// state machine
	stateMachine      sm.StateMachine
	persistentStorage pstorage.PersistentStorage

	timingFactor int // read-only

	// don't use this one directly, use [triggerTimeout] or [resetTimeout]
	timeoutLoopChan          chan (struct{})
	timeoutLoopSkipThisRound bool

	// leader only
	nextIndex                    map[rpccore.NodeID]int // index of the next log entry to send to that server
	matchIndex                   map[rpccore.NodeID]int // index of highest log entry known to be replicated on server
	appendingEntries             map[rpccore.NodeID]bool
	logIndexMajorityCheckChannel map[int]chan rpccore.NodeID
	lastHeardFromFollower        map[rpccore.NodeID]time.Time

	// follower only
	heardFromLeader bool // follower becomes candidate if no connection with leader

	leaderID *rpccore.NodeID
}

// NewPeer creates a new peer
func NewPeer(node rpccore.Node, peers []rpccore.NodeID, logger *logrus.Entry,
	sm sm.StateMachine, ps pstorage.PersistentStorage, timingFactor int, snapshotThreshold int) (*Peer, error) {
	p := new(Peer)

	// initialisation
	// initialise leader only fields (nextIndex, matchIndex) when becoming leader
	p.currentTerm = 0
	p.votedFor = nil
	p.log = []LogEntry{{Cmd: nil, Term: 0}}

	p.commitIndex = 0

	p.rpcPeersIds = make([]rpccore.NodeID, len(peers))
	copy(p.rpcPeersIds, peers)

	p.node = node
	node.RegisterRawRequestCallback(p.handleRPCCallAndLogError)

	p.shutdown = true
	p.startBefore = false
	p.logger = logger
	p.stateMachine = sm
	p.snapshot = nil
	p.snapshotThreshold = snapshotThreshold
	p.stateMachine.Reset()
	p.timingFactor = timingFactor

	p.timeoutLoopChan = make(chan struct{}, 1)
	p.timeoutLoopSkipThisRound = false

	p.appendingEntries = make(map[rpccore.NodeID]bool, len(peers))
	for _, peer := range peers {
		p.appendingEntries[peer] = false
	}

	// try to recover from persistent storage
	p.persistentStorage = ps
	err := p.loadFromPersistentStorage()
	if err != nil {
		return nil, err
	}

	// peer is required to start from follower state
	p.changeState(Follower)
	return p, nil
}

// timeoutLoop loops the timeout
func (p *Peer) timeoutLoop() {
	for {
		p.mutex.Lock()
		currentState := p.state
		currentTerm := p.currentTerm
		//p.logger.Debugf("timeout loop before wait, state: %v, term: %v", currentState, currentTerm)
		p.mutex.Unlock()

		// timeout based on state
		var timeout time.Duration
		switch currentState {
		case Follower:
			timeout = time.Duration(utils.Random(400, 800) * p.timingFactor)
		case Candidate:
			timeout = time.Duration(utils.Random(400, 800) * p.timingFactor)
		case Leader:
			timeout = time.Duration(100 * p.timingFactor)
		}

		select {
		case <-time.After(timeout * time.Millisecond):
		case <-p.timeoutLoopChan:
		}
		p.mutex.Lock()
		//p.logger.Debugf("timeout loop after wait, state: %v, term: %v, skip: %v", p.state, p.currentTerm, p.timeoutLoopSkipThisRound)
		skipThisRound := p.timeoutLoopSkipThisRound
		p.timeoutLoopSkipThisRound = false
		// ignore this round if the state has been changed.
		if currentState == p.state && currentTerm == p.currentTerm && !skipThisRound {
			switch p.state {
			case Follower:
				if !p.heardFromLeader {
					p.changeState(Candidate)
				} else {
					p.heardFromLeader = false
				}
			case Leader:
				p.triggerLeaderHeartbeat()
			case Candidate:
				//p.logger.Debug("start election: timeout loop")
				p.startElection()
			}
		}
		//p.logger.Debug("AAA")
		//p.logger.Debug("BBB")
		shutdown := p.shutdown
		p.mutex.Unlock()
		if shutdown {
			break
		}
	}
}

func (p *Peer) triggerTimeout() {
	select {
	case p.timeoutLoopChan <- struct{}{}:
	default: // message dropped
	}
	p.timeoutLoopSkipThisRound = false
	//p.logger.Debug("trigger timeout")
}

func (p *Peer) resetTimeout() {
	select {
	case p.timeoutLoopChan <- struct{}{}:
	default: // message dropped
	}
	p.timeoutLoopSkipThisRound = true
	//p.logger.Debug("reset timeout")
}

func (p *Peer) changeState(state PeerState) {
	// when change to the same state
	if state == p.state {
		return
	}

	p.logger.Infof("Change from state: %v to state: %v.", p.state, state)

	switch p.state {
	// change from leader, remove some attributes
	case Leader:
		for _, c := range p.logIndexMajorityCheckChannel {
			close(c)
		}
		p.logIndexMajorityCheckChannel = nil
		p.lastHeardFromFollower = nil
		p.nextIndex = nil
		p.matchIndex = nil
	}

	p.state = state

	switch state {
	// change to other states
	case Follower:
		p.heardFromLeader = false
	case Candidate:
		p.leaderID = nil
		p.logger.Debug("start election: change state")
		p.startElection()
	case Leader:
		p.nextIndex = make(map[rpccore.NodeID]int, len(p.rpcPeersIds))
		p.matchIndex = make(map[rpccore.NodeID]int, len(p.rpcPeersIds))
		p.logIndexMajorityCheckChannel = make(map[int]chan rpccore.NodeID)
		p.lastHeardFromFollower = make(map[rpccore.NodeID]time.Time, len(p.rpcPeersIds))
		for _, peers := range p.rpcPeersIds {
			p.nextIndex[peers] = p.logLen()
			p.matchIndex[peers] = 0
			p.lastHeardFromFollower[peers] = time.Now()
		}
		p.triggerLeaderHeartbeat()
	}
	p.resetTimeout()
}

func (p *Peer) updateLastHeard(target rpccore.NodeID) {
	if p.lastHeardFromFollower != nil {
		p.lastHeardFromFollower[target] = time.Now()
	}
}

// isValidLeader checks if the peer is still a valid leader when received read request
// from client by counting how many followers it can communicate with.
func (p *Peer) isValidLeader() bool {
	if p.state != Leader {
		return false
	}
	now := time.Now()
	count := 1
	for _, lastHeard := range p.lastHeardFromFollower {
		if now.Sub(lastHeard) < time.Duration(250*p.timingFactor)*time.Millisecond {
			count++
		}
	}
	return 2*count > p.getTotalPeers()
}

// Start fire up this peer
func (p *Peer) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.shutdown {
		if p.startBefore {
			p.logger.Warn("Started before. You should new a new peer and load from pstorage.")
			return
		}

		p.logger.Info("Starting peer.")
		p.shutdown = false
		p.startBefore = true
		go p.timeoutLoop()
	} else {
		p.logger.Warning("This peer is already running.")
	}
}

// ShutDown stops this peer from running
func (p *Peer) ShutDown() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if !p.shutdown {
		p.shutdown = true
		p.logger.Info("Stopping peer.")
		p.resetTimeout()
		if p.logIndexMajorityCheckChannel != nil {
			for _, channel := range p.logIndexMajorityCheckChannel {
				close(channel)
			}
		}
	} else {
		p.logger.Warning("This peer is already stopped.")
	}
}

// updateTerm updates the term and resets the votedFor
func (p *Peer) updateTerm(term int) bool {
	if term > p.currentTerm {
		p.logger.Infof("Term is incremented from %v to %v.", p.currentTerm, term)
		p.currentTerm = term
		// clear votedFor in the new term
		p.votedFor = nil
		return true
	}
	return false
}

// updateCommitIndex updates the commit index
func (p *Peer) updateCommitIndex(idx int) {
	idx = utils.Min(idx, p.logLen()-1)
	if idx > p.commitIndex {
		for i := p.commitIndex + 1; i <= idx; i++ {
			action := p.log[p.toLogIndex(i)].Cmd
			if action != nil {
				_ = p.stateMachine.ApplyAction(action)
			}
			if p.toLogIndex(i)+1 >= p.snapshotThreshold {
				err := p.saveToSnapshot(i)
				if err != nil {
					p.logger.Errorf("Unable to save Snapshot: %+v.", err)
				}
			}
		}
		p.logger.Infof("CommitIndex is incremented from %v to %v.", p.commitIndex, idx)
		p.commitIndex = idx

		// update CommitIndex, Log, Snapshot
		p.saveToPersistentStorageAndLogError()
	}
}

// GetTerm returns the current term of this peer
func (p *Peer) GetTerm() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.currentTerm
}

// GetState returns the state of this peer
func (p *Peer) GetState() PeerState {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.state
}

// GetRestLog returns the log entries of this peer
func (p *Peer) GetRestLog() []LogEntry {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	var peerLog = make([]LogEntry, len(p.log))
	copy(peerLog, p.log)
	return peerLog
}

// GetNodeID returns the node ID of this peer
func (p *Peer) GetNodeID() rpccore.NodeID {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.node.NodeID()
}

// GetRecentSnapshot returns the latest snapshot of this peer
func (p *Peer) GetRecentSnapshot() *Snapshot {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.snapshot == nil {
		return nil
	} else {
		snapshot := *p.snapshot
		return &snapshot
	}
}

// TakeStateMachineSnapshot takes the snapshot and returns it
func (p *Peer) TakeStateMachineSnapshot() []byte {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	data, err := p.stateMachine.TakeSnapshot()
	if err != nil {
		p.logger.Errorf("Unable to get current state machine state, %+v", err)
		data = nil
	}
	return data
}

// GetVoteCount returns the vote count of this peer
func (p *Peer) GetVoteCount() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.voteCount
}

// getTotalPeers returns the total number of peers
func (p *Peer) getTotalPeers() int {
	return len(p.rpcPeersIds) + 1
}

// toLogIndex returns the corresponding log index
func (p *Peer) toLogIndex(trueIndex int) int {
	if p.snapshot == nil {
		return trueIndex
	}
	logidx := trueIndex - p.snapshot.LastIncludedIndex - 1
	return logidx
}

// logLen returns the corresponding log length
func (p *Peer) logLen() int {
	if p.snapshot == nil {
		return len(p.log)
	}
	return len(p.log) + p.snapshot.LastIncludedIndex + 1
}

// getLogTermByIndex returns the term of given index in log entries
func (p *Peer) getLogTermByIndex(trueIndex int) int {
	if p.snapshot != nil && trueIndex == p.snapshot.LastIncludedIndex {
		return p.snapshot.LastIncludedTerm
	}
	return p.log[p.toLogIndex(trueIndex)].Term
}
