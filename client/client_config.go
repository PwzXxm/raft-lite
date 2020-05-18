package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/PwzXxm/raft-lite/rpccore"
)

type clientConfig struct {
	NodeAddrMap map[rpccore.NodeID]string
	ClientID    string
}

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
