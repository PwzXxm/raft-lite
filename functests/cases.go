package functests

import (
	"fmt"
	"time"

	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/pkg/errors"
)

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
