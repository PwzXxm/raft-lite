package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/PwzXxm/raft-lite/pstorage"
	"github.com/PwzXxm/raft-lite/raft"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/gofrs/flock"
	"github.com/sirupsen/logrus"
)

type peerConfig struct {
	Timeout           time.Duration
	NodeAddrMap       map[rpccore.NodeID]string
	NodeID            rpccore.NodeID
	ListenAddr        string
	PstorageFilePath  string
	TimingFactor      int
	SnapshotThreshold int
}

func StartPeerFromFile(configFilepath string) error {
	var config peerConfig
	err := utils.ReadClientFromJSON(&config, configFilepath)
	if err != nil {
		return err
	}

	fl := flock.New(configFilepath)
	if locked, _ := fl.TryLock(); !locked {
		return errors.New("Unable to lock the config file," +
			" make sure there isn't another instance running.")
	}
	defer func() {
		_ = fl.Unlock()
	}()

	// new tcp network
	n := rpccore.NewTCPNetwork(config.Timeout * time.Second)
	node, err := n.NewLocalNode(config.NodeID, config.NodeAddrMap[config.NodeID], config.ListenAddr)
	if err != nil {
		return err
	}
	for nodeID, addr := range config.NodeAddrMap {
		if nodeID != config.NodeID {
			err = n.NewRemoteNode(nodeID, addr)
			if err != nil {
				return err
			}
		}
	}

	// set logger
	logger := logrus.New()
	logger.Out = os.Stdout
	loggerEntry := logger.WithFields(logrus.Fields{"nodeID": node.NodeID()})

	// create directory for Pstorage if needed
	pFilePath := filepath.Dir(config.PstorageFilePath)

	if _, err := os.Stat(pFilePath); os.IsNotExist(err) {
		err = os.MkdirAll(pFilePath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	ps := pstorage.NewHybridPersistentStorage(config.PstorageFilePath,
		2*time.Second, loggerEntry)
	// new peer
	peers := []rpccore.NodeID{}
	for peer := range config.NodeAddrMap {
		if peer != config.NodeID {
			peers = append(peers, peer)
		}
	}
	p, err := raft.NewPeer(node, peers, loggerEntry,
		sm.NewTransactionStateMachine(), ps, config.TimingFactor,
		config.SnapshotThreshold)
	if err != nil {
		return err
	}
	p.Start()

	// wait for stop signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// start shutdown process
	fmt.Println("Shutting down peer...")
	p.ShutDown()
	time.Sleep(2 * time.Second)
	n.Shutdown()
	return ps.Stop()
}
