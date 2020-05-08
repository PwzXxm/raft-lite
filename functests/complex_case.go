package functests

import (
	"math/rand"
	"fmt"
	"time"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/pkg/errors"
)

const (
	networkReliability   = "nr"
	networkPartition  = "np"
	nodeStatusChange  = "nsc"
	networkBackToNormal = "nb"
	clientRequest = "cr"
)

const (
	nrWeight   = 1
	npWeight    = 1
	nscWeight   = 1
	nbWeight = 1
	crWeight   = 1
	totalWeight = nrWeight + npWeight + nscWeight + crWeight
)

func getRandomEvent() string {
	weightList := []int{nrWeight + npWeight + nscWeight + nbWeight + crWeight}
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

func complexTest() {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()

	// process inital election normally
	for true{
		time.Sleep(5 * time.Second)
		switch getRandomEvent() {
			case networkReliability:
				latencyMin := time.Duration(utils.Random(0, 1000))
				latencyMax := time.Duration(utils.Random(2000, 3000))
				packetLossRate := utils.RandomFloat(0, 0.6)
				sl.SetNetworkReliability(latencyMin, latencyMax, packetLossRate)
				fmt.Printf("Network one way latency Min: %v, Max: %v, packet loss rate: %v\n", latencyMin, latencyMax, packetLossRate)
			case networkPartition:
				partitions := utils.Random(1,5)
				pmap := map[rpccore.NodeID]int{
					"0": rand.Intn(partitions),
					"1": rand.Intn(partitions),
					"2": rand.Intn(partitions),
					"3": rand.Intn(partitions),
					"4": rand.Intn(partitions),
				}
				sl.SetNetworkPartition(pmap)
				fmt.Print("Network partition becomes:", pmap)
			case nodeStatusChange:
				


		}
	}
}