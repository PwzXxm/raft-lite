package raft

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

type PersistentData struct {
	// TODO: check this
	CurrentTerm int
	VotedFor    *rpccore.NodeID
	VoteCount   int
	Log         []LogEntry

	CommitIndex int

	Snapshot          *Snapshot
	SnapshotThreshold int

	// for validation
	NodeID    rpccore.NodeID
	NumOfNode int
}

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
		p.votedFor = data.VotedFor
		p.voteCount = data.VoteCount
		p.log = data.Log
		p.commitIndex = data.CommitIndex
		p.snapshot = data.Snapshot
		p.snapshotThreshold = data.SnapshotThreshold
		// update state machine
		// TODO: should add snapshot to persistentData, apply the statemachine in snapshot first,
		// and change the for loop to be the following line
		// for i := p.snapshot.LastIncludedIndex +1; i <= p.commitIndex; i++ {
		for i := 0; i <= p.commitIndex; i++ {
			action := p.log[p.toLogIndex(i)].Cmd
			if action != nil {
				err := p.stateMachine.ApplyAction(action)
				if err != nil {
					p.logger.Error(err)
				}
			}
		}
	}
	return nil
}

func (p *Peer) saveToPersistentStorage() error {
	var data PersistentData
	data.CurrentTerm = p.currentTerm
	data.VotedFor = p.votedFor
	data.VoteCount = p.voteCount
	data.Log = p.log
	data.CommitIndex = p.commitIndex
	data.Snapshot = p.snapshot
	data.SnapshotThreshold = p.snapshotThreshold
	data.NodeID = p.node.NodeID()
	data.NumOfNode = len(p.rpcPeersIds)
	return p.persistentStorage.Save(data)
}
