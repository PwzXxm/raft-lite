package simulation

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/PwzXxm/raft-lite/rpccore"
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
	cmdReq:      "<node_id> <log>",
	cmdStopAll:  "",
	cmdShutdown: "<node_id_1> <node_id_2> ...",
	cmdWait:     "<seconds>",
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
					break
				case cmdStopAll:
					rf.StopAll()
					break
				case cmdHelp:
					printUsage()
					break
				}

				break
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
			case cmdReq:
				// TODO: add request cmd
				break
			case cmdWait:
				if l != 2 {
					err = combineErrorUsage(invalidCommandError, cmd[0])
					break
				}

				sec, e := rf.getSeconds(cmd[1])
				if e != nil {
					err = e
					break
				}

				rf.Wait(sec)
			default:
				err = invalidCommandError
				break
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

func (rf *local) validateNodeIds(nodes []string, l, r int) ([]rpccore.NodeID, error) {
	rst := make([]rpccore.NodeID, 0)
	for i := l; i < r && i < len(nodes); i++ {
		addr := rpccore.NewChanAddress(nodes[i])
		if _, ok := rf.raftPeers[addr.NodeID()]; ok {
			rst = append(rst, addr.NodeID())
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

func printUsage() {
	fmt.Println("Usage: <cmd> <args> ...")
	var longest int = -1

	rst := make([]string, 0, len(usageMp))
	for cmd := range usageMp {
		l := len(cmd)
		if longest < l {
			longest = l
		}
		rst = append(rst, cmd)
	}

	sort.Strings(rst)

	for _, cmd := range rst {
		fmt.Printf("%-*v %v\n", longest, cmd, usageMp[cmd])
	}
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
	for k, v := range p.GetInfo() {
		fmt.Printf("%v: %v\n", k, v)
	}
}