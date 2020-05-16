package client

import (
	"os"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/sirupsen/logrus"
)

const tcpTimeout = time.Second

type Client struct {
	n        int
	clientID string
	net      *rpccore.TCPNetwork
	node     rpccore.Node
	nl       []rpccore.NodeID
	peers    map[rpccore.NodeID]*rpccore.TCPNode
	ab       *sm.TSMActionBuilder

	logger logrus.Logger
}

func NewClientFromConfig(config interface{}) *Client {
	c := new(Client)

    // TODO: init
    c.n = 
    c.clientID = 
    c.net = rpccore.NewTCPNetwork(tcpTimeout)

    // TODO: change to no server version
    c.node = c.net.NewLocalNode()

    c.nl = 
    c.peers = 

    c.ab = sm.NewTSMActionBuilder(c.clientID)

    c.looger = logrus.New()
    c.logger.Out = os.Stdout

	return c
}

func (c *Client) parseAndBuild(cmd interface{}) (interface{}, error) {
	// parse

	c.ab.TSMActionIncrValue()
	c.ab.TSMActionMoveValue()
	c.ab.TSMActionSetValue()
	sm.NewTSMDataQuery(d)
}

func (c *Client) Request(cmd interface{}) error {
	tsmAction, err := parseAndBuild(cmd)

	if err != nil {
		return err
	}

	for {
		// find leader
		pl := c.peers[c.nl[utils.Random(0, c.n-1)]].NodeID()

		var leaderRes LeaderRes
		err := c.callRPC(pl, RPCMethodLeaderRequest, "", &leaderRes)
		if err != nil {
			c.logger.Warn("RPC call failed ", err)
			continue
		}

		if leaderRes.HasLeader {
            l := leaderRes.LeaderID
            



			c.callRPC(l)

			sm.NewTSMLatestRequestQuery()
		}
	}
	return nil
}
