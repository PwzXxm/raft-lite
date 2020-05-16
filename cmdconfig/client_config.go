package cmdconfig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
)

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

type clientConfig struct {
	NodeAddrMap map[rpccore.NodeID]string
	ClientID    string
}

//StartClientFromFile is good
func StartClientFromFile(filePath string) error {
	config, err := readClientFromJSON(filePath)
	if err != nil {
		return err
	}
	fmt.Println(config)
	startReadingCmd()
	return nil
}

func startReadingCmd() {
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
				handleQueryRequest(cmd[1])
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
					handleSetRequest(cmd[1], value)
				case cmdIncre:
					handleIncreRequest(cmd[1], value)
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
				handleMoveRequest(cmd[1], cmd[2], value)
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

func readClientFromJSON(filepath string) (clientConfig, error) {
	v := clientConfig{}
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

func handleQueryRequest(key string) error {
	return nil
}

func handleSetRequest(key string, value int) error {
	return nil
}

func handleIncreRequest(key string, value int) error {
	return nil
}

func handleMoveRequest(key1, key2 string, value int) error {
	return nil
}
