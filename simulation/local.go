package simulation

import (
	"os"
	"strconv"

	"github.com/PwzXxm/raft-lite/raft"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type local struct {
	n         int
	network   *rpccore.ChanNetwork
	rpcPeers  map[rpccore.NodeID]*rpccore.ChanNode
	raftPeers map[rpccore.NodeID]*raft.Peer
	loggers   map[rpccore.NodeID]*logrus.Logger

	// TODO: add rpc network
	// rpc network may simulate long delays and unreliable communications
	// add a random waiting time?
}

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Out = os.Stdout
}

func RunLocally(n int) *local {
	rf, err := new_local(n)
	if err != nil {
		log.Panicln(err)
	}

	log.Info("Starting simulation locally ...")

	for _, node := range rf.raftPeers {
		node.Start()
	}

	return rf
}

func new_local(n int) (*local, error) {
	if n <= 0 {
		err := errors.Errorf("The number of peers should be positive, but got %v", n)
		return nil, err
	}

	rf := new(local)
	rf.n = n
	rf.network = rpccore.NewChanNetwork()
	rf.rpcPeers = make(map[rpccore.NodeID]*rpccore.ChanNode)
	rf.raftPeers = make(map[rpccore.NodeID]*raft.Peer)
	rf.loggers = make(map[rpccore.NodeID]*logrus.Logger)

	// create rpc nodes
	for i := 0; i < n; i++ {
		addr := rpccore.NewChanAddress(strconv.Itoa(i))
		node, err := rf.network.NewNode(addr)
		if err != nil {
			errors.Errorf("Failed to allocate a new node with node id %v", node.NodeID)
			return nil, err
		}
		rf.rpcPeers[node.NodeID()] = node

		logger := logrus.New()
		logger.Out = os.Stdout
		rf.loggers[node.NodeID()] = logger
	}

	// gather nodeIDs and a map (nodeID -> idex in nodeIDs slice)
	nodeIDs := make([]rpccore.NodeID, n)
	idxMap := make(map[rpccore.NodeID]int)
	var i int = 0
	for k := range rf.rpcPeers {
		nodeIDs[i] = k
		idxMap[k] = i
		i++
	}

	// create raft nodes
	for id, node := range rf.rpcPeers {
		// move current nodeID to the last of the nodeIDs slice
		i, j := idxMap[id], n-1
		nodeIDs[i], nodeIDs[j] = nodeIDs[j], nodeIDs[i]
		idxMap[nodeIDs[i]] = i
		idxMap[id] = j

		// TODO: implement NewPeer func
		rf.raftPeers[id] = raft.NewPeer(node, nodeIDs[:n-1], rf.loggers[id].WithFields(logrus.Fields{
			"nodeID": node.NodeID(),
		}))
	}

	return rf, nil
}

func (rf *local) StopAll() {
	for _, node := range rf.raftPeers {
		node.ShutDown()
	}
}

func (rf *local) Request(cmd interface{}) {
}

func (rf *local) ShutDownPeer(id rpccore.NodeID) {
}

func (rf *local) ConnectPeer(id rpccore.NodeID) {
}

func (rf *local) DisconnectPeer(id rpccore.NodeID) {
}
