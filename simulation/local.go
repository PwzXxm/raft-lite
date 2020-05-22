package simulation

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/PwzXxm/raft-lite/client"
	"github.com/PwzXxm/raft-lite/pstorage"
	"github.com/PwzXxm/raft-lite/raft"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	testTimingFactor         = 1
	clientRequestTimeout     = 1 * time.Second
	rpcTimeout               = 80 * testTimingFactor * time.Millisecond
	defaultSnapshotThreshold = 10000
)

type Local struct {
	n                 int
	network           *rpccore.ChanNetwork
	rpcPeers          map[rpccore.NodeID]*rpccore.ChanNode
	raftPeers         map[rpccore.NodeID]*raft.Peer
	logger            *logrus.Logger
	pstorages         map[rpccore.NodeID]pstorage.PersistentStorage
	smMaker           stateMachineMaker
	snapshotThreshold int
	clientCore        client.ClientCore

	// network related
	netLock          sync.RWMutex
	offlineNodes     map[rpccore.NodeID]bool
	nodePartition    map[rpccore.NodeID]int
	oneWayLatencyMin time.Duration
	oneWayLatencyMax time.Duration
	packetLossRate   float64
}

type stateMachineMaker func() sm.StateMachine

var Log *logrus.Logger

func init() {
	Log = logrus.New()
	Log.Out = os.Stdout
}

func RunLocally(n int) *Local {
	return RunLocallyOptional(n, defaultSnapshotThreshold, func() sm.StateMachine { return sm.NewEmptyStateMachine() })
}

func RunLocallyOptional(n int, snapshotThreshold int, smMaker stateMachineMaker) *Local {
	Log.Info("Starting simulation locally ...")

	l, err := newLocalOptional(n, snapshotThreshold, smMaker)
	if err != nil {
		Log.Panicln(err)
	}

	for _, node := range l.raftPeers {
		node.Start()
	}

	return l
}

func SetupLocally(n int) *Local {
	Log.Info("Setting up simulation locally ...")

	l, err := newLocal(n)
	if err != nil {
		Log.Panicln(err)
	}

	return l
}

func (l *Local) StartAll() {
	for _, node := range l.raftPeers {
		node.Start()
	}
}

// Current limitation:
// 	1. latency is uniform distribution
// 	2. don't support one way connection lost (packet lost is one way)
// 	2. all nodes share the same latency and packet lost rate
func (l *Local) delayGenerator(source, target rpccore.NodeID) time.Duration {
	l.netLock.RLock()
	defer l.netLock.RUnlock()
	if !l.offlineNodes[source] && !l.offlineNodes[target] {
		if l.nodePartition[source] == l.nodePartition[target] {
			if !utils.RandomBool(l.packetLossRate) {
				return utils.RandomTime(l.oneWayLatencyMin, l.oneWayLatencyMax)
			}
		}
	}
	return rpcTimeout + time.Second
}

func newLocal(n int) (*Local, error) {
	return newLocalOptional(n, defaultSnapshotThreshold, func() sm.StateMachine { return sm.NewEmptyStateMachine() })
}

func newLocalOptional(n int, snapshotThreshold int, smMaker stateMachineMaker) (*Local, error) {
	if n <= 1 {
		err := errors.Errorf("The number of peers should be positive, but got %v", n)
		return nil, err
	}

	l := new(Local)
	l.n = n
	l.network = rpccore.NewChanNetwork(rpcTimeout)
	l.network.SetDelayGenerator(l.delayGenerator)
	l.rpcPeers = make(map[rpccore.NodeID]*rpccore.ChanNode, n)
	l.raftPeers = make(map[rpccore.NodeID]*raft.Peer, n)
	l.logger = logrus.New()
	l.logger.Out = os.Stdout
	l.logger.SetLevel(logrus.DebugLevel)
	l.logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.999999999Z07:00",
	})
	l.pstorages = make(map[rpccore.NodeID]pstorage.PersistentStorage, n)
	l.smMaker = smMaker
	l.snapshotThreshold = snapshotThreshold

	// create rpc nodes
	for i := 0; i < n; i++ {
		node, err := l.network.NewNode(rpccore.NodeID(strconv.Itoa(i)))
		if err != nil {
			err = errors.Errorf("Failed to allocate a new node with node id %v", node.NodeID())
			return nil, err
		}
		l.rpcPeers[node.NodeID()] = node
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
		l.pstorages[id] = pstorage.NewMemoryBasedPersistentStorage()
		// move current nodeID to the last of the nodeIDs slice
		i, j := idxMap[id], n-1
		nodeIDs[i], nodeIDs[j] = nodeIDs[j], nodeIDs[i]
		idxMap[nodeIDs[i]] = i
		idxMap[id] = j

		var err error
		l.raftPeers[id], err = raft.NewPeer(node, nodeIDs[:n-1], l.logger.WithFields(logrus.Fields{
			"nodeID": node.NodeID(),
		}), smMaker(), l.pstorages[id], testTimingFactor, snapshotThreshold)
		if err != nil {
			return nil, err
		}
	}

	l.offlineNodes = make(map[rpccore.NodeID]bool, n)
	l.nodePartition = make(map[rpccore.NodeID]int, n)
	for k := range l.rpcPeers {
		l.offlineNodes[k] = false
		l.nodePartition[k] = 0
	}
	l.oneWayLatencyMin = 10 * time.Millisecond
	l.oneWayLatencyMax = 40 * time.Millisecond
	l.packetLossRate = 0.01

	cNode, err := l.network.NewNode("client")
	if err != nil {
		return nil, err
	}

	l.clientCore = client.NewClientCore("client", nodeIDs, cNode, Log)

	return l, nil
}

func (l *Local) StopAll() {
	for _, peer := range l.raftPeers {
		peer.ShutDown()
	}
	l.network.Shutdown()
}

func (l *Local) RequestRaw(cmd interface{}) chan bool {

	// use timeout here rather inside to handle
	// 1. leadership change after for loop, waiting for new leader got elected
	// 2. request timeout, auto retry
	c := make(chan bool)

	go func() {
		for {
			var leader *raft.Peer = nil
			for _, p := range l.raftPeers {
				if p.GetState() == raft.Leader {
					if leader == nil || leader.GetTerm() < p.GetTerm() {
						t := p
						leader = t
					}
				}
			}

			if leader != nil && leader.HandleClientRequest(cmd) {
				c <- true
				return
			}
		}
	}()

	return c
}

func (l *Local) RequestSync(cmd interface{}) bool {
	c := l.RequestRaw(cmd)
	select {
	case done := <-c:
		return done
	case <-time.After(clientRequestTimeout):
		Log.Warn("Client request timeout")
		return false
	}
}

func (l *Local) RequestActionSync(act sm.TSMAction) error {
	ok, msg := l.clientCore.ExecuteActionRequest(act)
	if ok {
		return nil
	}

	return errors.New(msg)
}

func (l *Local) RequestQuerySync(key string) (interface{}, error) {
	return l.clientCore.ExecuteQueryRequest(sm.NewTSMDataQuery(key))
}

func (l *Local) ShutDownPeer(id rpccore.NodeID) {
	l.raftPeers[id].ShutDown()
}

func (l *Local) StartPeer(id rpccore.NodeID) {
	l.raftPeers[id].Start()
}

func (l *Local) Wait(sec int) {
	if sec <= 0 {
		Log.Warnf("Seconds to wait should be positive integer, not %v", sec)
		return
	}

	Log.Infof("Sleeping for %v second(s)", sec)
	time.Sleep(time.Duration(sec) * time.Second)
}

func (l *Local) getAllNodeIDs() []rpccore.NodeID {
	rst := make([]rpccore.NodeID, len(l.rpcPeers))
	i := 0
	for _, rpcNode := range l.rpcPeers {
		rst[i] = rpcNode.NodeID()
		i++
	}
	return rst
}

func (l *Local) GetAllNodeIDs() []rpccore.NodeID {
	return l.getAllNodeIDs()
}

func (l *Local) GetNetWork() *rpccore.ChanNetwork {
	return l.network
}

func (l *Local) PrintAllNodeInfo() {
	m := l.getAllNodeInfo()
	for k, v := range m {
		Log.Infof("%v:\n%v", k, v)
	}
}

func (l *Local) ResetPeer(nodeID rpccore.NodeID) error {
	peer := l.raftPeers[nodeID]
	peer.ShutDown()
	nodeIDs := make([]rpccore.NodeID, 0, len(l.raftPeers)-1)
	for k := range l.raftPeers {
		if k != nodeID {
			nodeIDs = append(nodeIDs, k)
		}
	}
	var err error
	l.raftPeers[nodeID], err = raft.NewPeer(l.rpcPeers[nodeID], nodeIDs,
		l.logger.WithFields(logrus.Fields{
			"nodeID": nodeID,
		}), l.smMaker(), l.pstorages[nodeID], testTimingFactor, l.snapshotThreshold)
	return err
}

func (l *Local) getAllNodeInfo() map[rpccore.NodeID]string {
	m := make(map[rpccore.NodeID]string)
	for nodeID, peer := range l.raftPeers {
		m[nodeID] = peer.GetInfo()
	}
	return m
}

func (l *Local) getAllNodeLogs() map[rpccore.NodeID][]raft.LogEntry {
	m := make(map[rpccore.NodeID][]raft.LogEntry)
	for nodeID, peer := range l.raftPeers {
		m[nodeID] = peer.GetRestLog()
	}
	return m
}

func (l *Local) getAllNodeSnapshots() map[rpccore.NodeID]*raft.Snapshot {
	m := make(map[rpccore.NodeID]*raft.Snapshot)
	for nodeID, peer := range l.raftPeers {
		m[nodeID] = peer.GetRecentSnapshot()
	}
	return m
}

func (l *Local) AgreeOnLeader() (*rpccore.NodeID, error) {
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

func (l *Local) AgreeOnTerm() (int, error) {
	term := -1
	for _, peer := range l.raftPeers {
		if term == -1 {
			term = peer.GetTerm()
		} else {
			if term != peer.GetTerm() {
				return 0, errors.Errorf("Failed to agree on term.\n\n%v\n",
					l.getAllNodeInfo())
			}
		}
	}
	return term, nil
}

func (l *Local) AgreeOnLogEntriesIfSnapshotEqual() (bool, error) {
	logEntriesMap := l.getAllNodeLogs()
	snapshotMap := l.getAllNodeSnapshots()
	for peer1, logEntry1 := range logEntriesMap {
		for peer2, logEntry2 := range logEntriesMap {
			if peer1 != peer2 {
				equal, err := raft.SnapshotEqual(snapshotMap[peer1], snapshotMap[peer2])
				if err != nil {
					return false, err
				}
				if equal {
					_, err := l.AgreeOnTwoLogEntries(logEntry1, logEntry2)
					if err != nil {
						return false, errors.Errorf("node %v and %v not agree on log entries. \n%v", peer1, peer2, l.getAllNodeInfo())
					}
				}
			}
		}
	}
	return true, nil
}

func (l *Local) IdenticalLogEntries() error {
	var peerLogs1 []raft.LogEntry
	for _, peer := range l.raftPeers {
		peerLogs1 = peer.GetRestLog()
		break
	}
	for _, peer := range l.raftPeers {
		peerLogs2 := peer.GetRestLog()
		if len(peerLogs1) != len(peerLogs2) {
			return errors.Errorf("not identical log entries.\n\n%v\n",
				l.getAllNodeInfo())
		}
		for j, peerLog := range peerLogs1 {
			if peerLog != peerLogs2[j] {
				return errors.Errorf("not identical log entries.\n\n%v\n",
					l.getAllNodeInfo())
			}
		}
		peerLogs1 = peerLogs2
	}
	return nil
}

func (l *Local) AgreeOnTwoLogEntries(logEntry1, logEntry2 []raft.LogEntry) (bool, error) {
	cmdIdentical := true
	for i := 0; i < utils.Min(len(logEntry1), len(logEntry2)); i++ {
		if logEntry1[i].Cmd != logEntry2[i].Cmd {
			cmdIdentical = false
		}
		if logEntry1[i].Term == logEntry2[i].Term {
			if !cmdIdentical {
				return false, errors.Errorf("not agree on log entries .\n\n%v\n", l.getAllNodeInfo())
			}
		}
	}
	return true, nil
}

func (l *Local) AgreeOnLogEntries() error {
	logEntriesMap := l.getAllNodeLogs()
	for peer1, logEntry1 := range logEntriesMap {
		for peer2, logEntry2 := range logEntriesMap {
			if peer1 != peer2 {
				_, err := l.AgreeOnTwoLogEntries(logEntry1, logEntry2)
				if err != nil {
					return errors.Errorf("node %v and %v not agree on log entries. \n%v", peer1, peer2, l.getAllNodeInfo())
				}
			}
		}
	}
	return nil
}

func (l *Local) AgreeOnSnapshot() (int, int, error) {
	var ss *raft.Snapshot
	for _, peer := range l.raftPeers {
		if peer.GetRecentSnapshot() == nil {
			return -1, -1, errors.Errorf("Node %v does not have snapshot. \n", peer.GetNodeID())
		}
		if ss == nil {
			ss = peer.GetRecentSnapshot()
		} else {
			ss2 := peer.GetRecentSnapshot()
			isEqual, err := raft.SnapshotEqual(ss, ss2)
			if err != nil {
				fmt.Printf("fail to compare snapshots, %v\n", err)
				return -1, -1, err
			}
			if !isEqual {
				return -1, -1, errors.Errorf("Node %v has different snapshot\n s1:\n%v\nlast term:%v\nlast index:%v, \n s2:\n%v\nlast term:%v\nlast index:%v",
					peer.GetNodeID(),
					sm.TSMToStringHuman(ss.StateMachineSnapshot),
					ss.LastIncludedTerm,
					ss.LastIncludedIndex,
					sm.TSMToStringHuman(ss2.StateMachineSnapshot),
					ss2.LastIncludedTerm,
					ss2.LastIncludedIndex)
			}
		}
	}
	return ss.LastIncludedIndex, ss.LastIncludedTerm, nil
}

func (l *Local) AgreeOnStateMachine() ([]byte, error) {
	var ss []byte
	for _, peer := range l.raftPeers {
		if ss == nil {
			ss = peer.TakeStateMachineSnapshot()
		} else {
			ss2 := peer.TakeStateMachineSnapshot()
			isEqual, err := sm.TSMIsSnapshotEqual(ss, ss2)
			if err != nil {
				fmt.Printf("fail to compare snapshots, %v\n", err)
				return nil, err
			}
			if !isEqual {
				return nil, errors.Errorf("Node %v has different StateMachine, ss: %v, ss2: %v\n",
					peer.GetNodeID(),
					sm.TSMToStringHuman(ss),
					sm.TSMToStringHuman(ss2))
			}
		}
	}
	return ss, nil
}

func (l *Local) SetNetworkReliability(oneWayLatencyMin, oneWayLatencyMax time.Duration, packetLossRate float64) {
	l.netLock.Lock()
	defer l.netLock.Unlock()
	l.oneWayLatencyMin = oneWayLatencyMin
	l.oneWayLatencyMax = oneWayLatencyMax
	l.packetLossRate = packetLossRate
}

func (l *Local) SetNodeNetworkStatus(nodeID rpccore.NodeID, online bool) {
	l.netLock.Lock()
	defer l.netLock.Unlock()
	l.offlineNodes[nodeID] = !online
}

func (l *Local) SetNetworkPartition(pMap map[rpccore.NodeID]int) {
	Log.Info("Set network partition...")
	l.netLock.Lock()
	defer l.netLock.Unlock()
	for k := range l.rpcPeers {
		l.nodePartition[k] = pMap[k]
	}
}

func (l *Local) GetActionBuilder() *sm.TSMActionBuilder {
	return l.clientCore.ActBuilder
}

func (l *Local) GetPeer(nodeID rpccore.NodeID) *raft.Peer {
	return l.raftPeers[nodeID]
}
