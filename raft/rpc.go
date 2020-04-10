package raft

func (p *Peer) requestVote(term, candidateId, lastLogIndex, lastLogTerm int) (int, bool) {
    return 0, false
}

func (p *Peer) appendEntries(term, leaderId, prevLogIndex, prevLogTerm, leaderCommit int, entries []LogEntry) (int, bool) {
    return 0, false
}
