package raft

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/utils"
)

func (p *Peer) handleAppendEntries(req appendEntriesReq) *appendEntriesRes {
	// consistency check
	consistent := p.consitencyCheck(req)
	if !consistent {
		return &appendEntriesRes{Term: p.currentTerm, Success: false}
	}
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

//iteratively call appendEntry RPC until getting success result
func (p *Peer) callAppendEntryRPC(target rpccore.NodeID) {
	nextIndex := p.nextIndex[target]
	prevLogTerm := p.log[nextIndex-1].term
	// call append entry RPC
	req := appendEntriesReq{Term: p.currentTerm, LeaderID: p.node.NodeID(), PrevLogIndex: nextIndex - 1, PrevLogTerm: prevLogTerm, LeaderCommit: p.commitIndex, Entries: p.log[nextIndex:]}
	res := p.appendEntries(target, req)
	for res.Success == false && p.state == Leader {
		// update nextIndex for target node
		p.mutex.Lock()
		p.nextIndex[target]--
		p.mutex.Unlock()
		// update and resend appendEntry request
		nextIndex = p.nextIndex[target]
		req.PrevLogIndex = nextIndex - 1
		req.PrevLogTerm = p.log[nextIndex-1].term
		req.Entries = p.log[nextIndex:]
		res = p.appendEntries(target, req)
	}
	//if success, update nextIndex for the node
	if res.Success == true && p.state == Leader {
		p.mutex.Lock()
		p.nextIndex[target] = p.nextIndex[target] + len(req.Entries)
		p.mutex.Unlock()
	}
}
