package raft

import (
	"bytes"
	"encoding/gob"
	"fmt"

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
	if p.callRpcAndLogError(target, rpcMethodRequestVote, arg, &res) == nil {
		return &res
	} else {
		return nil
	}
}

func (p *Peer) appendEntries(target rpccore.NodeID, arg appendEntriesReq) *appendEntriesRes {
	var res appendEntriesRes
	if p.callRpcAndLogError(target, rpcMethodAppendEntries, arg, &res) == nil {
		return &res
	} else {
		return nil
	}
}

func (p *Peer) callRpcAndLogError(target rpccore.NodeID, method string, req, res interface{}) error {
	err := p.callRpc(target, method, req, res)
	if err != nil {
		p.logger.Warnf("RPC call failed. \n target: %v, method: %v, err: %+v",
			target, method, err)
	}
	return err
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
	return nil
}

func (p *Peer) handleRpcCallAndLogError(source rpccore.NodeID, method string, data []byte) ([]byte, error) {
	res, err := p.handleRpcCall(source, method, data)
	if err != nil {
		p.logger.Warning("Handle RPC call failed. \n source: %v, method: %v, error: %+v",
			source, method, err)
	}
	return res, err
}

func (p *Peer) handleRpcCall(source rpccore.NodeID, method string, data []byte) ([]byte, error) {
	switch method {
	case rpcMethodRequestVote:
		var req requestVoteReq
		err := gob.NewDecoder(bytes.NewReader(data)).Decode(&req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// TODO: add handler here.
		var res requestVoteRes
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	case rpcMethodAppendEntries:
		var req appendEntriesReq
		err := gob.NewDecoder(bytes.NewReader(data)).Decode(&req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// TODO: add handler here.
		var res appendEntriesRes
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	default:
		err := errors.New(fmt.Sprintf("Unsupport method: %v", method))
		return nil, err
	}
}
