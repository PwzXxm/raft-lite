package cmdconfig

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/PwzXxm/raft-lite/pstorage"
	"github.com/PwzXxm/raft-lite/raft"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/sirupsen/logrus"
)

type raftConfig struct {
	Timeout          time.Duration
	NodeAddrMap      map[rpccore.NodeID]string
	NodeID           rpccore.NodeID
	ListenAddr       string
	PstorageFilePath string
	TimingFactor     int
}

//StartPeerFromFile is good
func StartPeerFromFile(filepath string) error {
	config, err := readPeerFromJSON(filepath)
	if err != nil {
		return err
	}
	//set logger
	logger := logrus.New()
	logger.Out = os.Stdout
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
	for {
	}
	return nil
}

func readPeerFromJSON(filepath string) (raftConfig, error) {
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
