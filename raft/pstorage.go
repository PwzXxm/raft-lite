package raft

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

type PersistentData struct {
	CurrentTerm int
	CommitIndex int
	VoteCount   int
	VotedFor    *rpccore.NodeID
	Log         []LogEntry
	Snapshot    *Snapshot

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
		p.commitIndex = data.CommitIndex
		p.voteCount = data.VoteCount
		p.votedFor = data.VotedFor
		p.log = data.Log
		p.snapshot = data.Snapshot
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
	data.CommitIndex = p.commitIndex
	data.VoteCount = p.voteCount
	data.VotedFor = p.votedFor
	data.Log = p.log
	data.Snapshot = p.snapshot

	data.NodeID = p.node.NodeID()
	data.NumOfNode = len(p.rpcPeersIds)
	return p.persistentStorage.Save(data)
}

func (p *Peer) saveToPersistentStorageAndLogError() error {
	err := p.saveToPersistentStorage()
	if err != nil {
		p.logger.Errorf("Unable to save state: %+v.", err)
	}
	return err
}
