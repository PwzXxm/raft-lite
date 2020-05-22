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

package simulation

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/pkg/errors"
)

const (
	cmdID       = "id"
	cmdNodeInfo = "nodeinfo"
	cmdReq      = "request"
	cmdStopAll  = "stopall"
	cmdShutdown = "shutdown"
	cmdWait     = "wait"
	cmdHelp     = "help"
)

var usageMp = map[string]string{
	cmdID:       "",
	cmdNodeInfo: "<node_id_1> <node_id_2> ...",
	cmdReq:      "<log>",
	cmdStopAll:  "",
	cmdShutdown: "<node_id_1> <node_id_2> ...",
	cmdWait:     "<seconds>",
	cmdHelp:     "",
}

var scanner *bufio.Scanner

func init() {
	scanner = bufio.NewScanner(os.Stdin)
}

// StartReadingCMD reads cmd from STDIN until EOF
func (rf *local) StartReadingCMD() {
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
			case cmdID, cmdStopAll, cmdHelp:
				if l != 1 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}

				switch cmd[0] {
				case cmdID:
					rf.printIDs()
				case cmdStopAll:
					rf.StopAll()
				case cmdHelp:
					utils.PrintUsage(usageMp)
				}
			case cmdNodeInfo, cmdShutdown:
				if l < 2 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}

				nodes, e := rf.validateNodeIds(cmd, 1, len(cmd))
				if e != nil {
					err = e
					break
				}

				for _, node := range nodes {
					switch cmd[0] {
					case cmdNodeInfo:
						rf.printNodeInfo(node)
					case cmdShutdown:
						rf.ShutDownPeer(node)
					}
				}
			case cmdWait, cmdReq:
				if l != 2 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}

				switch cmd[0] {
				case cmdWait:
					sec, e := rf.getSeconds(cmd[1])
					if e != nil {
						err = e
						break
					}

					rf.Wait(sec)
				case cmdReq:
					// handle log as string at this time
					rf.RequestSync(cmd[1])
				}
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

// validateNodeIds checks whether the node id in current network
func (rf *local) validateNodeIds(nodes []string, l, r int) ([]rpccore.NodeID, error) {
	rst := make([]rpccore.NodeID, 0)
	for i := l; i < r && i < len(nodes); i++ {
		nodeID := rpccore.NodeID(nodes[i])
		if _, ok := rf.raftPeers[nodeID]; ok {
			rst = append(rst, nodeID)
		} else {
			return nil, errors.New("Unable to find node in the current list")
		}
	}
	return rst, nil
}

func (rf *local) getSeconds(sec string) (int, error) {
	x, err := strconv.Atoi(sec)
	if err != nil {
		return 0, err
	}
	return x, nil
}

func (rf *local) printIDs() {
	fmt.Print("[")
	rst := rf.getAllNodeIDs()
	for i, id := range rst {
		if i == 0 {
			fmt.Printf("%v", id)
		} else {
			fmt.Printf(" %v", id)
		}
	}
	fmt.Println("]")
}

func (rf *local) printNodeInfo(node rpccore.NodeID) {
	p := rf.raftPeers[node]
	fmt.Printf("Node info of [%v]\n", node)
	for k, v := range p.GetInfo() {
		fmt.Printf("  %v: %v\n", k, v)
	}
}
