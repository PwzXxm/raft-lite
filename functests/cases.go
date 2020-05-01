package functests

import (
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

	// after a while, since the network is fine, it should be the same
	time.Sleep(60 * time.Second)
	leader2, err := sl.AgreeOnLeader()
	if err != nil {
		return
	}
	term2, err := sl.AgreeOnTerm()
	if err != nil {
		return
	}

	if leader2 != leader1 || term2 != term1 {
		return errors.Errorf("Leader amd/or term changed. l1:%v, l2:%v, t1:%v, t2:%v",
			leader1, leader2, term1, term2)
	}

	return nil
}
