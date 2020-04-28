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

//iteratively call appendEntry RPC until the follower is up to date with leader.
func (p *Peer) callAppendEntryRPC(target rpccore.NodeID) {
	p.mutex.Lock()
	p.appendingEntries[target] = true
	leaderID := p.node.NodeID()
	p.mutex.Unlock()
	defer func() {
		p.mutex.Lock()
		p.appendingEntries[target] = false
		p.mutex.Unlock()
	}()
	// call append entry RPC
	for true {
		// TODO: add other conditions that should stop sending request
		p.mutex.Lock()
		if p.state != Leader {
			p.mutex.Unlock()
			return
		}
		nextIndex := p.nextIndex[target]
		currentTerm := p.currentTerm
		prevLogTerm := p.log[nextIndex-1].term
		leaderCommit := p.commitIndex
		entries := p.log[nextIndex:]
		p.mutex.Unlock()
		// if no more entries need to be updated, return
		if len(entries) == 0 {
			return
		}
		req := appendEntriesReq{Term: currentTerm, LeaderID: leaderID, PrevLogIndex: nextIndex - 1,
			PrevLogTerm: prevLogTerm, LeaderCommit: leaderCommit, Entries: entries}
		res := p.appendEntries(target, req)
		if res == nil {
			// retry call appendEntries rpc if response is nil
			continue
		} else if res.Success == false {
			// update nextIndex for target node
			p.mutex.Lock()
			p.nextIndex[target]--
			p.mutex.Unlock()
		} else {
			// if success, update nextIndex for the node
			p.mutex.Lock()
			commitIndex := p.commitIndex
			p.nextIndex[target] = nextIndex + len(entries)
			// send signal to the channels for index greater than commit index
			for i := commitIndex + 1; i < p.nextIndex[target]; i++ {
				p.logIndexMajorityCheckChannel[i] <- target
			}
			p.mutex.Unlock()
		}
	}
}

// this is a blocking function
func (p *Peer) onReceiveClientRequest(cmd interface{}) {
	p.mutex.Lock()
	newlog := LogEntry{term: p.currentTerm, cmd: cmd}
	p.log = append(p.log, newlog)
	newLogIndex := len(p.log) - 1
	totalPeers := len(p.rpcPeersIds)
	majorityCheckChannel := make(chan rpccore.NodeID, totalPeers)
	p.logIndexMajorityCheckChannel[newLogIndex] = majorityCheckChannel
	p.mutex.Unlock()
	// trigger timeout to initialize call appendEntryRPC
	p.triggerTimeout()
	count := 0
	for range majorityCheckChannel {
		count++
		if 2*count > totalPeers {
			p.mutex.Lock()
			// update commitIndex, use max in case commitIndex is already updated by other client request
			p.commitIndex = utils.Max(p.commitIndex, newLogIndex)
			// delete channel for the committed index
			delete(p.logIndexMajorityCheckChannel, newLogIndex)
			close(majorityCheckChannel)
			p.mutex.Unlock()
			p.respondClient(newLogIndex)
			break
		}
	}
}

// TODO: maybe respond to client and commit change to the state machine later
func (p *Peer) respondClient(logIndex int) {
}

func (p *Peer) HandleClientRequest(cmd interface{}) bool {
	p.mutex.Lock()
	if p.state != Leader {
		p.mutex.Unlock()
		return false
	}

	p.mutex.Unlock()

	// add time out

	p.onReceiveClientRequest(cmd)

	return false
}
