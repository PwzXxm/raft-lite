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
	net  *rpccore.TCPNetwork
	core ClientCore
}

type ClientCore struct {
	ActBuilder *sm.TSMActionBuilder

	clientID        string
	leaderID        *rpccore.NodeID
	nl              []rpccore.NodeID
	node            rpccore.Node
	logger          *logrus.Logger
	backOffDuration int
}

const (
	maxBackOffDuration  = 1600 // ms
	initBackOffDuration = 100  // ms
)

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
	loggerLevelError  = "error"
)

var usageMp = map[string]string{
	cmdQuery:          "<key>",
	cmdSet:            "<key> <value>",
	cmdIncre:          "<key> <value>",
	cmdMove:           "<source> <target> <value>",
	cmdSetLoggerLevel: "<level> (warn, info, debug, error)",
}

func NewClientFromConfig(config clientConfig) (*Client, error) {
	c := new(Client)

	c.net = rpccore.NewTCPNetwork(tcpTimeout)

	cnode, err := c.net.NewLocalClientOnlyNode(rpccore.NodeID(config.ClientID))
	if err != nil {
		return nil, err
	}

	nl := make([]rpccore.NodeID, len(config.NodeAddrMap))
	i := 0
	for nodeID, addr := range config.NodeAddrMap {
		nl[i] = nodeID
		i++
		err := c.net.NewRemoteNode(nodeID, addr)
		if err != nil {
			return nil, err
		}
	}

	logger := logrus.New()
	logger.Out = os.Stdout
	logger.SetLevel(logrus.DebugLevel)

	c.core = NewClientCore(config.ClientID, nl, cnode, logger)

	return c, nil
}

func NewClientCore(clientID string, nodeIDs []rpccore.NodeID, cnode rpccore.Node, logger *logrus.Logger) ClientCore {
	return ClientCore{
		clientID:        clientID,
		leaderID:        nil,
		nl:              nodeIDs,
		ActBuilder:      sm.NewTSMActionBuilder(clientID),
		node:            cnode,
		logger:          logger,
		backOffDuration: initBackOffDuration,
	}
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
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				res, err := ExecuteQueryRequest(&c.core, sm.NewTSMDataQuery(cmd[1]))
				if err != nil {
					_, _ = red.Println(err)
				} else {
					green.Printf("The query result for key %v: %v\n", cmd[1], res)
				}
			case cmdSetLoggerLevel:
				if l != 2 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				switch cmd[1] {
				case loggerLevelDebug:
					c.core.logger.SetLevel(logrus.DebugLevel)
					_, _ = green.Println("Logger level set to debug")
				case loggerLevelInfo:
					c.core.logger.SetLevel(logrus.InfoLevel)
					_, _ = green.Println("Logger level set to info")
				case loggerLevelWarn:
					c.core.logger.SetLevel(logrus.WarnLevel)
					_, _ = green.Println("Logger level set to warn")
				case loggerLevelError:
					c.core.logger.SetLevel(logrus.ErrorLevel)
					_, _ = green.Println("Logger level set to error")
				default:
					err = combineErrorUsage(invalidCommandError, cmd[0])
				}
			case cmdSet, cmdIncre:
				if l != 3 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				value, e := strconv.Atoi(cmd[2])
				if e != nil {
					err = errors.New("value should be an integer")
					break
				}
				switch cmd[0] {
				case cmdSet:
					c.executeActionRequestAndPrint(c.core.ActBuilder.TSMActionSetValue(cmd[1], value))
				case cmdIncre:
					c.executeActionRequestAndPrint(c.core.ActBuilder.TSMActionIncrValue(cmd[1], value))
				}
			case cmdMove:
				if l != 4 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}
				value, e := strconv.Atoi(cmd[3])
				if e != nil {
					err = errors.New("value should be an integer")
					break
				}
				c.executeActionRequestAndPrint(c.core.ActBuilder.TSMActionMoveValue(cmd[1], cmd[2], value))
			default:
				_, _ = red.Fprintln(os.Stderr, invalidCommandError)
				utils.PrintUsage(usageMp)
			}
		}
		if err != nil {
			_, _ = red.Fprintln(os.Stderr, err)
		}
		green.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading stdout: ", err)
	}
}

func (c *Client) executeActionRequestAndPrint(act sm.TSMAction) {
	success, msg := ExecuteActionRequest(&c.core, act)
	var ca color.Attribute
	if success {
		ca = color.FgGreen
	} else {
		ca = color.FgHiRed
	}
	_, _ = color.New(ca).Println(msg)
}

func combineErrorUsage(e error, cmd string) error {
	return errors.New(e.Error() + "\nUsage: " + cmd + " " + usageMp[cmd])
}

func lookForLeader(core *ClientCore) rpccore.NodeID {
	// cached, the cache will be cleaned if there is any issue
	// blocking, keep trying until find a leader
	for core.leaderID == nil {
		// select a client by random
		pl := core.nl[utils.Random(0, len(core.nl)-1)]
		var leaderRes LeaderRes
		err := callRPC(core, pl, RPCMethodLeaderRequest, "", &leaderRes)
		if err == nil {
			if leaderRes.HasLeader {
				core.logger.Infof("Node %v answered with leader = %v", pl,
					leaderRes.LeaderID)
				core.leaderID = &leaderRes.LeaderID
				resetBackOffDuration(core)
				return *core.leaderID
			}

			err = errors.Errorf("Node %v doesn't know the leader.", pl)
		}
		logErrAndBackoff(core, "Unable to find leader. ", err)
	}
	return *core.leaderID
}

func resetBackOffDuration(core *ClientCore) {
	core.backOffDuration = initBackOffDuration
}

func logErrAndBackoff(core *ClientCore, msg string, err error) {
	core.leaderID = nil
	core.logger.Debug(msg, err)

	// this function can only be called when one action failed
	// thus, only one counter is necessary
	time.Sleep(time.Duration(core.backOffDuration) * time.Millisecond)

	core.backOffDuration = utils.Min(maxBackOffDuration, core.backOffDuration*2)
}

func sendActionRequest(core *ClientCore, actReq ActionReq) error {
	leader := lookForLeader(core)
	var actionRes ActionRes
	err := callRPC(core, leader, RPCMethodActionRequest, actReq, &actionRes)
	if err == nil && !actionRes.Started {
		err = errors.Errorf("Node %v declined the request.", leader)
	}
	return err
}

func checkActionRequest(core *ClientCore, queryReq QueryReq) (*sm.TSMRequestInfo, error) {
	leader := lookForLeader(core)
	var queryRes QueryRes
	err := callRPC(core, leader, RPCMethodQueryRequest, queryReq, &queryRes)
	if err == nil {
		if queryRes.Success {
			if queryRes.QueryErr == nil {
				info := queryRes.Data.(sm.TSMRequestInfo)
				return &info, nil
			}
			// query success, but there is no related request info
			return nil, nil
		}
		err = errors.Errorf("Node %v decliend the query request.", leader)
	}
	return nil, err
}

func ExecuteActionRequest(core *ClientCore, act sm.TSMAction) (bool, string) {
	actReq := ActionReq{Cmd: act}
	queryReq := QueryReq{Cmd: sm.NewTSMLatestRequestQuery(core.clientID)}
	reqID := act.GetRequestID()
	for {
		err := sendActionRequest(core, actReq)
		if err != nil {
			logErrAndBackoff(core, "send action request failed. ", err)
			continue
		}
		resetBackOffDuration(core)

		for i := 0; i < 4; i++ {
			info, err := checkActionRequest(core, queryReq)
			if err != nil {
				logErrAndBackoff(core, "check action request failed. ", err)
			}

			if info != nil && info.RequestID == reqID {
				resetBackOffDuration(core)
				if info.Err != nil {
					return false, *info.Err
				}

				return true, "action success"
			}

			if err == nil {
				logErrAndBackoff(core, "info is nil or wrong request ID", err)
			}
		}
	}
}

func ExecuteQueryRequest(core *ClientCore, query sm.TSMQuery) (interface{}, error) {
	queryReq := QueryReq{Cmd: query}
	for {
		leader := lookForLeader(core)
		var queryRes QueryRes
		err := callRPC(core, leader, RPCMethodQueryRequest, queryReq, &queryRes)
		if err == nil {
			if queryRes.Success {
				resetBackOffDuration(core)
				if queryRes.QueryErr == nil {
					return queryRes.Data, nil
				}

				return nil, errors.New(*queryRes.QueryErr)
			}

			err = errors.Errorf("Node %v decliend the query request.", leader)
		}
		if err != nil {
			logErrAndBackoff(core, "Request query failed. ", err)
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
