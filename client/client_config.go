package client

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/goinggo/mapstructure"
)

type clientConfig struct {
	NodeAddrMap map[rpccore.NodeID]string
	ClientID    string
}

func StartClientFromFile(filepath string) error {
	config, err := utils.ReadClientFromJSON(clientConfig{}, filepath)
	if err != nil {
		return err
	}
	var cconfig clientConfig
	err = mapstructure.Decode(config, &cconfig)
	if err != nil {
		return err
	}
	c, err := NewClientFromConfig(cconfig)
	if err != nil {
		return err
	}
	c.startReadingCmd()
	return nil
}
