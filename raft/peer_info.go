package raft

import (
	"fmt"
)

// GetInfo
func (p *Peer) GetInfo() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return fmt.Sprintf("%+v\nsnapshot:%+v\n\n", p, p.snapshot)
}
