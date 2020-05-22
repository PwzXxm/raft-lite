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

package client

import (
	"bytes"
	"encoding/gob"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/pkg/errors"
)

const (
	RPCMethodLeaderRequest = "cl"
	RPCMethodActionRequest = "ca"
	RPCMethodQueryRequest  = "cq"
)

// LeaderRes representing the leader response
type LeaderRes struct {
	HasLeader bool
	LeaderID  rpccore.NodeID
}

// ActionReq representing the action request
type ActionReq struct {
	Cmd sm.TSMAction
}

// ActionRes representing the action response
type ActionRes struct {
	Started bool
}

// QueryReq representing the query request
type QueryReq struct {
	Cmd sm.TSMQuery
}

// QueryRes representing the query response
type QueryRes struct {
	Success  bool
	QueryErr *string
	Data     interface{}
}

// callRPC takes arguments and returns error value if occurs
func callRPC(core *ClientCore, target rpccore.NodeID, method string, req, res interface{}) error {
	var buf bytes.Buffer
	// encore request data
	err := gob.NewEncoder(&buf).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}
	// send raw request
	resData, err := core.node.SendRawRequest(target, method, buf.Bytes())
	if err != nil {
		// already wrapped
		return err
	}
	// decode request data
	err = gob.NewDecoder(bytes.NewReader(resData)).Decode(res)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
