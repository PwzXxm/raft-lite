package raft

import (
	"fmt"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/utils"
)

const clientRequestTimeout = 5 * time.Second

func (p *Peer) handleAppendEntries(req appendEntriesReq) *appendEntriesRes {
	// consistency check
	p.logger.Infof("Received ae from %v", req.LeaderID)
	consistent := p.consitencyCheck(req)
	if !consistent {
		p.logger.Info("failed consistent check")
		return &appendEntriesRes{Term: p.currentTerm, Success: false}
	}
	// if the request is heartbeat, return true
	if len(req.Entries) == 0 {
		return &appendEntriesRes{Term: p.currentTerm, Success: true}
	}
	// TODO: check this.
	prevLogIndex := req.PrevLogIndex
	newLogIndex := 0
	// find the index that the peer is consistent with the new entries
	for len(p.log) > (prevLogIndex+newLogIndex+1) &&
		p.log[prevLogIndex+newLogIndex+1].Term == req.Entries[newLogIndex].Term {
		newLogIndex++
	}
	p.log = append(p.log[0:prevLogIndex+newLogIndex+1], req.Entries[newLogIndex:]...)
	if len(req.Entries) > newLogIndex {
		p.logger.Infof("Delete and append new logs from index %v", prevLogIndex+newLogIndex+1)
	}
	// consistency check ensure that req.Term >= p.currentTerm
	p.updateTerm(req.Term)
	if p.state != Follower {
		p.changeState(Follower)
	}
	if req.LeaderCommit > p.commitIndex {
		p.commitIndex = utils.Min(req.LeaderCommit, len(p.log)-1)
	}
	return &appendEntriesRes{Term: p.currentTerm, Success: true}
}

// check consitency, update state and term if necessary
func (p *Peer) consitencyCheck(req appendEntriesReq) bool {
	if req.Term < p.currentTerm {
		return false
	} else {
		p.heardFromLeader = true
		p.updateTerm(req.Term)
		if p.state != Follower {
			p.changeState(Follower)
		}
	}
	if len(p.log) <= req.PrevLogIndex || p.log[req.PrevLogIndex].Term != req.PrevLogTerm {
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
		if p.state != Leader {
			p.mutex.Unlock()
			return
		}
		nextIndex := p.nextIndex[target]
		currentTerm := p.currentTerm
		if nextIndex <= 0 {
			p.logger.Warn("nextIndex out of range")
		}
		prevLogTerm := p.log[nextIndex-1].Term
		leaderCommit := p.commitIndex
		entries := p.log[nextIndex:]
		p.mutex.Unlock()
		// if no more entries need to be updated, return
		if len(entries) == 0 && !isFirstTime {
			return
		}
		isFirstTime = false
		req := appendEntriesReq{Term: currentTerm, LeaderID: leaderID, PrevLogIndex: nextIndex - 1,
			PrevLogTerm: prevLogTerm, LeaderCommit: leaderCommit, Entries: entries}
		p.logger.Infof("sending to %v, leaderCommit: %v", target, leaderCommit)
		res := p.appendEntries(target, req)
		p.logger.Infof("sent to %v", target)
		if res == nil {
			// retry call appendEntries rpc if response is nil
			continue
		} else if !res.Success {
			// update nextIndex for target node
			p.mutex.Lock()
			if res.Term > currentTerm {
				p.updateTerm(res.Term)
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
				p.logIndexMajorityCheckChannel[i] <- target
			}
			p.mutex.Unlock()
		}
	}
}

// this is a blocking function
func (p *Peer) onReceiveClientRequest(cmd interface{}) {
	p.logger.Info("received client request: ", cmd)
	p.mutex.Lock()
	newlog := LogEntry{Term: p.currentTerm, Cmd: cmd}
	p.log = append(p.log, newlog)
	newLogIndex := len(p.log) - 1
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
	p.logger.Infof("New log %v has been commited with log index %v", p.log[logIndex], logIndex)
}

func (p *Peer) HandleClientRequest(cmd interface{}) bool {
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
		p.onReceiveClientRequest(cmd)
		c <- true
	}()

	fmt.Println("timeout?????")
	select {
	case done := <-c:
		return done
	case <-time.After(clientRequestTimeout):
		fmt.Println("timeout-------------------")
		return false
	}
}
