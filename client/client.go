package client

import (
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"
)

type clientConfig struct {
	NodeAddrMap map[rpccore.NodeID]string
	ClientID    string
}

func StartClientFromFile(filepath string) error {
	var config clientConfig
	err := utils.ReadClientFromJSON(&config, filepath)
	if err != nil {
		return err
	}

	fl := flock.New(filepath)
	if locked, _ := fl.TryLock(); !locked {
		return errors.New("Unable to lock the config file," +
			" make sure there isn't another instance running.")
	}
	defer func() {
		_ = fl.Unlock()
	}()

	c, err := NewClientFromConfig(config)
	if err != nil {
		return err
	}
	c.startReadingCmd()
	return nil
}
