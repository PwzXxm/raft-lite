package clicmd

import (
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
)

type tcpConfig struct {
	timeout     time.Duration
	nodeAddrMap map[rpccore.NodeID]string
	nodeID      rpccore.NodeID
	remoteAddr  string
	listenAddr  string
}

//StartFromFile is good
func StartFromFile(filepath string) error {
	tcpConfig := readFromJSON(filepath)
	n := rpccore.NewTCPNetwork(tcpConfig.timeout)
	n.NewLocalNode(tcpConfig.nodeID, tcpConfig.remoteAddr, tcpConfig.listenAddr)
	for nodeID, addr := range tcpConfig.nodeAddrMap {
		n.NewRemoteNode(nodeID, addr)
	}
	
	return nil
}

func readFromJSON(filepath string) tcpConfig {
	return tcpConfig{}
}
