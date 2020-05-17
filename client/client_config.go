package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/PwzXxm/raft-lite/rpccore"
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
	c, err := NewClientFromConfig(config)
	if err != nil {
		return err
	}
	c.startReadingCmd()
	return nil
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
