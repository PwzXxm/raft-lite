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

import "github.com/PwzXxm/raft-lite/rpccore"

// handleRequestVote takes a requestVoteRes struct and returns a requestVoteRes
// checks the required qualifications and responds with a term and bool value
func (p *Peer) handleRequestVote(req requestVoteReq) requestVoteRes {
	if p.updateTerm(req.Term) {
		p.changeState(Follower)
		// p.resetTimeout()
	}
	// check candidate's qualification:
	//  1. Deny if its term is samller than mine
	//  2. or my log is more up to date
	if req.Term < p.currentTerm || p.logPriorCheck(req.LastLogIndex, req.LastLogTerm) {
		return requestVoteRes{Term: p.currentTerm, VoteGranted: false}
	}
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

// startElection begins the leader election
//  1. increment current term
//  2. vote for self
//  3. send request vote RPCs to all other servers
//  4. reset election timeout (trigger in changeState)
func (p *Peer) startElection() {
	p.logger.Info("Start election.")
	p.voteCount = 1
	voteID := p.node.NodeID()
	_ = p.updateTerm(p.currentTerm + 1)
	p.votedFor = &voteID

	// update CurrentTerm, VoteCount, VotedFor
	p.saveToPersistentStorageAndLogError()

	term := p.currentTerm
	req := requestVoteReq{Term: p.currentTerm, CandidateID: p.node.NodeID(), LastLogIndex: p.logLen() - 1, LastLogTerm: p.getLogTermByIndex(p.logLen() - 1)}
	for _, peerID := range p.rpcPeersIds {
		go func(peerID rpccore.NodeID, term int) {
			for {
				// return if this election is invalid
				//  1. peer is not candidate anymore
				//  2. next round of election starts
				p.mutex.Lock()
				if p.state != Candidate || p.currentTerm != term {
					p.mutex.Unlock()
					return
				}
				p.mutex.Unlock()

				res := p.requestVote(peerID, req)
				if res != nil {
					p.mutex.Lock()
					p.handleRequestVoteRespond(*res, peerID)
					p.mutex.Unlock()
					return
				}
			}
		}(peerID, term)
	}
}

// handleRequestVoteRespond handles the response, candidate only
// called by go routine, plz check lock status first
func (p *Peer) handleRequestVoteRespond(res requestVoteRes, id rpccore.NodeID) {
	// response contains term > currentTerm, convert to Follower
	if p.updateTerm(res.Term) {
		p.changeState(Follower)
		return
	}

	// outdated response case
	if res.Term < p.currentTerm || p.state != Candidate {
		return
	}

	if res.VoteGranted {
		p.voteCount++
		p.logger.Debugf("Received vote granted from peer %v for term %v, current count %v", id, p.currentTerm, p.voteCount)
		// Note that p.rpcPeersIds dose not include itself
		totalPeers := p.getTotalPeers()
		// received majority votes, become leader
		if p.voteCount > totalPeers/2 {
			p.logger.Info("Change to leader.")
			p.changeState(Leader)
			go p.onReceiveClientRequest(nil)
		}
	}

	// save to persistent storage
	//  1. update VoteCount if vote is granted
	//  2. update CurrentTerm, VotedFor if steps down
	p.saveToPersistentStorageAndLogError()
}
