package cmdconfig

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PwzXxm/raft-lite/pstorage"
	"github.com/PwzXxm/raft-lite/raft"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/goinggo/mapstructure"
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

func StartPeerFromFile(filepath string) error {
	configMap, err := utils.ReadClientFromJSON(raftConfig{}, filepath)
	if err != nil {
		return err
	}
	var config raftConfig
	err = mapstructure.Decode(configMap, &config)
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
	setupSignalHandler(p)
	select {}
}

func setupSignalHandler(p *raft.Peer) (stopCh <-chan struct{}) {
	var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, shutdownSignals...)
	go func() {
		<-c
		p.ShutDown()
		fmt.Println("Shutting down peer...")
		os.Exit(1)
	}()

	return stop
}
