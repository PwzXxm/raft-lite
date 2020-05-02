package functests

import (
	"fmt"
	"time"

	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/pkg/errors"
)

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
