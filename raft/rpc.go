package raft

import (
	"bytes"
	"encoding/gob"
	"log"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

const (
	rpcMethodRequestVote   = "rv"
	rpcMethodAppendEntries = "ae"
)

type appendEntriesReq struct {
	Term         int
	LeaderId     int
	PrevLogTerm  int
	LeaderCommit int
	Entries      []LogEntry
}

type appendEntriesRes struct {
	Term    int
	Success bool
}

type requestVoteReq struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type requestVoteRes struct {
	Term        int
	VoteGranted bool
}

func (p *Peer) requestVote(target rpccore.NodeID, arg requestVoteReq) *requestVoteRes {
	var res requestVoteRes
	if p.callRpcAndLogError(target, rpcMethodRequestVote, arg, &res) {
		return &res
	} else {
		return nil
	}
}

func (p *Peer) appendEntries(target rpccore.NodeID, arg appendEntriesReq) *appendEntriesRes {
	var res appendEntriesRes
	if p.callRpcAndLogError(target, rpcMethodAppendEntries, arg, &res) {
		return &res
	} else {
		return nil
	}
}

func (p *Peer) callRpcAndLogError(target rpccore.NodeID, method string, req, res interface{}) bool {
	err := p.callRpc(target, method, req, res)
	if err != nil {
		// TODO: we need better logging to group messages by peers.
		log.Printf("RPC call failed. \n target: %v, method: %v, err: %+v",
			target, method, err)
		return false
	}
	return true
}

func (p *Peer) callRpc(target rpccore.NodeID, method string, req, res interface{}) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}
	resData, err := p.node.SendRawRequest(target, method, buf.Bytes())
	if err != nil {
		// already wrapped
		return err
	}
	err = gob.NewDecoder(bytes.NewReader(resData)).Decode(res)
	if err != nil {
		return errors.WithStack(err)
	}
}
