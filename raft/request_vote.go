package raft

import (
	"github.com/PwzXxm/raft-lite/rpccore"
)

func (p *Peer) startRequestVote() {
	req := requestVoteReq{Term: p.currentTerm, CandidateID: p.node.NodeID(), LastLogIndex: len(p.log) - 1, LastLogTerm: p.log[len(p.log)-1].term}

	for _, peerID := range p.rpcPeersIds {
		go func(peerID rpccore.NodeID) {
			res := p.requestVote(peerID, req)
			p.handleRequestVoteRespond(res.Term, res.VoteGranted)
		}(peerID)
	}
}

func (p *Peer) handleRequestVote(term int, candidateID rpccore.NodeID, lastLogIndex int, lastLogTerm int) requestVoteRes {
	// check candidate's qualification: 1. Deny if its term is samller than mine 2. or my log is more up to date
	if term < p.currentTerm || p.logPriorCheck(lastLogIndex, lastLogTerm) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	// check receiver's qualification: Have not voted before, or voted to you before
	if !(p.votedFor == nil || p.votedFor == &candidateID) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	// vote for this candidate
	// what else should this receiver do?
	p.votedFor = &candidateID
	// change its current term to candidate's term
	p.currentTerm = term
	return requestVoteRes{Term: p.currentTerm, VoteGranted: true}
}

func (p *Peer) logPriorCheck(lastLogIndex int, lastLogTerm int) bool {
	myLastLogIndex := len(p.log) - 1
	myLastLogTerm := p.log[myLastLogIndex].term
	if myLastLogTerm < lastLogTerm {
		return false
	}
	return myLastLogTerm > lastLogTerm ||
		(myLastLogTerm == lastLogTerm && myLastLogIndex > lastLogIndex)
}

// Called by go routine, plz check lock status first
func (p *Peer) handleRequestVoteRespond(term int, success bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if success && term == p.currentTerm {
		p.voteCount += 1

		totalPeers := len(p.rpcPeersIds)
		// received majority votes, become leader
		if p.voteCount > totalPeers/2 && p.state == Candidate {
			p.changeState(Leader)
		}
	} else {
		if term > p.currentTerm {
			p.currentTerm = term
			p.changeState(Follower)
		}
	}
}
