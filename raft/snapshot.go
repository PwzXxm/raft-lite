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
	"github.com/PwzXxm/raft-lite/sm"
)

// makeSnapshot returns the latest snapshot based on lastIncludedIndex
func (p *Peer) makeSnapshot(lastIncludedIndex int) (*Snapshot, error) {
	stateMachineSnapshot, err := p.stateMachine.TakeSnapshot()
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		LastIncludedIndex:    lastIncludedIndex,
		LastIncludedTerm:     p.getLogTermByIndex(lastIncludedIndex),
		StateMachineSnapshot: stateMachineSnapshot,
	}, nil
}

// saveToSnapshot saves the laest snapshot into peer
func (p *Peer) saveToSnapshot(lastIncludedIndex int) error {
	newSnapshot, err := p.makeSnapshot(lastIncludedIndex)
	if err != nil {
		return err
	}
	p.log = p.log[p.toLogIndex(lastIncludedIndex+1):]
	p.snapshot = newSnapshot
	return nil
}

// handleInstallSnapshot takes an installSnapshotReq struct as an argument,
// and returns an installSnapshotRes struct with peer's current term
func (p *Peer) handleInstallSnapshot(req installSnapshotReq) installSnapshotRes {
	// return current term anyway
	res := installSnapshotRes{Term: p.currentTerm}
	if req.Term < p.currentTerm {
		return res
	}
	// retain following logs if mine is longer
	myLastRecoredIndex := p.logLen() - 1
	if req.LastIncludedIndex < myLastRecoredIndex && p.getLogTermByIndex(req.LastIncludedIndex) == req.LastIncludedTerm {
		p.log = p.log[p.toLogIndex(req.LastIncludedIndex+1):]
	} else {
		p.log = []LogEntry{}
	}

	p.commitIndex = req.LastIncludedIndex
	p.snapshot = req.Snapshot
	p.heardFromLeader = true
	p.stateMachine.ResetWithSnapshot(p.snapshot.StateMachineSnapshot)

	// update CommitIndex, Log, Snapshot
	p.saveToPersistentStorageAndLogError()

	return res
}

// handleInstallSnapshotRes handles the response
func (p *Peer) handleInstallSnapshotRes(res *installSnapshotRes) {
	// update leader's term if response includes a higher term
	if res.Term > p.currentTerm {
		p.updateTerm(res.Term)
		p.changeState(Follower)
		// update CurrentTerm, VotedFor
		p.saveToPersistentStorageAndLogError()
	}
}

// SnapshotEqual returns a bool value whether two snapshots are equal and error value if occurs
func SnapshotEqual(s1 *Snapshot, s2 *Snapshot) (bool, error) {
	if s1 == nil && s2 == nil {
		return true, nil
	}
	if s1 != nil && s2 != nil {
		smEqual, err := sm.TSMIsSnapshotEqual(s1.StateMachineSnapshot, s2.StateMachineSnapshot)
		if err != nil {
			return false, err
		}
		if smEqual && s1.LastIncludedIndex == s2.LastIncludedIndex && s1.LastIncludedTerm == s2.LastIncludedTerm {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}
