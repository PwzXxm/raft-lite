package raft

import (
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
)

const leaderRequestTimeout = 2 * time.Second

func (p *Peer) handleAppendEntries(req appendEntriesReq) *appendEntriesRes {
	// consistency check
	consistent := p.consitencyCheck(req)
	if !consistent {
		return &appendEntriesRes{Term: p.currentTerm, Success: false}
	}

	// consistency check ensure that req.Term >= p.currentTerm
	if len(req.Entries) != 0 {
		// TODO: check this.
		prevLogIndex := req.PrevLogIndex
		newLogIndex := 0
		// find the index that the peer is consistent with the new entries
		for p.logLen() > (prevLogIndex+newLogIndex+1) &&
			p.log[p.toLogIndex(prevLogIndex+newLogIndex+1)].Term == req.Entries[newLogIndex].Term {
			newLogIndex++
		}
		p.log = append(p.log[0:p.toLogIndex(prevLogIndex+newLogIndex+1)], req.Entries[newLogIndex:]...)
		if len(req.Entries) > newLogIndex {
			p.logger.Infof("Delete and append new logs from index %v", prevLogIndex+newLogIndex+1)
		}
	}

	p.updateCommitIndex(req.LeaderCommit)
	return &appendEntriesRes{Term: p.currentTerm, Success: true}
}

// check consitency, update state and term if necessary
func (p *Peer) consitencyCheck(req appendEntriesReq) bool {
	if req.Term < p.currentTerm {
		return false
	} else {
		p.heardFromLeader = true
		p.updateTerm(req.Term)
		p.changeState(Follower)
	}
	if p.logLen() <= req.PrevLogIndex {
		return false
	}
	var myPrevLogTerm int
	if p.toLogIndex(req.PrevLogIndex+1) == 0 && p.snapshot != nil {
		myPrevLogTerm = p.snapshot.LastIncludedTerm
	} else {
		myPrevLogTerm = p.log[p.toLogIndex(req.PrevLogIndex)].Term
	}
	if myPrevLogTerm != req.PrevLogTerm {
		return false
	}
	return true
}

//iteratively call appendEntry RPC until the follower is up to date with leader.
func (p *Peer) callAppendEntryRPC(target rpccore.NodeID) {
	p.mutex.Lock()
	if p.appendingEntries[target] {
		p.mutex.Unlock()
		return
	}
	p.appendingEntries[target] = true
	leaderID := p.node.NodeID()
	p.mutex.Unlock()
	defer func() {
		p.mutex.Lock()
		p.appendingEntries[target] = false
		p.mutex.Unlock()
	}()
	isFirstTime := true
	// call append entry RPC
	for {
		// TODO: add other conditions that should stop sending request
		p.mutex.Lock()
		if p.state != Leader || p.shutdown {
			p.mutex.Unlock()
			return
		}
		nextIndex := p.nextIndex[target]
		// do install snapshot
		if p.toLogIndex(nextIndex) < 0 && p.snapshot != nil {
			p.mutex.Unlock()
			req := installSnapshotReq{Term: p.currentTerm, LeaderID: leaderID, LastIncludedIndex: p.snapshot.LastIncludedIndex,
				LastIncludedTerm: p.snapshot.LastIncludedTerm, Snapshot: p.snapshot}
			res := p.installSnapshot(target, req)
			if res == nil {
				continue
			} else {
				p.mutex.Lock()
				p.handleInstallSnapshotRes(res)
				p.nextIndex[target] = p.snapshot.LastIncludedIndex + 1
				p.mutex.Unlock()
			}
			// do append entries
		} else {
			currentTerm := p.currentTerm
			if nextIndex <= 0 {
				p.logger.Warn("nextIndex out of range")
			}
			var prevLogTerm int
			if p.toLogIndex(nextIndex) == 0 && p.snapshot != nil {
				prevLogTerm = p.snapshot.LastIncludedTerm
			} else {
				prevLogTerm = p.log[p.toLogIndex(nextIndex-1)].Term
			}
			leaderCommit := p.commitIndex
			entries := p.log[p.toLogIndex(nextIndex):]
			p.mutex.Unlock()
			// if no more entries need to be updated, return
			if len(entries) == 0 && !isFirstTime {
				return
			}
			isFirstTime = false
			req := appendEntriesReq{Term: currentTerm, LeaderID: leaderID, PrevLogIndex: nextIndex - 1,
				PrevLogTerm: prevLogTerm, LeaderCommit: leaderCommit, Entries: entries}
			res := p.appendEntries(target, req)
			if res == nil {
				// retry call appendEntries rpc if response is nil
				continue
			} else if !res.Success {
				// update nextIndex for target node
				p.mutex.Lock()
				if res.Term > currentTerm {
					p.updateTerm(res.Term)
					p.changeState(Follower)
				} else {
					p.nextIndex[target]--
				}
				p.mutex.Unlock()
			} else {
				// if success, update nextIndex for the node
				p.mutex.Lock()
				commitIndex := p.commitIndex
				p.nextIndex[target] = nextIndex + len(entries)
				// send signal to the channels for index greater than commit index
				for i := commitIndex + 1; i < p.nextIndex[target]; i++ {
					c, ok := p.logIndexMajorityCheckChannel[i]
					if ok {
						c <- target
					}
				}
				p.mutex.Unlock()
			}
		}
	}
}

// this is a blocking function
func (p *Peer) onReceiveClientRequest(cmd interface{}) bool {
	p.mutex.Lock()
	if p.state != Leader {
		p.mutex.Unlock()
		return false
	}
	newlog := LogEntry{Term: p.currentTerm, Cmd: cmd}
	p.log = append(p.log, newlog)
	newLogIndex := p.logLen() - 1
	totalPeers := p.getTotalPeers()
	majorityCheckChannel := make(chan rpccore.NodeID, totalPeers)
	majorityCheckChannel <- p.node.NodeID()
	p.logIndexMajorityCheckChannel[newLogIndex] = majorityCheckChannel
	// trigger timeout to initialize call appendEntryRPC
	p.triggerTimeout()
	p.mutex.Unlock()
	count := 0
	for range majorityCheckChannel {
		count++
		if 2*count > totalPeers {
			p.mutex.Lock()
			// update commitIndex, use max in case commitIndex is already updated by other client request
			p.updateCommitIndex(newLogIndex)
			// delete channel for the committed index
			delete(p.logIndexMajorityCheckChannel, newLogIndex)
			close(majorityCheckChannel)
			p.mutex.Unlock()
			p.respondClient(newLogIndex)
			break
		}
	}
	return true
}

// TODO: maybe respond to client and commit change to the state machine later
func (p *Peer) respondClient(logIndex int) {
	p.logger.Infof("New log has been commited with log index %v", logIndex)
}

func (p *Peer) HandleClientRequest(cmd interface{}) bool {
	// double check leader status
	p.mutex.Lock()
	if p.state != Leader {
		p.mutex.Unlock()
		return false
	}

	p.mutex.Unlock()

	p.logger.Infof("Received new request to append %v", cmd)

	// use timeout to chec
	c := make(chan bool)
	go func() {
		c <- p.onReceiveClientRequest(cmd)
	}()

	select {
	case done := <-c:
		return done
	case <-time.After(leaderRequestTimeout):
		return false
	}
}
