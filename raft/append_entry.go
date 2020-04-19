package raft

import "github.com/PwzXxm/raft-lite/utils"

func (p *Peer) handleAppendEntries(req appendEntriesReq) *appendEntriesRes {
	// consistency check
	consistent := p.consitencyCheck(req)
	if !consistent {
		return &appendEntriesRes{Term: p.currentTerm, Success: false}
	}
	// TODO: check this.
	p.heardFromLeader = true
	prevLogIndex := req.PrevLogIndex
	newLogIndex := 0
	// find the index that the peer is consistent with the new entries
	for len(p.log) > (prevLogIndex+newLogIndex+1) &&
		p.log[prevLogIndex+newLogIndex+1].term == req.Entries[newLogIndex].term {
		newLogIndex++
	}
	p.log = append(p.log[0:prevLogIndex+newLogIndex+1], req.Entries[newLogIndex:]...)
	p.logger.Infof("Delete and append new logs from index %v", prevLogIndex+newLogIndex+1)
	// consistency check ensure that req.Term >= p.currentTerm
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
