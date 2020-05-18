package raft

func (p *Peer) handleRequestVote(req requestVoteReq) requestVoteRes {
	// check candidate's qualification:
	// 1. Deny if its term is samller than mine
	// 2. or my log is more up to date
	if req.Term < p.currentTerm || p.logPriorCheck(req.LastLogIndex, req.LastLogTerm) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	p.updateTerm(req.Term)
	// check receiver's qualification:
	// 1. Have not voted before, or voted to you before
	if !(p.votedFor == nil || p.votedFor == &req.CandidateID) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	p.votedFor = &req.CandidateID
	// TODO: double check with paper
	p.resetTimeout()
	return requestVoteRes{Term: p.currentTerm, VoteGranted: true}
}

// TODO: [4], [1][2][3], which log is more up-to-date?
// Consider case one node seprate from others and increase its term to high number sololy.
func (p *Peer) logPriorCheck(lastLogIndex int, lastLogTerm int) bool {
	myLastLogIndex := p.logLen() - 1
	myLastLogTerm := p.log[p.toLogIndex(myLastLogIndex)].Term
	return myLastLogTerm > lastLogTerm ||
		(myLastLogTerm == lastLogTerm && myLastLogIndex > lastLogIndex)
}

// Called by go routine, plz check lock status first
func (p *Peer) handleRequestVoteRespond(res requestVoteRes) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if res.Term < p.currentTerm {
		return
	}

	if res.VoteGranted {
		p.voteCount += 1

		// Note that p.rpcPeersIds dose not include itself
		totalPeers := p.getTotalPeers()
		// received majority votes, become leader
		if p.voteCount > totalPeers/2 && p.state == Candidate {
			p.logger.Info("Change to leader.")
			p.changeState(Leader)
		}
	} else {
		if res.Term > p.currentTerm {
			p.updateTerm(res.Term)
			p.changeState(Follower)
		}
	}
}
