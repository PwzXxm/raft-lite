package raft

func (p *Peer) handleRequestVote(req requestVoteReq) requestVoteRes {
	// check candidate's qualification: 1. Deny if its term is samller than mine 2. or my log is more up to date
	if req.Term < p.currentTerm || p.logPriorCheck(req.LastLogIndex, req.LastLogTerm) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	// check receiver's qualification: Have not voted before, or voted to you before
	if !(p.votedFor == nil || p.votedFor == &req.CandidateID) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	// vote for this candidate
	// what else should this receiver do?
	p.votedFor = &req.CandidateID
	// change its current term to candidate's term
	p.currentTerm = req.Term
	return requestVoteRes{Term: p.currentTerm, VoteGranted: true}
}

func (p *Peer) logPriorCheck(lastLogIndex int, lastLogTerm int) bool {
	myLastLogIndex := len(p.log) - 1
	myLastLogTerm := p.log[myLastLogIndex].term
	return myLastLogTerm > lastLogTerm ||
		(myLastLogTerm == lastLogTerm && myLastLogIndex > lastLogIndex)
}

// Called by go routine, plz check lock status first
func (p *Peer) handleRequestVoteRespond(res requestVoteRes) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if res.VoteGranted && res.Term == p.currentTerm {
		p.voteCount += 1

		totalPeers := len(p.rpcPeersIds)
		// received majority votes, become leader
		if p.voteCount > totalPeers/2 && p.state == Candidate {
			p.logger.Info("Change to leader.")
			p.changeState(Leader)
		}
	} else {
		if res.Term > p.currentTerm {
			p.currentTerm = res.Term
			p.changeState(Follower)
		}
	}
}
