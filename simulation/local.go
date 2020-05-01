package simulation

import (
	"os"
	"strconv"
	"time"

	"github.com/PwzXxm/raft-lite/raft"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const clientRequestTimeout = 5 * time.Second

type local struct {
	n         int
	network   *rpccore.ChanNetwork
	rpcPeers  map[rpccore.NodeID]*rpccore.ChanNode
	raftPeers map[rpccore.NodeID]*raft.Peer
	loggers   map[rpccore.NodeID]*logrus.Logger
}

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Out = os.Stdout
}

func RunLocally(n int) *local {
	log.Info("Starting simulation locally ...")

	rf, err := newLocal(n)
	if err != nil {
		log.Panicln(err)
	}

	for _, node := range rf.raftPeers {
		node.Start()
	}

	return rf
}

func newLocal(n int) (*local, error) {
	if n <= 0 {
		err := errors.Errorf("The number of peers should be positive, but got %v", n)
		return nil, err
	}

	rf := new(local)
	rf.n = n
	rf.network = rpccore.NewChanNetwork(4 * time.Second)
	rf.rpcPeers = make(map[rpccore.NodeID]*rpccore.ChanNode)
	rf.raftPeers = make(map[rpccore.NodeID]*raft.Peer)
	rf.loggers = make(map[rpccore.NodeID]*logrus.Logger)

	// create rpc nodes
	for i := 0; i < n; i++ {
		addr := rpccore.NewChanAddress(strconv.Itoa(i))
		node, err := rf.network.NewNode(addr)
		if err != nil {
			errors.Errorf("Failed to allocate a new node with node id %v", node.NodeID())
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
	for _, p := range rf.raftPeers {
		if p.HandleClientRequest(cmd) {
			return
		}
	}

	log.Warnf("Request failed")
}

func (rf *local) ShutDownPeer(id rpccore.NodeID) {
	rf.raftPeers[id].ShutDown()
}

func (rf *local) ConnectPeer(id rpccore.NodeID) {
}

func (rf *local) DisconnectPeer(id rpccore.NodeID) {
}

func (rf *local) Wait(sec int) {
	if sec <= 0 {
		log.Warnf("Seconds to wait should be positive integer, not %v", sec)
		return
	}

	log.Infof("Sleeping for %v second(s)", sec)
	time.Sleep(time.Duration(sec) * time.Second)
}

func (rf *local) getAllNodeIDs() []rpccore.NodeID {
	rst := make([]rpccore.NodeID, len(rf.rpcPeers))
	i := 0
	for _, rpcNode := range rf.rpcPeers {
		rst[i] = rpcNode.NodeID()
		i++
	}
	return rst
}
