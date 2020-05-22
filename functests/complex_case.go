/*
 * Project: raft-lite
 * ---------------------
 * Authors:
 *   Minjian Chen 813534
 *   Shijie Liu   813277
 *   Weizhi Xu    752454
 *   Wenqing Xue  813044
 *   Zijun Chen   813190
 */

package functests

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/PwzXxm/raft-lite/client"
	"github.com/PwzXxm/raft-lite/rpccore"
	"github.com/PwzXxm/raft-lite/simulation"
	"github.com/PwzXxm/raft-lite/sm"
	"github.com/PwzXxm/raft-lite/utils"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

const (
	networkPacketLoss   = "net packet loss"
	networkPartition    = "net partition"
	nodeStatusChange    = "node net status change"
	networkBackToNormal = "net back to normal"
	nodeReset           = "node reset"
	query               = "query"
	set                 = "set"
	move                = "move"
	incr                = "incr"
)

const (
	packetLossWeight   = 1
	partitionWeight    = 1
	statusChangeWeight = 1
	backToNormalWeight = 3
	nodeResetWeight    = 2

	eventTotalWeight = packetLossWeight + partitionWeight + statusChangeWeight + backToNormalWeight + nodeResetWeight
)

const (
	setWeight         = 1
	incrWeight        = 2
	moveWeight        = 2
	queryWeight       = 3
	actionTotalWeight = incrWeight + setWeight + moveWeight + queryWeight
)

const (
	checkTotalCnt = "check total"
)

var eventWeightList = [...]int{packetLossWeight, partitionWeight, statusChangeWeight, backToNormalWeight, nodeResetWeight}
var eventList = [...]string{networkPacketLoss, networkPartition, nodeStatusChange, networkBackToNormal, nodeReset}
var actionWeightList = [...]int{setWeight, incrWeight, moveWeight, queryWeight}
var actionList = [...]string{set, incr, move, query}
var yellow = color.New(color.FgYellow)
var green = color.New(color.FgGreen)
var red = color.New(color.FgRed)
var blue = color.New(color.FgBlue)

// get random event
func getRandomEvent() string {
	rd := utils.Random(0, eventTotalWeight)
	sum := 0
	for i, weight := range eventWeightList {
		sum += weight
		if sum >= rd {
			return eventList[i]
		}
	}
	return ""
}

func getRandomAction() string {
	rd := utils.Random(0, actionTotalWeight)
	sum := 0
	for i, weight := range actionWeightList {
		sum += weight
		if sum >= rd {
			return actionList[i]
		}
	}
	return ""
}

func randMapKey(m map[string]int) string {
	mapKeys := make([]string, 0, len(m))
	for key := range m {
		mapKeys = append(mapKeys, key)
	}
	return mapKeys[rand.Intn(len(mapKeys))]
}

func clientRandomlySendRequest(clientName string, sl *simulation.Local) {
	prefix := clientName + "-"
	node, err := sl.GetNetWork().NewNode(rpccore.NodeID(clientName))
	if err != nil {
		panic(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)
	clientCore := client.NewClientCore(clientName, sl.GetAllNodeIDs(), node, log)
	actionBuilder := clientCore.ActBuilder
	localStateMachine := make(map[string]int)
	for i := 0; i < 10; i++ {
		key := prefix + strconv.Itoa(utils.Random(0, 100))
		value := utils.Random(0, 1000)
		_, _ = yellow.Printf("%v request set action, key: %v, value: %v\n", clientName, key, value)
		action := actionBuilder.TSMActionSetValue(key, value)
		_, _ = clientCore.ExecuteActionRequest(action)
		localStateMachine[key] = value
	}
	for {
		// time.Sleep(time.Duration(utils.Random(0, 10000)))
		time.Sleep(time.Duration(1000 * time.Millisecond))
		switch getRandomAction() {
		case set:
			key := prefix + strconv.Itoa(utils.Random(0, 100))
			value := utils.Random(0, 1000)
			_, _ = yellow.Printf("%v request set action, key: %v, value: %v\n", clientName, key, value)
			action := actionBuilder.TSMActionSetValue(key, value)
			_, _ = clientCore.ExecuteActionRequest(action)
			localStateMachine[key] = value
		case incr:
			if len(localStateMachine) == 0 {
				continue
			}
			key := randMapKey(localStateMachine)
			value := utils.Random(-500, 1000)
			action := actionBuilder.TSMActionIncrValue(key, value)
			_, _ = yellow.Printf("%v request Incr action, key: %v, value: %v\n", clientName, key, value)
			_, _ = clientCore.ExecuteActionRequest(action)
			if localStateMachine[key]+value >= 0 {
				localStateMachine[key] += value
			}
		case move:
			if len(localStateMachine) == 0 {
				continue
			}
			key1 := randMapKey(localStateMachine)
			key2 := randMapKey(localStateMachine)
			value := utils.Random(-300, 300)
			action := actionBuilder.TSMActionMoveValue(key1, key2, value)
			_, _ = yellow.Printf("%v request move action, key1: %v, key2: %v, value: %v\n", clientName, key1, key2, value)
			_, _ = clientCore.ExecuteActionRequest(action)
			if localStateMachine[key1]-value >= 0 && localStateMachine[key2]+value >= 0 {
				localStateMachine[key1] -= value
				localStateMachine[key2] += value
			}
		case query:
			if len(localStateMachine) == 0 {
				continue
			}
			key := randMapKey(localStateMachine)
			correctResult := localStateMachine[key]
			_, _ = yellow.Printf("%v request query action, key: %v, result should be: %v\n", clientName, key, correctResult)
			result, _ := clientCore.ExecuteQueryRequest(sm.NewTSMDataQuery(key))
			if result == correctResult {
				_, _ = green.Printf("%v's Query result for %v correct, receive result: %v\n", clientName, key, result)
			} else if result == nil {
				simulation.Log.Panicf("%v's Query result for %v is nil.", clientName, key)
			} else {
				simulation.Log.Panicf("Error: %v's Query result for %v is %v\n", clientName, key, result)
			}
		}
	}
}

// complex test for checking algorithm implementation
func complexTest(ctx context.Context, wg *sync.WaitGroup, rst map[string]int) error {
	sl := simulation.RunLocallyOptional(5, 5, func() sm.StateMachine { return sm.NewTransactionStateMachine() })
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
			equal, err := sl.AgreeOnLogEntriesIfSnapshotEqual()
			if err == nil && equal {
				_, _ = green.Printf("Check %v passed, agree on log entries\n", i)
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

	go clientRandomlySendRequest("clientA", sl)
	go clientRandomlySendRequest("clientB", sl)
	go clientRandomlySendRequest("clientC", sl)
	go clientRandomlySendRequest("clientD", sl)
	go clientRandomlySendRequest("clientE", sl)

	nodeWorking := make(map[rpccore.NodeID]bool, 5)
	for _, nodeID := range nodeIDs {
		nodeWorking[nodeID] = true
	}

	for true {
		switch getRandomEvent() {
		case networkPacketLoss:
			// increace the network latency and package loss rate
			latencyMin := time.Duration(utils.Random(10, 20)) * time.Millisecond
			latencyMax := time.Duration(utils.Random(35, 45)) * time.Millisecond
			packetLossRate := utils.RandomFloat(0, 0.4)
			sl.SetNetworkReliability(latencyMin, latencyMax, packetLossRate)
			_, _ = blue.Printf("Network one way latency Min: %v, Max: %v, packet loss rate: %.2f\n", latencyMin, latencyMax, packetLossRate)
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
			_, _ = blue.Println("Network partition becomes: ", pmap)
			time.Sleep(time.Duration(5 * time.Second))
			rst[networkPartition]++
		case nodeStatusChange:
			// each node has 20% probablity to be offline
			_, _ = blue.Println("Node working status change")
			for _, nodeID := range nodeIDs {
				if rand.Float64() < 0.2 {
					sl.SetNodeNetworkStatus(nodeID, false)
					_, _ = blue.Println(nodeID, " is not working...")
				} else {
					sl.SetNodeNetworkStatus(nodeID, true)
					_, _ = blue.Println(nodeID, " is working...")
				}
			}
			time.Sleep(time.Duration(5 * time.Second))
			rst[nodeStatusChange]++
		case networkBackToNormal:
			// all nodes are online, network has no latency and packet loss
			_, _ = blue.Println("Network back to normal")
			sl.SetNetworkReliability(10*time.Millisecond, 40*time.Millisecond, 0)
			for _, nodeID := range nodeIDs {
				sl.SetNodeNetworkStatus(nodeID, true)
			}
			pmap := map[rpccore.NodeID]int{"0": 0, "1": 0, "2": 0, "3": 0, "4": 0}
			sl.SetNetworkPartition(pmap)
			for node, working := range nodeWorking {
				if !working {
					sl.StartPeer(node)
					nodeWorking[node] = true
				}
			}
			time.Sleep(time.Duration(10 * time.Second))
			rst[networkBackToNormal]++
		case nodeReset:
			resetNode := nodeIDs[rand.Intn(len(nodeIDs))]
			_, _ = blue.Printf("Node %v will be reset...\n", resetNode)
			if nodeWorking[resetNode] {
				err := sl.ResetPeer(resetNode)
				if err != nil {
					simulation.Log.Panicf("Reset fail: %v", err)
				}
				nodeWorking[resetNode] = false
			}
			_, _ = blue.Printf("Current working nodes: ")
			count := 0
			for node, working := range nodeWorking {
				if working {
					_, _ = blue.Printf("%v, ", node)
					count++
				}
			}
			blue.Printf(" %v nodes altogether\n", count)
			time.Sleep(time.Duration(5 * time.Second))
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
