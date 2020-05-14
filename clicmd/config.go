package clicmd

import (
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
	RemoteAddr       string
	ListenAddr       string
	Peers            []rpccore.NodeID
	Sm               sm.StateMachine
	PstorageFilePath string
	LogPath          string
	TimingFactor     int
}

//StartFromFile is good
func StartFromFile(filepath string) error {
	config := readFromJSON(filepath)
	n := rpccore.NewTCPNetwork(config.Timeout)
	node, err := n.NewLocalNode(config.NodeID, config.RemoteAddr, config.ListenAddr)
	if err != nil {
		return err
	}
	for nodeID, addr := range config.NodeAddrMap {
		n.NewRemoteNode(nodeID, addr)
	}
	logger := logrus.New()

	file, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.Out = file
	} else {
		logger.Info("Failed to log to file, using default stderr")
		logger.Out = os.Stdout
	}
	ps := pstorage.NewFileBasedPersistentStorage(config.PstorageFilePath)
	p, err := raft.NewPeer(node, config.Peers, logger.WithFields(logrus.Fields{
		"nodeID": node.NodeID()}), config.Sm, ps, config.TimingFactor)
	startReadingCmd(p)
	return nil
}

func readFromJSON(filepath string) raftConfig {
	return raftConfig{}
}

func startReadingCmd(p *raft.Peer) {
	
}
