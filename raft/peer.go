package raft

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
	peersIds []int
	dead     bool
}

// start fire up a new peer in the network
func (p *Peer) start(id int) {
}

// shutDown stop this peer from running
func (p *Peer) shutDown() {
}

func (p *Peer) startElection() {
}

func (p *Peer) runTimer() {
	// checkout Ticker
}

func (p *Peer) sendHeartBeats() {
    // send heartbeats to all peers
}