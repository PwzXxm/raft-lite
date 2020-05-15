package clicmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PwzXxm/raft-lite/pstorage"
	"github.com/PwzXxm/raft-lite/raft"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type raftConfig struct {
	Timeout          time.Duration
	NodeAddrMap      map[rpccore.NodeID]string
	NodeID           rpccore.NodeID
	ListenAddr       string
	PstorageFilePath string
	LogPath          string
	TimingFactor     int
}

const (
	cmdQuery = "query"
	cmdSet   = "set"
	cmdIncre = "increment"
	cmdMove  = "move"
)

var usageMp = map[string]string{
	cmdQuery: "<key>",
	cmdSet:   "<key> <value>",
	cmdIncre: "<key> <value>",
	cmdMove:  "<source> <target> <value>",
}

var scanner *bufio.Scanner

func init() {
	scanner = bufio.NewScanner(os.Stdin)
}

//StartFromFile is good
func StartFromFile(filepath string) error {
	config, err := readFromJSON(filepath)
	if err != nil {
		return err
	}
	//set logger
	logger := logrus.New()
	file, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	defer file.Close()
	if err == nil {
		logger.Out = file
	} else {
		logger.Info("Failed to log to file, using default stderr")
		logger.Out = os.Stdout
	}
	//new tcp network
	n := rpccore.NewTCPNetwork(config.Timeout * time.Second)
	node, err := n.NewLocalNode(config.NodeID, config.NodeAddrMap[config.NodeID], config.ListenAddr)
	if err != nil {
		return err
	}
	for nodeID, addr := range config.NodeAddrMap {
		n.NewRemoteNode(nodeID, addr)
	}
	ps := pstorage.NewFileBasedPersistentStorage(config.PstorageFilePath)
	//new peer
	peers := []rpccore.NodeID{}
	for peer := range config.NodeAddrMap {
		if peer != config.NodeID {
			peers = append(peers, peer)
		}
	}
	p, err := raft.NewPeer(node, peers, logger.WithFields(logrus.Fields{
		"nodeID": node.NodeID()}), sm.NewTransactionStateMachine(), ps, config.TimingFactor)
	if err != nil {
		return err
	}
	p.Start()
	startReadingCmd(p)
	return nil
}

func readFromJSON(filepath string) (raftConfig, error) {
	v := raftConfig{}
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(data, &v)
	if err != nil {
		return v, err
	}
	return v, nil
}

func startReadingCmd(p *raft.Peer) {
	invalidCommandError := errors.New("Invalid command")
	var err error

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
				result, e := p.QueryStateMachine(cmd[1])
				if e != nil {
					err = e
					break
				}
				fmt.Print("The query result for key: ", cmd[1], " is ", result)
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
					go func() {
						p.HandleClientCmd(sm.TSMActionSetValue(cmd[1], value))
						fmt.Println("Request ", cmd, " sent")
					}()
				case cmdIncre:
					go func() {
						p.HandleClientCmd(sm.TSMActionIncrValue(cmd[1], value))
						fmt.Println("Request ", cmd, " sent")
					}()
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
				go func() {
					p.HandleClientCmd(sm.TSMActionMoveValue(cmd[1], cmd[2], value))
					fmt.Println("Request ", cmd, " sent")
				}()
			default:
				err = invalidCommandError
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading stdout: ", err)
	}
}

func combineErrorUsage(e error, cmd string) error {
	return errors.New(e.Error() + "\nUsage: " + cmd + " " + usageMp[cmd])
}
