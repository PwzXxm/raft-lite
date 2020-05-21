package raft

// handleRequestVote takes a requestVoteRes struct and returns a requestVoteRes
// checks the required qualifications and responds with a term and bool value
func (p *Peer) handleRequestVote(req requestVoteReq) requestVoteRes {
	// check candidate's qualification:
	//  1. Deny if its term is samller than mine
	//  2. or my log is more up to date
	if req.Term < p.currentTerm || p.logPriorCheck(req.LastLogIndex, req.LastLogTerm) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	p.updateTerm(req.Term)
	// check receiver's qualification:
	//  1. Have not voted before, or voted to you before
	if !(p.votedFor == nil || p.votedFor == &req.CandidateID) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
	p.votedFor = &req.CandidateID
	p.resetTimeout()

	// update CurrentTerm, VotedFor
	p.saveToPersistentStorageAndLogError()

	return requestVoteRes{Term: p.currentTerm, VoteGranted: true}
}

// logPriorCheck takes a log index and term, returns the check result
// it is used for determining two logs which is more up-to-date
func (p *Peer) logPriorCheck(lastLogIndex int, lastLogTerm int) bool {
	myLastLogIndex := p.logLen() - 1
	myLastLogTerm := p.getLogTermByIndex(myLastLogIndex)
	// return logic: 1 || 2
	//  1. Two logs have last entries with different terms,
	//     log with later term is more up-to-date
	//  2. Logs with the same term, then longer log is more up-to-date
	return myLastLogTerm > lastLogTerm ||
		(myLastLogTerm == lastLogTerm && myLastLogIndex > lastLogIndex)
}

// Called by go routine, plz check lock status first
func (p *Peer) handleRequestVoteRespond(res requestVoteRes) {
	if res.Term < p.currentTerm {
		return
	}

	if res.VoteGranted {
		p.voteCount++
		totalPeers := p.getTotalPeers()
		// received majority votes (more than half), become leader
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
	// save to persistent storage
	//  1. update VoteCount if vote is granted
	//  2. update CurrentTerm, VotedFor if steps down
	p.saveToPersistentStorageAndLogError()
}
