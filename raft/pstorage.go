package raft

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

type persistentData struct {
	// TODO: check this
	Log         []LogEntry
	CommitIndex int
	NodeID      rpccore.NodeID
	NumOfNode   int
}

func (p *Peer) LoadFromPersistentStorage() error {
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
			err := p.stateMachine.ApplyAction(p.log[i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Peer) SaveToPersistentStorage() {

}
