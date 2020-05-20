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

type LeaderRes struct {
	HasLeader bool
	LeaderID  rpccore.NodeID
}

type ActionReq struct {
	Cmd sm.TSMAction
}

type ActionRes struct {
	Started bool
}

type QueryReq struct {
	Cmd sm.TSMQuery
}

type QueryRes struct {
	Success  bool
	QueryErr *string
	Data     interface{}
}

func callRPC(core *ClientCore, target rpccore.NodeID, method string, req, res interface{}) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(req)
	if err != nil {
		return errors.WithStack(err)
	}
	resData, err := core.node.SendRawRequest(target, method, buf.Bytes())
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
