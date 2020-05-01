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

	l, err := newLocal(n)
	if err != nil {
		log.Panicln(err)
	}

	for _, node := range l.raftPeers {
		node.Start()
	}

	return l
}

func newLocal(n int) (*local, error) {
	if n <= 0 {
		err := errors.Errorf("The number of peers should be positive, but got %v", n)
		return nil, err
	}

	l := new(local)
	l.n = n
	l.network = rpccore.NewChanNetwork(4 * time.Second)
	l.rpcPeers = make(map[rpccore.NodeID]*rpccore.ChanNode)
	l.raftPeers = make(map[rpccore.NodeID]*raft.Peer)
	l.loggers = make(map[rpccore.NodeID]*logrus.Logger)

	// create rpc nodes
	for i := 0; i < n; i++ {
		node, err := l.network.NewNode(rpccore.NodeID(strconv.Itoa(i)))
		if err != nil {
			err = errors.Errorf("Failed to allocate a new node with node id %v", node.NodeID())
			return nil, err
		}
		l.rpcPeers[node.NodeID()] = node

		logger := logrus.New()
		logger.Out = os.Stdout
		l.loggers[node.NodeID()] = logger
	}

	// gather nodeIDs and a map (nodeID -> idex in nodeIDs slice)
	nodeIDs := make([]rpccore.NodeID, n)
	idxMap := make(map[rpccore.NodeID]int)
	var i int = 0
	for k := range l.rpcPeers {
		nodeIDs[i] = k
		idxMap[k] = i
		i++
	}

	// create raft nodes
	for id, node := range l.rpcPeers {
		// move current nodeID to the last of the nodeIDs slice
		i, j := idxMap[id], n-1
		nodeIDs[i], nodeIDs[j] = nodeIDs[j], nodeIDs[i]
		idxMap[nodeIDs[i]] = i
		idxMap[id] = j

		l.raftPeers[id] = raft.NewPeer(node, nodeIDs[:n-1], l.loggers[id].WithFields(logrus.Fields{
			"nodeID": node.NodeID(),
		}))
	}

	return l, nil
}

func (l *local) StopAll() {
	for _, node := range l.raftPeers {
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

func (l *local) ShutDownPeer(id rpccore.NodeID) {
	l.raftPeers[id].ShutDown()
}

func (l *local) ConnectPeer(id rpccore.NodeID) {
}

func (l *local) DisconnectPeer(id rpccore.NodeID) {
}

func (l *local) Wait(sec int) {
	if sec <= 0 {
		log.Warnf("Seconds to wait should be positive integer, not %v", sec)
		return
	}

	log.Infof("Sleeping for %v second(s)", sec)
	time.Sleep(time.Duration(sec) * time.Second)
}

func (l *local) getAllNodeIDs() []rpccore.NodeID {
	rst := make([]rpccore.NodeID, len(l.rpcPeers))
	i := 0
	for _, rpcNode := range l.rpcPeers {
		rst[i] = rpcNode.NodeID()
		i++
	}
	return rst
}

func (l *local) getAllNodeInfo() map[rpccore.NodeID]map[string]string {
	m := make(map[rpccore.NodeID]map[string]string)
	for nodeID, peer := range l.raftPeers {
		m[nodeID] = peer.GetInfo()
	}
	return m
}

func (l *local) AgreeOnLeader() (*rpccore.NodeID, error) {
	var leaderID *rpccore.NodeID
	for nodeID, peer := range l.raftPeers {
		if peer.GetState() == raft.Leader {
			if leaderID == nil {
				n := nodeID
				leaderID = &n
			} else {
				return nil, errors.Errorf("Failed to agree on leader.\n\n%v\n",
					l.getAllNodeInfo())
			}
		}
	}
	if leaderID == nil {
		return nil, errors.Errorf("Unable to find leader.\n\n%v\n", l.getAllNodeInfo())
	}
	return leaderID, nil
}

func (l *local) AgreeOnTerm() (int, error) {
	term := -1
	for _, peer := range l.raftPeers {
		if term == -1 {
			term = peer.GetTerm()
		} else {
			if term != peer.GetTerm() {
				return 0, errors.Errorf("Failed to agree on leader.\n\n%v\n",
					l.getAllNodeInfo())
			}
		}
	}
	return term, nil
}
