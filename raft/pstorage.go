package raft

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

type persistentData struct {
	// TODO: check this
	Log         []LogEntry
	CommitIndex int

	// for validation
	NodeID    rpccore.NodeID
	NumOfNode int
}

func (p *Peer) loadFromPersistentStorage() error {
	var data persistentData
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
		p.log = data.Log
		p.commitIndex = data.CommitIndex
		// update state machine
		for i := 0; i <= p.commitIndex; i++ {
			err := p.stateMachine.ApplyAction(p.log[p.toLogIndex(i)])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Peer) saveToPersistentStorage() error {
	var data persistentData
	data.Log = p.log
	data.CommitIndex = p.commitIndex
	data.NodeID = p.node.NodeID()
	data.NumOfNode = len(p.rpcPeersIds)
	return p.persistentStorage.Save(data)
}
