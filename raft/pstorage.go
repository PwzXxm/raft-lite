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

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

// PersistentData takes relevant attributes from Peer
// in order to maintain the data for later recovery
type PersistentData struct {
	CurrentTerm int             // latest term (increases monotonically)
	CommitIndex int             // index of highest log entry known to be committed (increases monotonically)
	VoteCount   int             // count for other peers vote me in this term
	VotedFor    *rpccore.NodeID // candidateID that peer votes in this term, nil if none
	Log         []LogEntry      // log entries
	Snapshot    *Snapshot       // snapshot

	// for validation
	NodeID    rpccore.NodeID // self node ID
	NumOfNode int            // total number of nodes
}

// loadFromPersistentStorage loads any existing data in previous
func (p *Peer) loadFromPersistentStorage() error {
	var data PersistentData
	hasData, err := p.persistentStorage.Load(&data)
	if err != nil {
		return err
	}

	if hasData {
		p.logger.Info("Start loading persistent data.")
		// verify
		if data.NumOfNode != len(p.rpcPeersIds) || data.NodeID != p.node.NodeID() {
			return errors.Errorf("Invalid data: %v", data)
		}
		p.currentTerm = data.CurrentTerm
		p.commitIndex = data.CommitIndex
		p.voteCount = data.VoteCount
		p.votedFor = data.VotedFor
		p.log = data.Log
		p.snapshot = data.Snapshot
		// update state machine
		startIdx := 0
		if p.snapshot != nil {
			p.stateMachine.ResetWithSnapshot(p.snapshot.StateMachineSnapshot)
			startIdx = p.snapshot.LastIncludedIndex + 1
		}
		for i := startIdx; i <= p.commitIndex; i++ {
			action := p.log[p.toLogIndex(i)].Cmd
			if action != nil {
				_ = p.stateMachine.ApplyAction(action)
			}
		}
	}
	return nil
}

// saveToPersistentStorage saves the data into persistent storage
func (p *Peer) saveToPersistentStorage() error {
	var data PersistentData
	data.CurrentTerm = p.currentTerm
	data.CommitIndex = p.commitIndex
	data.VoteCount = p.voteCount
	data.VotedFor = p.votedFor
	data.Log = p.log
	data.Snapshot = p.snapshot

	data.NodeID = p.node.NodeID()
	data.NumOfNode = len(p.rpcPeersIds)
	return p.persistentStorage.Save(data)
}

// saveToPersistentStorageAndLogError returns error value if occurs
func (p *Peer) saveToPersistentStorageAndLogError() {
	err := p.saveToPersistentStorage()
	if err != nil {
		p.logger.Errorf("Unable to save state: %+v.", err)
	}
}
