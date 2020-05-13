package functests

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/PwzXxm/raft-lite/utils"
)

const (
	networkPacketLoss   = "net packet loss"
	networkPartition    = "net partition"
	nodeStatusChange    = "node net status change"
	networkBackToNormal = "net back to normal"
	clientRequest       = "client request"
)

const (
	nrWeight    = 1
	npWeight    = 1
	nscWeight   = 1
	nbWeight    = 1
	crWeight    = 5
	totalWeight = nrWeight + npWeight + nscWeight + nbWeight + crWeight
)

const (
	checkTotalCnt = "check total"
)

var weightList = [...]int{nrWeight, npWeight, nscWeight, nbWeight, crWeight}
var eventList = [...]string{networkPacketLoss, networkPartition, nodeStatusChange, networkBackToNormal, clientRequest}

func getRandomEvent() string {
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

func complexTest(ctx context.Context, wg *sync.WaitGroup, rst map[string]int) error {
	sl := simulation.RunLocally(5)
	defer sl.StopAll()
	nodeIDs := []rpccore.NodeID{"0", "1", "2", "3", "4"}
	// process inital election normally
	time.Sleep(5 * time.Second)

	//check the log entries each 10 seconds
	c := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; ; i++ {
			time.Sleep(10 * time.Second)
			err := sl.AgreeOnLogEntries()
			if err == nil {
				fmt.Printf("Check %v passed, agree on log entries\n", i)
			} else {
				c <- err
				return
			}

			select {
			case <-ctx.Done():
				fmt.Println("quitting log entry check go routine")
				return
			default:
			}

			rst[checkTotalCnt]++
		}
	}()

	for true {
		switch getRandomEvent() {
		case networkPacketLoss:
			latencyMin := time.Duration(utils.Random(0, 1000))
			latencyMax := time.Duration(utils.Random(2000, 3000))
			packetLossRate := utils.RandomFloat(0, 0.4)
			sl.SetNetworkReliability(latencyMin, latencyMax, packetLossRate)
			fmt.Printf("Network one way latency Min: %v, Max: %v, packet loss rate: %.2f\n", latencyMin, latencyMax, packetLossRate)
			time.Sleep(time.Duration(5 * time.Second))
			rst[networkPacketLoss]++
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
			rst[networkPartition]++
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
			rst[nodeStatusChange]++
		case networkBackToNormal:
			// all nodes are online, network has no latency and packet loss
			sl.SetNetworkReliability(0, 0, 0)
			for _, nodeID := range nodeIDs {
				sl.SetNodeNetworkStatus(nodeID, true)
			}
			time.Sleep(time.Duration(5 * time.Second))
			rst[networkBackToNormal]++
		case clientRequest:
			sl.RequestSync(getRandomAction())
			//randomly sleep 0 to 1 second
			time.Sleep(time.Duration(utils.Random(0, 1000)))
			rst[clientRequest]++
		}

		select {
		case err := <-c:
			return err
		case <-ctx.Done():
			fmt.Println("quitting complex running loop")
			return nil
		default:
		}
	}
	return nil
}

// RunComplex runs complex test cases with randomly selected actions
// it stops whenever error occurs or timeout
func RunComplex(duration int64) error {
	fmt.Printf("--------------------\n")
	fmt.Printf("running complex test\n")
	fmt.Printf("--------------------\n")

	rst := make(map[string]int)
	for _, e := range eventList {
		rst[e] = 0
	}
	rst[checkTotalCnt] = 0

	t := time.Now()
	var err error
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(duration)*time.Minute)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = complexTest(ctx, &wg, rst)
		fmt.Println("cancelling")
		cancel()
	}()
	wg.Wait()

	fmt.Printf("\n--------------------\n")
	if err == nil {
		fmt.Printf("SUCCESS\n")
	} else {
		fmt.Printf("FAIL\n")
	}
	fmt.Printf("Time used: %.2fs\n", time.Since(t).Seconds())
	fmt.Printf("--------------------\n")
	ks := make([]string, 0, len(rst))
	for k := range rst {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		if k != checkTotalCnt {
			fmt.Printf("| %-25s | %-25v |\n", k, rst[k])
		}
	}
	fmt.Println(checkTotalCnt, ": ", rst[checkTotalCnt])
	fmt.Printf("--------------------\n")
	return err
}
