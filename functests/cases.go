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
	fmt.Println("Initial election finished, leader:%v, term:%v", *leader1, term1)

	// mess up the network
	fmt.Println("Mess up the network for 10 secs...")
	sl.SetNetworkReliability(2, 4, 0.5)
	time.Sleep(10 * time.Second)

	// process second election normally
	fmt.Println("Network back to normal...")
	sl.SetNetworkReliability(0, 0, 0.0)
	time.Sleep(5 * time.Second)
	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}
	fmt.Println("Second election finished, leader:%v, term:%v", *leader2, term2)
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
