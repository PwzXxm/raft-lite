package functests

import (
	"fmt"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/pkg/errors"
)

func caseEvenPartitionLeaderElection() (err error) {
	sl := simulation.RunLocally(6)
	defer sl.StopAll()

	// make partition [0, 1, 2] [3, 4, 5]
	pmap := map[rpccore.NodeID]int{
		"0": 0,
		"1": 0,
		"2": 0,
		"3": 1,
		"4": 1,
		"5": 1,
	}
	sl.SetNetworkPartition(pmap)
	time.Sleep(5 * time.Second)

	// no leader should be elected
	leader1, err := sl.AgreeOnLeader()
	if err == nil {
		return errors.Errorf("Leader is elected, leader:%v", *leader1)
	}

	// recovery from partition
	pmap = map[rpccore.NodeID]int{
		"0": 0,
		"1": 0,
		"2": 0,
		"3": 0,
		"4": 0,
		"5": 0,
	}
	sl.SetNetworkPartition(pmap)
	time.Sleep(20 * time.Second)
	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Printf("Recovery from partition, leader:%v, term:%v\n", *leader2, term2)
	return
}

func caseSkewedPartitionLeaderElection() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	// make partition [0, 1, 2] [3, 4]
	pmap := map[rpccore.NodeID]int{
		"0": 0,
		"1": 0,
		"2": 0,
		"3": 1,
		"4": 1,
	}
	sl.SetNetworkPartition(pmap)

	// leader should be elected in [0, 1, 2]
	time.Sleep(5 * time.Second)
	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	fmt.Printf("First leader:%v\n", *leader1)
	if *leader1 == "3" || *leader1 == "4" {
		return errors.Errorf("Leader elected in wrong partition, leader:%v", leader1)
	}

	// send request otherwise the leader might be the same as before
	rst := sl.RequestSync(1)
	fmt.Printf("Request sent: %v\n", rst)
	time.Sleep(5 * time.Second)

	// recovery from partition
	pmap = map[rpccore.NodeID]int{
		"0": 0,
		"1": 0,
		"2": 0,
		"3": 0,
		"4": 0,
	}
	sl.SetNetworkPartition(pmap)
	time.Sleep(5 * time.Second)
	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Printf("Recovery from partition, leader:%v, term:%v\n", *leader2, term2)
	return sl.IdenticalLogEntries()
}

func caseRecoverLeaderElection() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	// process inital election normally
	time.Sleep(5 * time.Second)
	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term1, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Printf("Initial election finished, leader:%v, term:%v\n", *leader1, term1)

	// mess up the network
	fmt.Println("Mess up the network for 10 secs...")
	sl.SetNetworkReliability(time.Duration(6*time.Second), time.Duration(6*time.Second), 0.0)
	time.Sleep(10 * time.Second)

	// process second election normally
	fmt.Println("Network back to normal...")
	sl.SetNetworkReliability(time.Duration(0*time.Second), time.Duration(0*time.Second), 0.0)
	time.Sleep(10 * time.Second)
	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Printf("Second election finished, leader:%v, term:%v\n", *leader2, term2)

	return nil
}

func caseInitLeaderElection() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	// after initial election
	time.Sleep(5 * time.Second)
	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term1, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}

	// at least 1 as the initial election happens
	if term1 < 1 {
		return errors.Errorf("Term should be at least 1. t1: %v", term1)
	}

	fmt.Println("First check passed, leader is selected.")

	// after a while, since the network is fine, it should be the same
	time.Sleep(20 * time.Second)
	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}

	if *leader2 != *leader1 || term2 != term1 {
		return errors.Errorf("Leader and/or term changed. l1:%v, l2:%v, t1:%v, t2:%v",
			*leader1, *leader2, term1, term2)
	}

	return nil
}

func caseAppendLogEntry() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	// after initial election
	time.Sleep(5 * time.Second)

	// continually send client request with 1 second interval
	for i := 0; i < 5; i++ {
		sl.RequestSync(i)
		time.Sleep(1 * time.Second)
	}

	// after 5 seconds, the peers should agree on log entries
	time.Sleep(5 * time.Second)
	err = sl.IdenticalLogEntries()
	if err != nil {
		return
	}
	fmt.Print("Agree on log entry test passed\n")
	return nil
}

func caseLeaderOffline() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	// process inital election normally
	time.Sleep(5 * time.Second)
	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term1, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Printf("Initial election finished, leader: %v, term: %v\n", *leader1, term1)

	// set the leader to be offline
	sl.SetNodeNetworkStatus(*leader1, false)
	fmt.Println("Leader goes offline.")

	time.Sleep(10 * time.Second)
	// send request otherwise the leader might be the same as before
	rst := sl.RequestSync(1)
	fmt.Printf("Request sent: %v\n", rst)

	time.Sleep(5 * time.Second)
	sl.SetNodeNetworkStatus(*leader1, true)
	fmt.Println("Leader goes online.")

	time.Sleep(10 * time.Second)
	fmt.Println("Start the check.")

	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}

	if *leader2 == *leader1 || term2 == term1 {
		return errors.Errorf("Leader and term need to be different. l1: %v, l2: %v, t1: %v, t2: %v",
			*leader1, *leader2, term1, term2)
	}

	return nil
}

func caseHighPacketLossRate() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	// process inital election normally
	time.Sleep(5 * time.Second)
	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term1, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Printf("Initial election finished, leader: %v, term: %v\n", *leader1, term1)

	fmt.Println("High packet loss rate mode...")
	sl.SetNetworkReliability(10*time.Millisecond, 40*time.Millisecond, 0.5)

	time.Sleep(10 * time.Second)

	fmt.Println("Network back to normal...")
	sl.SetNetworkReliability(10*time.Millisecond, 40*time.Millisecond, 0.0)

	time.Sleep(10 * time.Second)
	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Printf("Second election finished, leader: %v, term: %v\n", *leader2, term2)

	return nil
}

func caseAgreeOnLogEntryWithPartitionAndLeaderReselection() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	time.Sleep(5 * time.Second)

	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}

	fmt.Printf("first leader selected: %v\n", *leader1)

	for i := 0; i < 5; i++ {
		sl.RequestSync(i)
		time.Sleep(500 * time.Millisecond)
	}

	// make partition [0, 1, 2] and [3, 4]
	pmap := map[rpccore.NodeID]int{
		"0": 0,
		"1": 0,
		"2": 0,
		"3": 1,
		"4": 1,
	}
	sl.SetNetworkPartition(pmap)
	time.Sleep(3 * time.Second)

	// only node in the leader's partition should append entries
	for i := 5; i < 10; i++ {
		sl.RequestSync(i)
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(3 * time.Second)

	// should agree on log entries
	err = sl.AgreeOnLogEntries()
	if err != nil {
		return
	}
	fmt.Println("first logs check finished, agree on log entries")

	// set the followers into same partition and leader to the other partition
	for nodeID := range pmap {
		pmap[nodeID] = 0
	}
	pmap[*leader1] = 1
	print(pmap)
	sl.SetNetworkPartition(pmap)
	time.Sleep(3 * time.Second)

	//leader should be reselected, now append another 5 entries
	for i := 10; i < 15; i++ {
		sl.RequestSync(i)
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(5 * time.Second)

	// should always agree on log entries
	err = sl.AgreeOnLogEntries()
	if err != nil {
		return
	}
	fmt.Println("Second logs check finished, agree on log entries")

	// all nodes back to normal
	for nodeID := range pmap {
		pmap[nodeID] = 0
	}
	sl.SetNetworkPartition(pmap)

	err = sl.AgreeOnLogEntries()
	if err != nil {
		return
	}
	fmt.Println("Third logs check finished, agree on log entries")
	return nil

}

func caseLeaderInOtherPartition() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	time.Sleep(5 * time.Second)

	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	fmt.Printf("first leader selected: %v\n", *leader1)

	for i := 0; i < 5; i++ {
		sl.RequestSync(i)
		time.Sleep(500 * time.Millisecond)
	}

	// time.Sleep(3 * time.Second)
	err = sl.AgreeOnLogEntries()
	if err != nil {
		return
	}

	sl.SetNodeNetworkStatus(*leader1, false)
	time.Sleep(5 * time.Second)

	for i := 5; i < 10; i++ {
		sl.RequestSync(i)
		time.Sleep(500 * time.Millisecond)
	}

	err = sl.AgreeOnLogEntries()
	if err != nil {
		return
	}

	fmt.Println("Finished")
	return nil
}

func caseRestartPeer() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	sl.SetNetworkReliability(10*time.Millisecond, 40*time.Millisecond, 0.02)
	time.Sleep(5 * time.Second)

	fmt.Print("Start sending request.\n")

	for i := 0; i < 5; i++ {
		sl.RequestRaw(i)
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	sl.ShutDownPeer("2")
	fmt.Print("Shutdown peer 2.\n")

	for i := 5; i < 10; i++ {
		sl.RequestRaw(i)
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(5 * time.Second)

	err = sl.AgreeOnLogEntries()
	if err != nil {
		return
	}
	err = sl.ResetPeer("2")
	if err != nil {
		return
	}
	sl.StartPeer("2")
	fmt.Print("Restart peer 2.\n")

	time.Sleep(2 * time.Second)
	for i := 10; i < 15; i++ {
		sl.RequestRaw(i)
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(10 * time.Second)
	err = sl.IdenticalLogEntries()
	return
}

func caseCandidateTimeout() error {
	sl := simulation.SetupLocally(5)
	defer sl.StopAll()
	sl.SetNetworkReliability(10*time.Millisecond, 40*time.Millisecond, 1)
	sl.StartAll()

	time.Sleep(4 * time.Second)
	sl.SetNetworkReliability(10*time.Millisecond, 40*time.Millisecond, 0)
	time.Sleep(4 * time.Second)

	_, err := sl.AgreeOnLeader()
	return err
}

func caseSaveToSnapshot() error {
	sl := simulation.RunLocallyOptional(5, 5, func() sm.StateMachine { return sm.NewTransactionStateMachine() })
	defer sl.StopAll()
	actioinBuilder := sm.NewTSMActionBuilder("client")
	// leader election
	time.Sleep(2 * time.Second)
	leader, err := sl.AgreeOnLeader()
	if err != nil {
		return err
	}

	// make requests, check each node has the same snapshot
	fmt.Print("Start sending request.\n")
	sl.RequestSync(actioinBuilder.TSMActionSetValue("key_a", 0))
	for i := 0; i < 20; i++ {
		sl.RequestSync(actioinBuilder.TSMActionIncrValue("key_a", 10))
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	li, lt, err := sl.AgreeOnSnapshot()
	fmt.Printf("LastIdx: %v LastTerm: %v\n", li, lt)
	if err != nil {
		return err
	}

	// isolate node i who is not leader
	var isolater rpccore.NodeID
	for _, p := range sl.GetAllNodeIDs() {
		if p != *leader {
			isolater = p
		}
	}
	fmt.Printf("ShutDown Peer %v\n", isolater)
	sl.ShutDownPeer(isolater)
	fmt.Print("Start sending request.\n")
	for i := 0; i < 20; i++ {
		sl.RequestRaw(actioinBuilder.TSMActionIncrValue("key_a", 10))
		time.Sleep(150 * time.Millisecond)
	}
	fmt.Printf("Restart Peer %v\n", isolater)
	sl.StartPeer(isolater)
	time.Sleep(4 * time.Second)

	li, lt, err = sl.AgreeOnSnapshot()
	fmt.Printf("LastIdx: %v LastTerm: %v\n %v", li, lt, err)
	return err
}

func caseSaveToPersistentStorage() (err error) {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	sl.SetNetworkReliability(10*time.Millisecond, 40*time.Millisecond, 0)

	// leader election
	time.Sleep(5 * time.Second)
	leader, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}

	fmt.Print("Start sending request.\n")

	for i := 0; i < 5; i++ {
		sl.RequestRaw(i)
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)

	// isolate node i who is not leader
	var isolater rpccore.NodeID
	for _, p := range sl.GetAllNodeIDs() {
		if p != *leader {
			isolater = p
		}
	}

	data1 := sl.GetPersistentStorage(isolater)

	// shut down particular peer
	sl.ShutDownPeer(isolater)
	fmt.Printf("ShutDown Peer %v\n", isolater)

	for i := 5; i < 10; i++ {
		sl.RequestRaw(i)
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(5 * time.Second)

	// recover particular peer
	sl.StartPeer(isolater)
	fmt.Printf("Restart Peer %v\n", isolater)

	// check persistent storage
	data2 := sl.GetPersistentStorage(isolater)

	err = sl.AgreeOnPersistentStorage(data1, data2)
	if err != nil {
		return
	}

	return
}
