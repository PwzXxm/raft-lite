package raft

func (p *Peer) makeSnapshot(lastIncludedIndex int) *Snapshot {
	return &Snapshot{
		LastIncludedIndex: lastIncludedIndex,
		LastIncludedTerm:  p.log[p.toLogIndex(lastIncludedIndex)].Term,
		// TODO: write state machine state into ss
		// TODO: write membership config into ss
	}
}

func (p *Peer) saveToSnapshot() {
	snapshot := p.makeSnapshot(p.commitIndex)
	p.snapshot = snapshot
	p.log = p.log[p.toLogIndex(p.commitIndex+1):]
}

func (p *Peer) handleInstallSnapshot(req installSnapshotReq) installSnapshotRes {
	// return current term anyway
	res := installSnapshotRes{Term: p.currentTerm}
	if req.Term < p.currentTerm {
		return res
	}
	// retain following logs if mine is longer
	myLastRecoredIndex := p.logLen() - 1
	if req.LastIncludedIndex < myLastRecoredIndex && p.log[p.toLogIndex(req.LastIncludedIndex)].Term == req.LastIncludedTerm {
		p.log = p.log[p.toLogIndex(req.LastIncludedIndex+1):]
	} else {
		p.log = []LogEntry{}
	}

	p.snapshot = req.Snapshot
	// TODO: write state machine state from ss
	// TODO: write membership config from ss
	return res
}

func (p *Peer) handleInstallSnapshotRes(res *installSnapshotRes) {
	// update leader's term if res includs a higher term?
	if res.Term > p.currentTerm {
		p.currentTerm = res.Term
	}
}
