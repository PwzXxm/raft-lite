package functests

import (
	"fmt"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/simulation"
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
	rst := sl.Request(1)
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
	if *leader1 != *leader2 {
		return errors.Errorf("Leader changed after recovery. l1:%v, l2:%v, t:%v",
			*leader1, *leader2, term2)
	}
	return
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
		sl.Request(i)
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

func caseCheckVoteCount() (err error) {
	n := 6
	sl := simulation.RunLocally(n)
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

	if leader1 == nil {
		return errors.Errorf("No leader is selected.")
	}
	// at least 1 as the initial election happens
	if term1 < 1 {
		return errors.Errorf("Term 1 should be at least 1. t1: %v", term1)
	}

	voteCount, err := sl.AgreeOnVoteCount()
	if voteCount != n {
		return errors.Errorf("Vote count changed from %v to %v.", n, voteCount)
	}
	fmt.Printf("Vote count check is succeed. v: %v\n", voteCount)
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
	rst := sl.Request(1)
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
	sl.SetNetworkReliability(time.Duration(0*time.Second), time.Duration(0*time.Second), 0.5)

	time.Sleep(10 * time.Second)

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
		sl.Request(i)
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
		sl.Request(i)
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
		sl.Request(i)
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
		sl.Request(i)
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
		sl.Request(i)
		time.Sleep(500 * time.Millisecond)
	}

	err = sl.AgreeOnLogEntries()
	if err != nil {
		return
	}

	fmt.Println("Finished")
	return nil
}
