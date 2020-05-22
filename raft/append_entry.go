package raft

import (
	"fmt"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/utils"
)

var Foo func()

func (p *Peer) handleAppendEntries(req appendEntriesReq) *appendEntriesRes {
	// consistency check
	consistent := p.consitencyCheck(req)
	if !consistent {
		return &appendEntriesRes{Term: p.currentTerm, Success: false}
	}

	// consistency check ensure that req.Term >= p.currentTerm
	if len(req.Entries) != 0 {
		prevLogIndex := req.PrevLogIndex
		newLogIndex := 0
		// find the index that the peer is consistent with the new entries
		for p.logLen() > (prevLogIndex+newLogIndex+1) &&
			p.getLogTermByIndex(prevLogIndex+newLogIndex+1) == req.Entries[newLogIndex].Term {
			newLogIndex++
		}
		if prevLogIndex + newLogIndex < p.commitIndex {
			Foo()
			panic("overriding committed log")
		}
		p.log = append(p.log[0:p.toLogIndex(prevLogIndex+newLogIndex+1)], req.Entries[newLogIndex:]...)
		if len(req.Entries) > newLogIndex {
			p.logger.Infof("Delete and append new logs from index %v", prevLogIndex+newLogIndex+1)
		}
	}

	p.updateCommitIndex(req.LeaderCommit)

	// update CommitIndex, Term, Log
	p.saveToPersistentStorageAndLogError()

	return &appendEntriesRes{Term: p.currentTerm, Success: true}
}

// check consitency, update state and term if necessary
func (p *Peer) consitencyCheck(req appendEntriesReq) bool {
	if req.Term < p.currentTerm {
		return false
	}

	p.heardFromLeader = true
	p.updateTerm(req.Term)
	p.changeState(Follower)
	// update CurrentTerm, VotedFor
	p.saveToPersistentStorageAndLogError()

	p.leaderID = &req.LeaderID

	if p.logLen() <= req.PrevLogIndex {
		return false
	}
	if p.toLogIndex(req.PrevLogIndex) < -1 {
		fmt.Errorf("AAAAAA")
		panic("")
	}
	myPrevLogTerm := p.getLogTermByIndex(req.PrevLogIndex)
	if myPrevLogTerm != req.PrevLogTerm {
		if req.PrevLogIndex <= p.commitIndex {
			p.logger.Errorf("invalid case! prevLogIndex:%v, logreq:%v, loglocal:%v", req.PrevLogIndex, req.Entries, p.log)
			panic("")
		}
		return false
	}
	return true
}

func (p *Peer) triggerLeaderHeartbeat() {
	for peerID, appendingEntry := range p.appendingEntries {
		if !appendingEntry {
			go p.callAppendEntryRPC(peerID)
		}
	}
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
		p.mutex.Lock()
		if p.state != Leader || p.shutdown {
			p.mutex.Unlock()
			return
		}
		nextIndex := p.nextIndex[target]
		if p.toLogIndex(nextIndex) < 0 && p.snapshot != nil {
			// do install snapshot
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
		} else {
			// do append entries
			currentTerm := p.currentTerm
			if nextIndex <= 0 {
				p.logger.Error("nextIndex out of range")
				p.mutex.Unlock()
				return
			}
			prevLogTerm := p.getLogTermByIndex(nextIndex - 1)
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
			} else {
				p.mutex.Lock()
				p.updateLastHeard(target)
				if !res.Success {
					// update nextIndex for target node
					if res.Term > currentTerm {
						p.updateTerm(res.Term)
						p.changeState(Follower)
					} else {
						p.nextIndex[target]--
					}
					p.mutex.Unlock()
				} else {
					// if success, update nextIndex for the node
					commitIndex := p.commitIndex
					p.nextIndex[target] = nextIndex + len(entries)
					// send signal to the channels for index greater than commit index
					for i := utils.Max(commitIndex, p.matchIndex[target]) + 1; i < p.nextIndex[target]; i++ {
						c, ok := p.logIndexMajorityCheckChannel[i]
						if ok {
							c <- target
						}
					}
					p.matchIndex[target] = p.nextIndex[target] - 1
					p.mutex.Unlock()
				}
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
	// update Log
	p.saveToPersistentStorageAndLogError()
	newLogIndex := p.logLen() - 1
	totalPeers := p.getTotalPeers()
	majorityCheckChannel := make(chan rpccore.NodeID, totalPeers)
	majorityCheckChannel <- p.node.NodeID()
	p.logIndexMajorityCheckChannel[newLogIndex] = majorityCheckChannel
	// trigger timeout to initialize call appendEntryRPC
	p.triggerTimeout()
	p.mutex.Unlock()
	count := 0
	for nodeID := range majorityCheckChannel {
		count++
		p.logger.Infof("peer %v confirm log with index %v", nodeID, newLogIndex)
		if 2*count > totalPeers {
			p.mutex.Lock()
			// update commitIndex, use max in case commitIndex is already updated by other client request
			p.updateCommitIndex(newLogIndex)
			// update CommitIndex
			p.saveToPersistentStorageAndLogError()
			// delete channel for the committed index
			delete(p.logIndexMajorityCheckChannel, newLogIndex)
			close(majorityCheckChannel)
			p.mutex.Unlock()
			p.logger.Infof("New log %v has been commited with log index %v", newlog, newLogIndex)
			break
		}
	}
	return true
}
