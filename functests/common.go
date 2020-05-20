package functests

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type testCase struct {
	name   string
	action func() error
}

var testCases = []testCase{
	{
		name:   "initial leader election",
		action: caseInitLeaderElection,
	},
	{
		name:   "recovery leader election",
		action: caseRecoverLeaderElection,
	},
	{
		name:   "Append log entries",
		action: caseAppendLogEntry,
	},
	{
		name:   "leader offline",
		action: caseLeaderOffline,
	},
	{
		name:   "high packet loss rate",
		action: caseHighPacketLossRate,
	},
	{
		name:   "Even partition leader election",
		action: caseEvenPartitionLeaderElection,
	},
	{
		name:   "Skewed partition leader election",
		action: caseSkewedPartitionLeaderElection,
	},
	{
		name:   "agree on log entry with partition and leader reselection",
		action: caseAgreeOnLogEntryWithPartitionAndLeaderReselection,
	},
	{
		name:   "leader in other partition",
		action: caseLeaderInOtherPartition,
	},
	{
		name:   "restart peer",
		action: caseRestartPeer,
	},
	{
		name:   "candidate timeout",
		action: caseCandidateTimeout,
	},
	{
		name:   "save to snapshots",
		action: caseSaveToSnapshot,
	},
	{
		name:   "save to persistent storage",
		action: caseSaveToPersistentStorage,
	},
	{
		name:   "check eventual consistency",
		action: caseCheckEventualConsistency,
	},
}

func List() {
	for i, c := range testCases {
		fmt.Printf("%2d: %v\n", i+1, c.name)
	}
}

func Count() {
	fmt.Printf("%v\n", len(testCases))
}

func Run(n int) error {
	if n <= 0 || n > len(testCases) {
		return errors.New("Please provide a valid test case id.")
	}
	c := testCases[n-1]
	fmt.Printf("--------------------\n")
	fmt.Printf("running test %2d: %v\n", n, c.name)
	fmt.Printf("--------------------\n")
	t := time.Now()
	err := c.action()
	fmt.Printf("\n--------------------\n")
	if err == nil {
		fmt.Printf("SUCCESS\n")
	} else {
		fmt.Printf("FAIL\n")
	}
	fmt.Printf("Time used: %.2fs\n", time.Since(t).Seconds())
	fmt.Printf("--------------------\n")
	return err
}
