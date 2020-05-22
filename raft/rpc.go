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
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/PwzXxm/raft-lite/client"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

const (
	rpcMethodRequestVote     = "rv"
	rpcMethodAppendEntries   = "ae"
	rpcMethodInstallSnapshot = "is"
)

type appendEntriesReq struct {
	Term         int            // leader's term
	LeaderID     rpccore.NodeID // leader ID, for redirecting clients
	PrevLogIndex int            // index of log entry immediately preceding new ones
	PrevLogTerm  int            // term of prevLogIndex entry
	LeaderCommit int            // leader’s commitIndex
	Entries      []LogEntry     // log entries
}

type appendEntriesRes struct {
	Term    int  // latest term, for leader updating
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm
}

type requestVoteReq struct {
	Term         int            // candidate's term
	CandidateID  rpccore.NodeID // candidate ID
	LastLogIndex int            // index of candidate’s last log entry
	LastLogTerm  int            // term of candidate’s last log entry
}

type requestVoteRes struct {
	Term        int  // latest term, for candidate updating
	VoteGranted bool // true means candidate received vote
}

type installSnapshotReq struct {
	Term              int            // latest term
	LeaderID          rpccore.NodeID // leader ID
	LastIncludedIndex int            // index of latest snapshot inluded
	LastIncludedTerm  int            // term of latest snapshot included
	Snapshot          *Snapshot      // snapshot
}

type installSnapshotRes struct {
	Term int // latest term
}

// installSnapshot takes target node ID and installSnapshotReq struct as arguments
// and returns an installSnapshotRes struct pointer, nil if not success
func (p *Peer) installSnapshot(target rpccore.NodeID, arg installSnapshotReq) *installSnapshotRes {
	var res installSnapshotRes
	if p.callRPCAndLogError(target, rpcMethodInstallSnapshot, arg, &res) == nil {
		return &res
	}
	return nil
}

// requestVote takes target node ID and requestVoteReq struct as arguments
// and returns a requestVoteRes struct pointer, nil if not success
func (p *Peer) requestVote(target rpccore.NodeID, arg requestVoteReq) *requestVoteRes {
	var res requestVoteRes
	if p.callRPCAndLogError(target, rpcMethodRequestVote, arg, &res) == nil {
		return &res
	}
	return nil
}

// appendEntries takes target node ID and appendEntriesReq struct as arguments
// and returns an appendEntriesRes struct pointer, nil if not success
func (p *Peer) appendEntries(target rpccore.NodeID, arg appendEntriesReq) *appendEntriesRes {
	var res appendEntriesRes
	if p.callRPCAndLogError(target, rpcMethodAppendEntries, arg, &res) == nil {
		return &res
	}
	return nil
}

// callRPCAndLogError takes arguments and traces error value if occurs
func (p *Peer) callRPCAndLogError(target rpccore.NodeID, method string, req, res interface{}) error {
	err := p.callRPC(target, method, req, res)
	if err != nil {
		p.logger.Tracef("RPC call failed. \n target: %v, method: %v, err: %v",
			target, method, err)
	}
	return err
}

// callRPC takes arguments and returns error value if occurs
func (p *Peer) callRPC(target rpccore.NodeID, method string, req, res interface{}) error {
	var buf bytes.Buffer
	// encode request data
	err := gob.NewEncoder(&buf).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}
	// send raw request
	resData, err := p.node.SendRawRequest(target, method, buf.Bytes())
	if err != nil {
		// already wrapped
		return err
	}
	// decode response data
	err = gob.NewDecoder(bytes.NewReader(resData)).Decode(res)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// handleRPCCallAndLogError takes arguments and returns response data and error value if occurs
func (p *Peer) handleRPCCallAndLogError(source rpccore.NodeID, method string, data []byte) ([]byte, error) {
	p.mutex.Lock()
	if p.shutdown {
		p.mutex.Unlock()
		// reduce the number of logs
		time.Sleep(1 * time.Second)
		return nil, errors.New("peer is not running")
	}
	p.mutex.Unlock()
	res, err := p.handleRPCCall(source, method, data)
	if err != nil {
		p.logger.Debugf("Handle RPC call failed. \n source: %v, method: %v, error: %v",
			source, method, err)
	}
	return res, err
}

// handleRPCCall takes arguments and returns response data and error value if occurs
func (p *Peer) handleRPCCall(source rpccore.NodeID, method string, data []byte) ([]byte, error) {
	switch method {
	case rpcMethodRequestVote:
		var req requestVoteReq
		err := gob.NewDecoder(bytes.NewReader(data)).Decode(&req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		p.mutex.Lock()
		res := p.handleRequestVote(req)
		p.mutex.Unlock()
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	case rpcMethodAppendEntries:
		var req appendEntriesReq
		err := gob.NewDecoder(bytes.NewReader(data)).Decode(&req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		p.mutex.Lock()
		res := p.handleAppendEntries(req)
		p.mutex.Unlock()
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	// NOTE: Client request only works with transaction state machine.
	case client.RPCMethodLeaderRequest:
		p.mutex.Lock()
		res := p.handleClientLeaderRequest()
		p.mutex.Unlock()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	case client.RPCMethodActionRequest:
		var req client.ActionReq
		err := gob.NewDecoder(bytes.NewReader(data)).Decode(&req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		p.mutex.Lock()
		res := p.handleClientActionRequest(req)
		p.mutex.Unlock()
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	case client.RPCMethodQueryRequest:
		var req client.QueryReq
		err := gob.NewDecoder(bytes.NewReader(data)).Decode(&req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		p.mutex.Lock()
		res := p.handleClientQueryRequest(req)
		p.mutex.Unlock()
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	case rpcMethodInstallSnapshot:
		var req installSnapshotReq
		err := gob.NewDecoder(bytes.NewReader(data)).Decode(&req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		p.mutex.Lock()
		res := p.handleInstallSnapshot(req)
		p.mutex.Unlock()
		var buf bytes.Buffer
		err = gob.NewEncoder(&buf).Encode(res)
		return buf.Bytes(), errors.WithStack(err)
	default:
		err := errors.New(fmt.Sprintf("Unsupport method: %v", method))
		return nil, err
	}
}
