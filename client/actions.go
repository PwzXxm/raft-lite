package client

import (
	"fmt"
	"os"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/pkg/errors"
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

	leaderID *rpccore.NodeID
}

func NewClientFromConfig(config interface{}) *Client {
	c := new(Client)

	// TODO: init
	// c.n =
	// c.clientID =
	c.net = rpccore.NewTCPNetwork(tcpTimeout)

	// TODO: change to no server version
	// c.node = c.net.NewLocalNode()

	// c.nl =
	// c.peers =

	c.ab = sm.NewTSMActionBuilder(c.clientID)

	// c.looger = logrus.New()
	c.logger.Out = os.Stdout

	return c
}

func (c *Client) parseAndBuild(cmd interface{}) (interface{}, error) {
	// parse

	// c.ab.TSMActionIncrValue()
	// c.ab.TSMActionMoveValue()
	// c.ab.TSMActionSetValue()
	// sm.NewTSMDataQuery(d)
	return nil, nil
}

func (c *Client) lookForLeader() rpccore.NodeID {
	// cached, the cache will be cleaned if there is any issue
	// blocking, keep trying until find a leader
	for c.leaderID != nil {
		pl := c.peers[c.nl[utils.Random(0, c.n-1)]].NodeID()
		var leaderRes LeaderRes
		err := c.callRPC(pl, RPCMethodLeaderRequest, "", &leaderRes)
		if err == nil {
			if leaderRes.HasLeader {
				c.logger.Infof("Node %v answered with leader = %v", pl,
					leaderRes.LeaderID)
				c.leaderID = &leaderRes.LeaderID
				return *c.leaderID
			} else {
				err = errors.Errorf("Node %v doesn't know the leader.", pl)
			}
		}
		c.logErrAndBackoff("Unable to find leader. ", err)
	}
	return *c.leaderID
}

func (c *Client) logErrAndBackoff(msg string, err error) {
	c.leaderID = nil
	c.logger.Debug(msg, err)
	// TODO: better backoff strategy?
	time.Sleep(100 * time.Millisecond)
}

func (c *Client) sendActionRequest(act sm.TSMAction) error {
	leader := c.lookForLeader()
	fmt.Print(leader)
	return nil
}

func (c *Client) checkActionRequest(act sm.TSMAction) (bool, error) {
	leader := c.lookForLeader()
	fmt.Print(leader)
	return false, nil
}

func (c *Client) executeActionRequest(act sm.TSMAction) {
	for {
		err := c.sendActionRequest(act)
		if err != nil {
			c.logErrAndBackoff("send action request failed. ", err)
			continue
		}

		// TODO: avg success time?
		time.Sleep(100 * time.Millisecond)

		for i := 0; i < 4; i++ {
			success, err := c.checkActionRequest(act)
			if err != nil {
				c.logErrAndBackoff("check action request failed. ", err)
			}
			if success {
				c.logger.Infof("action success.")
				return
			} else {
				// TODO: another backoff?
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (c *Client) Request(cmd interface{}) error {
	// tsmAction, err := parseAndBuild(cmd)

	// if err != nil {
	// 	return err
	// }

	// for {

	// 	if leaderRes.HasLeader {
	// 		l := leaderRes.LeaderID

	// 		c.callRPC(l)

	// 		sm.NewTSMLatestRequestQuery()
	// 	}
	// }
	// return nil
	return nil
}
