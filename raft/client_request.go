/*
 * Project: raft-lite
 * ---------------------
 * Authors:
 * Minjian Chen 813534
 * Shijie Liu   813277
 * Weizhi Xu    752454
 * Wenqing Xue  813044
 * Zijun Chen   813190
 */

package raft

import (
	"fmt"
	"time"

	"github.com/PwzXxm/raft-lite/client"
	"github.com/PwzXxm/raft-lite/rpccore"
)

const leaderRequestTimeout = 2 * time.Second

// HandleClientRequest returns a bool value,
// which tries to proceed client request with timeout
func (p *Peer) HandleClientRequest(cmd interface{}) bool {
	// double check leader status
	p.mutex.Lock()
	if p.state != Leader {
		p.mutex.Unlock()
		return false
	}
	p.mutex.Unlock()

	p.logger.Debugf("Received new request to append %v", cmd)

	// use timeout to check
	c := make(chan bool)
	go func() {
		c <- p.onReceiveClientRequest(cmd)
	}()

	select {
	case done := <-c:
		return done
	case <-time.After(leaderRequestTimeout):
		return false
	}
}

// handleClientLeaderRequest returns a LeaderRes struct,
// which includes a bool value whether leader exists and leader node ID
func (p *Peer) handleClientLeaderRequest() client.LeaderRes {
	if p.state != Leader {
		if p.leaderID == nil {
			return client.LeaderRes{HasLeader: false, LeaderID: rpccore.NodeID("")}
		}
		return client.LeaderRes{HasLeader: true, LeaderID: *p.leaderID}
	}
	return client.LeaderRes{HasLeader: true, LeaderID: p.node.NodeID()}
}

// handleClientActionRequest takes a ActionReq struct and returns a ActionRes struct
// only leader can return Started with a true bool value
func (p *Peer) handleClientActionRequest(req client.ActionReq) client.ActionRes {
	if p.state != Leader {
		return client.ActionRes{Started: false}
	}

	go func() {
		p.onReceiveClientRequest(req.Cmd)
	}()

	return client.ActionRes{Started: true}
}

// handleClientQueryRequest takes a QueryReq struct and returns a QueryRes struct
func (p *Peer) handleClientQueryRequest(req client.QueryReq) client.QueryRes {
	// no valid leader case
	if p.state != Leader || !p.isValidLeader() {
		return client.QueryRes{Success: false, QueryErr: nil, Data: nil}
	}

	v, err := p.stateMachine.Query(req.Cmd)
	if err != nil {
		// data is [nil] if the query is invalid
		serr := fmt.Sprint(err)
		return client.QueryRes{Success: true, QueryErr: &serr, Data: nil}
	}

	return client.QueryRes{Success: true, QueryErr: nil, Data: v}
}
