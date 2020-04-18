package raft

import "github.com/PwzXxm/raft-lite/utils"

func (p *Peer) handleAppendEntries(req appendEntriesReq) *appendEntriesRes {
	// consistency check
	consistent := p.consitencyCheck(req)
	if !consistent {
		return &appendEntriesRes{Term: p.currentTerm, Success: false}
	}
	PrevLogIndex := req.PrevLogIndex
	newLogIndex := 0
	// find the index that the peer is consistent with the new entries
	for len(p.log) > (PrevLogIndex+newLogIndex+1) &&
		p.log[PrevLogIndex+newLogIndex+1].term == req.Entries[newLogIndex].term {
		newLogIndex++
	}
	p.log = append(p.log[0:PrevLogIndex+newLogIndex+1], req.Entries[newLogIndex:]...)
	p.logger.Infof("Delete and append new logs from index %v \n", PrevLogIndex+newLogIndex+1)
	// how to set peer's current term?
	p.currentTerm = req.Term
	if req.LeaderCommit > p.commitIndex {
		p.commitIndex = utils.Min(req.LeaderCommit, req.Entries[len(req.Entries)-1].term)
	}
	return &appendEntriesRes{Term: p.currentTerm, Success: true}
}

func (p *Peer) consitencyCheck(req appendEntriesReq) bool {
	if req.Term < p.currentTerm {
		return false
	}
	if len(p.log) <= req.PrevLogIndex || p.log[req.PrevLogIndex].term != req.PrevLogTerm {
		return false
	}
	return true
}
