package functests

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/PwzXxm/raft-lite/utils"
)

const (
	networkReliability  = "nr"
	networkPartition    = "np"
	nodeStatusChange    = "nsc"
	networkBackToNormal = "nb"
	clientRequest       = "cr"
)

const (
	nrWeight    = 1
	npWeight    = 1
	nscWeight   = 1
	nbWeight    = 1
	crWeight    = 5
	totalWeight = nrWeight + npWeight + nscWeight + crWeight
)

func getRandomEvent() string {
	weightList := []int{nrWeight, npWeight, nscWeight, nbWeight, crWeight}
	eventList := []string{networkReliability, networkPartition, nodeStatusChange, networkBackToNormal, clientRequest}
	rd := utils.Random(0, totalWeight)
	sum := 0
	for i, weight := range weightList {
		sum += weight
		if sum >= rd {
			return eventList[i]
		}
	}
	return ""
}

func getRandomAction() interface{} {
	return utils.Random(0, 100)
}

func complexTest() {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()
	nodeIDs := []rpccore.NodeID{"0", "1", "2", "3", "4"}
	// process inital election normally
	time.Sleep(5 * time.Second)
	//check the log entries each 10 seconds
	go func() {
		for i := 0; ; i++ {
			time.Sleep(10 * time.Second)
			if sl.AgreeOnLogEntries() == nil {
				fmt.Printf("Check %v passed, agree on log entries\n", i)
			} else {
				fmt.Printf("Check %v failed, nodes do not agree on log entries\n", i)
			}
		}
	}()
	for true {
		switch getRandomEvent() {
		case networkReliability:
			latencyMin := time.Duration(utils.Random(0, 1000))
			latencyMax := time.Duration(utils.Random(2000, 3000))
			packetLossRate := utils.RandomFloat(0, 0.4)
			sl.SetNetworkReliability(latencyMin, latencyMax, packetLossRate)
			fmt.Printf("Network one way latency Min: %v, Max: %v, packet loss rate: %.2f\n", latencyMin, latencyMax, packetLossRate)
			time.Sleep(time.Duration(5 * time.Second))
		case networkPartition:
			// randomly chose number of partitions and randomly allocates the peers
			partitions := utils.Random(1, 5)
			pmap := map[rpccore.NodeID]int{
				"0": rand.Intn(partitions),
				"1": rand.Intn(partitions),
				"2": rand.Intn(partitions),
				"3": rand.Intn(partitions),
				"4": rand.Intn(partitions),
			}
			sl.SetNetworkPartition(pmap)
			fmt.Println("Network partition becomes: ", pmap)
			time.Sleep(time.Duration(5 * time.Second))
		case nodeStatusChange:
			// each node has 20% probablity to be offline
			for _, nodeID := range nodeIDs {
				if rand.Float64() < 0.2 {
					sl.SetNodeNetworkStatus(nodeID, false)
					fmt.Print(nodeID, " is not working...\n")
				} else {
					sl.SetNodeNetworkStatus(nodeID, true)
					fmt.Print(nodeID, " is working...\n")
				}
			}
			time.Sleep(time.Duration(5 * time.Second))
		case networkBackToNormal:
			// all nodes are online, network has no latency and packet loss
			sl.SetNetworkReliability(0, 0, 0)
			for _, nodeID := range nodeIDs {
				sl.SetNodeNetworkStatus(nodeID, true)
			}
			time.Sleep(time.Duration(5 * time.Second))
		case clientRequest:
			sl.RequestSync(getRandomAction())
			//randomly sleep 0 to 1 second
			time.Sleep(time.Duration(utils.Random(0, 1000)))
		}
	}
}
