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
	time.Sleep(10 * time.Second)

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
	time.Sleep(10 * time.Second)
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
	time.Sleep(10 * time.Second)
	leader1, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	fmt.Printf("First leader:%v\n", *leader1)
	if *leader1 == "3" || *leader1 == "4" {
		return errors.Errorf("Leader elected in wrong partition, leader:%v", leader1)
	}

	// recovery from partition
	pmap = map[rpccore.NodeID]int{
		"0": 0,
		"1": 0,
		"2": 0,
		"3": 0,
		"4": 0,
	}
	sl.SetNetworkPartition(pmap)
	time.Sleep(10 * time.Second)
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
	return

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
