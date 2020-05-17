package raft

import (
	"fmt"
	"sync"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

type ConfigurationState int

const (
	COld ConfigurationState = iota
	COldNew
	CNew
)

type Configuration struct {
	state     ConfigurationState
	mutex     sync.RWMutex
	cOldPeers map[rpccore.NodeID]*Peer
	cNewPeers map[rpccore.NodeID]*Peer
}

// Notes
// two-member cluster: one dies before the other commits

func (c *Configuration) AddPeers(n int, raftPeers map[rpccore.NodeID]*Peer) error {
	fmt.Println("Start adding peers.")

	if n <= 0 {
		err := errors.Errorf("The number of peers should be positive, but got %v", n)
		return err
	}

	return nil
}

func (c *Configuration) RemovePeers() {
	fmt.Println("Start removing peers.")
}

func (c *Configuration) handleConfigurationVoteCheck(res map[rpccore.NodeID]ConfigurationRes) (bool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Check COld votes
	cOldCount := 0
	cOldTotal := len(c.cOldPeers)
	for id := range c.cOldPeers {
		if res[id].Success {
			cOldCount++
		}
		if cOldCount > cOldTotal/2 {
			break
		}
	}

	// no majority success in Cold
	if cOldCount < cOldTotal/2 {
		return false, nil
	}

	if c.state == COld {
		return true, nil
	}

	// Check CNew votes
	if len(c.cNewPeers) <= 0 {
		return false, errors.New("No new peers existed")
	}

	cNewCount := 0
	cNewTotal := len(c.cNewPeers)
	for id := range c.cNewPeers {
		if res[id].Success {
			cNewCount++
		}
		if cNewCount > cNewTotal/2 {
			break
		}
	}

	// no majority success in Cnew
	if cNewCount < cNewTotal/2 {
		return false, nil
	}
	return true, nil
}

// update the state from Cold,new to Cold or Cnew
func (c *Configuration) updateState(state ConfigurationState) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.state != COldNew {
		return errors.New("Configuration state should be Cold,new.")
	}

	switch state {
	case COld:
		c.cNewPeers = nil
		c.state = COld
	case CNew:
		if len(c.cNewPeers) <= 0 {
			return errors.New("No new peers existed")
		}
		c.cOldPeers = c.cNewPeers
		c.cNewPeers = nil
		c.state = COld
	}

	return nil
}
