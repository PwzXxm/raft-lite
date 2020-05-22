package raft

import "github.com/PwzXxm/raft-lite/rpccore"


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
	p.resetTimeout()

	// update CurrentTerm, VotedFor
	p.saveToPersistentStorageAndLogError()

	return requestVoteRes{Term: p.currentTerm, VoteGranted: true}
}

func (p *Peer) logPriorCheck(lastLogIndex int, lastLogTerm int) bool {
	myLastLogIndex := p.logLen() - 1
	myLastLogTerm := p.getLogTermByIndex(myLastLogIndex)
	return myLastLogTerm > lastLogTerm ||
		(myLastLogTerm == lastLogTerm && myLastLogIndex > lastLogIndex)
}

func (p *Peer) startElection() {
	p.logger.Info("Start election.")
	p.voteCount = 1
	voteID := p.node.NodeID()
	p.updateTerm(p.currentTerm + 1)
	p.votedFor = &voteID

	// update CurrentTerm, VoteCount, VotedFor
	p.saveToPersistentStorageAndLogError()

	term := p.currentTerm
	req := requestVoteReq{Term: p.currentTerm, CandidateID: p.node.NodeID(), LastLogIndex: p.logLen() - 1, LastLogTerm: p.getLogTermByIndex(p.logLen() - 1)}
	for _, peerID := range p.rpcPeersIds {
		go func(peerID rpccore.NodeID, term int) {
			for {
				// return if this election is invalid
				// 1. peer is not candidate anymore
				// 2. next round of election starts
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

// Called by go routine, plz check lock status first
func (p *Peer) handleRequestVoteRespond(res requestVoteRes, id rpccore.NodeID) {
	if res.Term < p.currentTerm || p.state != Candidate {
		return
	}

	if res.VoteGranted {
		p.voteCount++
		p.logger.Infof("Received vote granted from peer %v for term %v, current count %v", id, p.currentTerm, p.voteCount)
		// Note that p.rpcPeersIds dose not include itself
		totalPeers := p.getTotalPeers()
		// received majority votes, become leader
		if p.voteCount > totalPeers/2 {
			p.logger.Info("Change to leader.")
			p.changeState(Leader)
			go p.onReceiveClientRequest(nil)
		}
	} else {
		if res.Term > p.currentTerm {
			p.updateTerm(res.Term)
			p.changeState(Follower)
		}
	}
	// 1. update VoteCount if vote is granted
	// 2. update CurrentTerm, VotedFor if steps down
	p.saveToPersistentStorageAndLogError()
}
