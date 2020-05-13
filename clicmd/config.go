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

type tcpConfig struct {
	timeout          time.Duration
	nodeAddrMap      map[rpccore.NodeID]string
	nodeID           rpccore.NodeID
	remoteAddr       string
	listenAddr       string
	peers            []rpccore.NodeID
	sm               sm.StateMachine
	pstorageFilePath string
	logPath          string
}

//StartFromFile is good
func StartFromFile(filepath string) error {
	tcpConfig := readFromJSON(filepath)
	n := rpccore.NewTCPNetwork(tcpConfig.timeout)
	node, err := n.NewLocalNode(tcpConfig.nodeID, tcpConfig.remoteAddr, tcpConfig.listenAddr)
	if err != nil {
		return err
	}
	for nodeID, addr := range tcpConfig.nodeAddrMap {
		n.NewRemoteNode(nodeID, addr)
	}
	logger := logrus.New()
	logger.Out = os.Stdout
	ps := pstorage.NewFileBasedPersistentStorage(tcpConfig.pstorageFilePath)
	p, err := raft.NewPeer(node, tcpConfig.peers, logger, tcpConfig.sm, ps)
	return nil
}

func readFromJSON(filepath string) tcpConfig {
	return tcpConfig{}
}
