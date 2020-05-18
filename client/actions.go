package client

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Client struct {
	n        int
	clientID string
	net      *rpccore.TCPNetwork
	node     rpccore.Node
	nl       []rpccore.NodeID
	ab       *sm.TSMActionBuilder

	logger *logrus.Logger

	leaderID *rpccore.NodeID
}

const (
	tcpTimeout        = time.Second
	cmdQuery          = "query"
	cmdSet            = "set"
	cmdIncre          = "increment"
	cmdMove           = "move"
	cmdSetLoggerLevel = "loggerLevel"
	loggerLevelDebug  = "debug"
	loggerLevelInfo   = "info"
	loggerLevelWarn   = "warn"
)

var usageMp = map[string]string{
	cmdQuery:          "<key>",
	cmdSet:            "<key> <value>",
	cmdIncre:          "<key> <value>",
	cmdMove:           "<source> <target> <value>",
	cmdSetLoggerLevel: "<level> (warn, info, debug)",
}

func NewClientFromConfig(config clientConfig) (*Client, error) {
	c := new(Client)

	c.n = len(config.NodeAddrMap)
	c.clientID = config.ClientID
	c.net = rpccore.NewTCPNetwork(tcpTimeout)
	cnode, err := c.net.NewLocalClientOnlyNode(rpccore.NodeID(config.ClientID))
	if err != nil {
		return nil, err
	}
	c.node = cnode

	c.nl = make([]rpccore.NodeID, c.n)
	i := 0
	for nodeID, addr := range config.NodeAddrMap {
		c.nl[i] = nodeID
		i++
		err := c.net.NewRemoteNode(nodeID, addr)
		if err != nil {
			return nil, err
		}
	}

	c.ab = sm.NewTSMActionBuilder(c.clientID)

	c.logger = logrus.New()
	c.logger.Out = os.Stdout
	c.logger.SetLevel(logrus.DebugLevel)

	return c, nil
}

func (c *Client) startReadingCmd() {
	printWelcomeMsg()

	invalidCommandError := errors.New("Invalid command")
	var err error

	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	green.Print("> ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := strings.Fields(scanner.Text())

		err = nil
		l := len(cmd)

		if l == 0 {
			err = errors.New("Command cannot be empty")
		}

		if err == nil {
			switch cmd[0] {
			case cmdQuery:
				if l != 2 {
					err = c.combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				c.executeQueryRequest(sm.NewTSMDataQuery(cmd[1]))
			case cmdSetLoggerLevel:
				if l != 2 {
					err = c.combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				switch cmd[1] {
				case loggerLevelDebug:
					c.logger.SetLevel(logrus.DebugLevel)
					green.Println("Logger level set to debug")
				case loggerLevelInfo:
					c.logger.SetLevel(logrus.InfoLevel)
					green.Println("Logger level set to info")
				case loggerLevelWarn:
					c.logger.SetLevel(logrus.WarnLevel)
					green.Println("Logger level set to warn")
				default:
					err = c.combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
			case cmdSet, cmdIncre:
				if l != 3 {
					err = c.combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				value, e := strconv.Atoi(cmd[2])
				if e != nil {
					err = errors.New("value should be an integer")
					break
				}
				switch cmd[0] {
				case cmdSet:
					c.executeActionRequest(c.ab.TSMActionSetValue(cmd[1], value))
				case cmdIncre:
					c.executeActionRequest(c.ab.TSMActionIncrValue(cmd[1], value))
				}
			case cmdMove:
				if l != 4 {
					err = c.combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				value, e := strconv.Atoi(cmd[3])
				if e != nil {
					err = errors.New("value should be an integer")
					break
				}
				c.executeActionRequest(c.ab.TSMActionMoveValue(cmd[1], cmd[2], value))
			default:
				red.Fprintln(os.Stderr, invalidCommandError)
				utils.PrintUsage(usageMp)
			}
		}
		if err != nil {
			red.Fprintln(os.Stderr, err)
		}
		green.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading stdout: ", err)
	}
}

func (c *Client) combineErrorUsage(e error, cmd string) error {
	return errors.New(e.Error() + "\nUsage: " + cmd + " " + usageMp[cmd])
}

func (c *Client) lookForLeader() rpccore.NodeID {
	// cached, the cache will be cleaned if there is any issue
	// blocking, keep trying until find a leader
	for c.leaderID == nil {
		// select a client by random
		pl := c.nl[utils.Random(0, c.n-1)]
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

func (c *Client) sendActionRequest(actReq ActionReq) error {
	leader := c.lookForLeader()
	var actionRes ActionRes
	err := c.callRPC(leader, RPCMethodActionRequest, actReq, &actionRes)
	if err == nil && !actionRes.Started {
		err = errors.Errorf("Node %v declined the request.", leader)
	}
	return err
}

func (c *Client) checkActionRequest(queryReq QueryReq, reqID uint32) (bool, error) {
	leader := c.lookForLeader()
	var queryRes QueryRes
	err := c.callRPC(leader, RPCMethodQueryRequest, queryReq, &queryRes)
	if err == nil {
		if queryRes.Success {
			return (queryRes.QueryErr == nil && queryRes.Data.(uint32) == reqID), nil
		} else {
			err = errors.Errorf("Node %v decliend the query request.", leader)
		}
	}
	return false, err
}

func (c *Client) executeActionRequest(act sm.TSMAction) {
	actReq := ActionReq{Cmd: act}
	queryReq := QueryReq{Cmd: sm.NewTSMLatestRequestQuery(c.clientID)}
	reqID := act.GetRequestID()
	for {
		err := c.sendActionRequest(actReq)
		if err != nil {
			c.logErrAndBackoff("send action request failed. ", err)
			continue
		}

		// TODO: avg success time?
		time.Sleep(100 * time.Millisecond)

		for i := 0; i < 4; i++ {
			success, err := c.checkActionRequest(queryReq, reqID)
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

func (c *Client) executeQueryRequest(query sm.TSMQuery) (interface{}, error) {
	queryReq := QueryReq{Cmd: query}
	for {
		leader := c.lookForLeader()
		var queryRes QueryRes
		err := c.callRPC(leader, RPCMethodQueryRequest, queryReq, &queryRes)
		if err == nil {
			if queryRes.Success {
				if queryRes.QueryErr == nil {
					return queryRes.Data, nil
				} else {
					return nil, errors.New(*queryRes.QueryErr)
				}
			} else {
				err = errors.Errorf("Node %v decliend the query request.", leader)
			}
		}
		if err != nil {
			c.logErrAndBackoff("Request query failed. ", err)
			continue
		}
	}
}

func printWelcomeMsg() {
	fmt.Printf("\n=============================================\n")
	figure.NewFigure("Raft lite", "doom", true).Print()
	fmt.Printf("\n\n Welcome to Raft Lite Transaction System\n")
	fmt.Printf("\n=============================================\n")
}
