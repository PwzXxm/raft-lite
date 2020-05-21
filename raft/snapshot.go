package raft

import (
	"github.com/PwzXxm/raft-lite/sm"
)

func (p *Peer) makeSnapshot(lastIncludedIndex int) (*Snapshot, error) {
	stateMachineSnapshot, err := p.stateMachine.TakeSnapshot()
	// fmt.Printf("StateMachine %v, Make SM snapshot: %v, log: %v\n", p.stateMachine, stateMachineSnapshot, p.log)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		LastIncludedIndex:    lastIncludedIndex,
		LastIncludedTerm:     p.getLogTermByIndex(lastIncludedIndex),
		StateMachineSnapshot: stateMachineSnapshot,
	}, nil
}

func (p *Peer) saveToSnapshot(lastIncludedIndex int) error {
	newSnapshot, err := p.makeSnapshot(lastIncludedIndex)
	if err != nil {
		return err
	}
	p.log = p.log[p.toLogIndex(lastIncludedIndex+1):]
	p.snapshot = newSnapshot
	// p.logger.Infof("Success save to snapshot: current log %v, current snapshot %v", p.log, p.snapshot)
	return nil
}

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

	// update CommitIndex, Snapshot
	p.saveToPersistentStorageAndLogError()

	return res
}

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
	smEqual, err := sm.TSMIsSnapshotEqual(s1.StateMachineSnapshot, s2.StateMachineSnapshot)
	if err != nil {
		return false, err
	}
	if smEqual && s1.LastIncludedIndex == s2.LastIncludedIndex && s1.LastIncludedTerm == s2.LastIncludedTerm {
		return true, nil
	}
	return false, nil
}
