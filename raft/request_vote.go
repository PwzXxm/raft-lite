package raft

import (
	"github.com/PwzXxm/raft-lite/rpccore"
)

func (p *Peer) handleRequestVote(term int, candidateID *rpccore.NodeID, lastLogIndex int, lastLogTerm int) (int, bool) {
	// check candidate's qualification: 1. Deny if its term is samller than mine 2. or my log is more up to date
	if term < p.currentTerm || p.logPriorCheck(lastLogIndex, lastLogTerm) {
		return p.currentTerm, false
	}
	// check receiver's qualification: Have not voted before, or voted to you before
	if !(p.votedFor == nil || p.votedFor == candidateID) {
		return p.currentTerm, false
	}
	// vote for this candidate
	// what else should this receiver do?
	p.votedFor = candidateID
	// change its current term to candidate's term
	p.currentTerm = term
	return p.currentTerm, true
}

func (p *Peer) logPriorCheck(lastLogIndex int, lastLogTerm int) bool {
	myLastLogIndex := len(p.log) - 1
	myLastLogTerm := p.log[myLastLogIndex].term
	switch {
	case myLastLogTerm > lastLogTerm:
		return true
	case myLastLogTerm == lastLogTerm:
		return myLastLogIndex > lastLogIndex
	default:
		return false
	}
}
