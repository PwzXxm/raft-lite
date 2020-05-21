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

type local struct {
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

var log *logrus.Logger

func init() {
	log = logrus.New()
	log.Out = os.Stdout
}

func RunLocally(n int) *local {
	return RunLocallyOptional(n, defaultSnapshotThreshold, func() sm.StateMachine { return sm.NewEmptyStateMachine() })
}

func RunLocallyOptional(n int, snapshotThreshold int, smMaker stateMachineMaker) *local {
	log.Info("Starting simulation locally ...")

	l, err := newLocalOptional(n, snapshotThreshold, smMaker)
	if err != nil {
		log.Panicln(err)
	}

	for _, node := range l.raftPeers {
		node.Start()
	}

	return l
}

func SetupLocally(n int) *local {
	log.Info("Setting up simulation locally ...")

	l, err := newLocal(n)
	if err != nil {
		log.Panicln(err)
	}

	return l
}

func (l *local) StartAll() {
	for _, node := range l.raftPeers {
		node.Start()
	}
}

// Current limitation:
// 	1. latency is uniform distribution
// 	2. don't support one way connection lost (packet lost is one way)
// 	2. all nodes share the same latency and packet lost rate
func (l *local) delayGenerator(source, target rpccore.NodeID) time.Duration {
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

func newLocal(n int) (*local, error) {
	return newLocalOptional(n, defaultSnapshotThreshold, func() sm.StateMachine { return sm.NewEmptyStateMachine() })
}

func newLocalOptional(n int, snapshotThreshold int, smMaker stateMachineMaker) (*local, error) {
	if n <= 1 {
		err := errors.Errorf("The number of peers should be positive, but got %v", n)
		return nil, err
	}

	l := new(local)
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

	l.clientCore = client.NewClientCore("client", nodeIDs, cNode, log)

	return l, nil
}

func (l *local) StopAll() {
	for _, peer := range l.raftPeers {
		peer.ShutDown()
	}
	l.network.Shutdown()
}

func (l *local) RequestRaw(cmd interface{}) chan bool {

	// use timeout here rather inside to handle
	//  1. leadership change after for loop, waiting for new leader got elected
	//  2. request timeout, auto retry
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

func (l *local) RequestSync(cmd interface{}) bool {
	c := l.RequestRaw(cmd)
	select {
	case done := <-c:
		return done
	case <-time.After(clientRequestTimeout):
		log.Warn("Client request timeout")
		return false
	}
}

func (l *local) RequestActionSync(act sm.TSMAction) error {
	ok, msg := client.ExecuteActionRequest(&l.clientCore, act)
	if ok {
		return nil
	}

	return errors.New(msg)
}

func (l *local) RequestQuerySync(key string) (interface{}, error) {
	return client.ExecuteQueryRequest(&l.clientCore, sm.NewTSMDataQuery(key))
}

func (l *local) ShutDownPeer(id rpccore.NodeID) {
	l.raftPeers[id].ShutDown()
}

func (l *local) StartPeer(id rpccore.NodeID) {
	l.raftPeers[id].Start()
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

func (l *local) GetAllNodeIDs() []rpccore.NodeID {
	return l.getAllNodeIDs()
}

func (l *local) PrintAllNodeInfo() {
	m := l.getAllNodeInfo()
	for k, v := range m {
		log.Infof("%v:\n%v", k, v)
	}
}

func (l *local) ResetPeer(nodeID rpccore.NodeID) error {
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

func (l *local) getAllNodeInfo() map[rpccore.NodeID]map[string]string {
	m := make(map[rpccore.NodeID]map[string]string)
	for nodeID, peer := range l.raftPeers {
		m[nodeID] = peer.GetInfo()
	}
	return m
}

func (l *local) getAllNodeLogs() map[rpccore.NodeID][]raft.LogEntry {
	m := make(map[rpccore.NodeID][]raft.LogEntry)
	for nodeID, peer := range l.raftPeers {
		m[nodeID] = peer.GetRestLog()
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
				return 0, errors.Errorf("Failed to agree on term.\n\n%v\n",
					l.getAllNodeInfo())
			}
		}
	}
	return term, nil
}

func (l *local) IdenticalLogEntries() error {
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

func (l *local) AgreeOnTwoLogEntries(logEntry1, logEntry2 []raft.LogEntry) (bool, error) {
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

func (l *local) AgreeOnLogEntries() error {
	logEntriesMap := l.getAllNodeLogs()
	for peer1, logEntry1 := range logEntriesMap {
		for peer2, logEntry2 := range logEntriesMap {
			if peer1 != peer2 {
				_, err := l.AgreeOnTwoLogEntries(logEntry1, logEntry2)
				if err != nil {
					return errors.Errorf("node %v and %v not agree on log entries. \n", peer1, peer2)
				}
			}
		}
	}
	return nil
}

func (l *local) AgreeOnSnapshot() (int, int, error) {
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

func (l *local) AgreeOnStateMachine() ([]byte, error) {
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

func (l *local) SetNetworkReliability(oneWayLatencyMin, oneWayLatencyMax time.Duration, packetLossRate float64) {
	l.netLock.Lock()
	defer l.netLock.Unlock()
	l.oneWayLatencyMin = oneWayLatencyMin
	l.oneWayLatencyMax = oneWayLatencyMax
	l.packetLossRate = packetLossRate
}

func (l *local) SetNodeNetworkStatus(nodeID rpccore.NodeID, online bool) {
	l.netLock.Lock()
	defer l.netLock.Unlock()
	l.offlineNodes[nodeID] = !online
}

func (l *local) SetNetworkPartition(pMap map[rpccore.NodeID]int) {
	log.Info("Set network partition...")
	l.netLock.Lock()
	defer l.netLock.Unlock()
	for k := range l.rpcPeers {
		l.nodePartition[k] = pMap[k]
	}
}

func (l *local) GetActionBuilder() *sm.TSMActionBuilder {
	return l.clientCore.ActBuilder
}

func (l *local) GetPeer(nodeID rpccore.NodeID) *raft.Peer {
	return l.raftPeers[nodeID]
}
