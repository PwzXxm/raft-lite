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

package rpccore

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func init() {
	fmt.Println("* rpc core unit test *")
}

func TestNewNode(t *testing.T) {
	network := NewChanNetwork(time.Second)
	defer network.Shutdown()

	_, err := network.NewNode(NodeID("node"))
	if err != nil {
		t.Errorf("Node A should have no error")
	}

	_, err = network.NewNode(NodeID("node"))
	if err == nil {
		t.Errorf("Node B should have same ID with A")
	}
}

func TestTimeoutAndDelayGenerator(t *testing.T) {
	network := NewChanNetwork(time.Second)
	defer network.Shutdown()

	nodeA, _ := network.NewNode("nodeA")
	nodeB, _ := network.NewNode("nodeB")
	nodeC, _ := network.NewNode("nodeC")

	network.SetDelayGenerator(func(source, target NodeID) time.Duration {
		// block all message from/to node A
		if source == nodeA.NodeID() || target == nodeA.NodeID() {
			return 2 * time.Second
		}
		return 0
	})

	callback := func(source NodeID, method string, data []byte) ([]byte, error) {
		return []byte("OK"), nil
	}
	nodeA.RegisterRawRequestCallback(callback)
	nodeB.RegisterRawRequestCallback(callback)

	_, err := nodeC.SendRawRequest(nodeA.NodeID(), "", nil)
	if err == nil {
		t.Errorf("Send raw request to A should fail.")
	}

	_, err = nodeC.SendRawRequest(nodeB.NodeID(), "", nil)
	if err != nil {
		t.Errorf("Send raw request to B shouldn't fail.\n%+v", err)
	}
}

func TestNewTCPNetwork(t *testing.T) {
	tcpNetwork := NewTCPNetwork(time.Second)
	defer tcpNetwork.Shutdown()

	if tcpNetwork.timeout != time.Second {
		t.Errorf("TCPNetwork timeout should have the same value.")
	}
}

func TestNewRemoteNode(t *testing.T) {
	tcpNetwork := NewTCPNetwork(time.Second)
	defer tcpNetwork.Shutdown()

	remoteNode := tcpNetwork.NewRemoteNode("node", "addr")
	if remoteNode != nil {
		t.Errorf("RemoteNode should return nil error value.")
	}

	remoteNodeB := tcpNetwork.NewRemoteNode("node", "addr")
	if remoteNodeB == nil {
		t.Errorf("RemoteNode should return no-nil error value.")
	}
}

func TestNewLocalNode(t *testing.T) {
	tcpNetwork := NewTCPNetwork(time.Second)
	defer tcpNetwork.Shutdown()

	localNode, err := tcpNetwork.NewLocalNode("nodeB", "127.0.0.1:1110", ":1110")
	if err != nil {
		t.Errorf("LocalNode should return nil error value.")
	}
	if localNode == nil {
		t.Errorf("LocalNode should return no-nil value.")
	}

	// check node with same ID already exists
	localNodeDup, err2 := tcpNetwork.NewLocalNode("nodeB", "127.0.0.1:1110", ":1110")
	if localNodeDup != nil && err2 == nil {
		t.Errorf("LocalNode should return error value.")
	}
}

func checkNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Shouldn't be an error: %+v", errors.WithStack(err))
	}
}

func TestChanNode(t *testing.T) {
	network := NewChanNetwork(time.Second)
	defer network.Shutdown()
	nodeA, err := network.NewNode("nodeA")
	checkNoError(t, err)
	nodeB, err := network.NewNode("nodeB")
	checkNoError(t, err)

	testNode(t, nodeA, nodeB)
}

func TestTCPNode(t *testing.T) {
	networkA := NewTCPNetwork(time.Second)
	defer networkA.Shutdown()
	nodeA, err := networkA.NewLocalNode("nodeA", "127.0.0.1:1111", ":1111")
	checkNoError(t, err)
	err = networkA.NewRemoteNode("nodeB", "127.0.0.1:1112")
	checkNoError(t, err)

	networkB := NewTCPNetwork(time.Second)
	defer networkB.Shutdown()
	nodeB, err := networkB.NewLocalNode("nodeB", "127.0.0.1:1112", ":1112")
	checkNoError(t, err)

	testNode(t, nodeA, nodeB)
}

func testNode(t *testing.T, nodeA Node, nodeB Node) {
	// test NodeID
	if nodeA.NodeID() != "nodeA" || nodeB.NodeID() != "nodeB" {
		t.Errorf("Node has incorrect NodeID.")
	}

	// test RegisterRawRequestCallback
	callback := func(source NodeID, method string, data []byte) ([]byte, error) {
		if string(data) == "Test: A -> B" {
			return []byte(string(source)), nil
		}

		return []byte(string(source)), errors.New("Incorrent data")
	}
	nodeB.RegisterRawRequestCallback(callback)

	// test SendRawRequest
	data := []byte("Test: A -> B")
	_, err := nodeA.SendRawRequest(nodeB.NodeID(), "test", data)
	if err != nil {
		t.Errorf("Node A should receive callback.\n%+v", err)
	}

	data = []byte("Test: B -> A")
	_, err = nodeB.SendRawRequest(nodeA.NodeID(), "test", data)
	if err == nil {
		t.Errorf("Node B should receive no callback error.")
	}
}

func BenchmarkChanCommunication(b *testing.B) {
	network := NewChanNetwork(time.Second)
	defer network.Shutdown()
	nodeA, _ := network.NewNode("nodeA")
	nodeB, _ := network.NewNode("nodeB")
	nodeC, _ := network.NewNode("nodeC")

	benchmarkCommunication(b, nodeA, nodeB, nodeC)
}

func BenchmarkTCPCommunication(b *testing.B) {
	networkA := NewTCPNetwork(time.Second)
	defer networkA.Shutdown()
	nodeA, _ := networkA.NewLocalNode("nodeA", "127.0.0.1:1113", ":1113")
	_ = networkA.NewRemoteNode("nodeB", "127.0.0.1:1114")
	_ = networkA.NewRemoteNode("nodeC", "127.0.0.1:1115")

	networkB := NewTCPNetwork(time.Second)
	defer networkB.Shutdown()
	nodeB, _ := networkB.NewLocalNode("nodeB", "127.0.0.1:1114", ":1114")
	_ = networkB.NewRemoteNode("nodeA", "127.0.0.1:1113")
	_ = networkB.NewRemoteNode("nodeC", "127.0.0.1:1115")

	networkC := NewTCPNetwork(time.Second)
	defer networkC.Shutdown()
	nodeC, _ := networkC.NewLocalNode("nodeC", "127.0.0.1:1115", ":1115")
	_ = networkC.NewRemoteNode("nodeA", "127.0.0.1:1113")
	_ = networkC.NewRemoteNode("nodeB", "127.0.0.1:1114")

	benchmarkCommunication(b, nodeA, nodeB, nodeC)
}

func benchmarkCommunication(b *testing.B, nodeA Node, nodeB Node, nodeC Node) {
	callbackHandler := func(source NodeID, method string, data []byte) ([]byte, error) {
		return []byte(string(source)), nil
	}

	nodeA.RegisterRawRequestCallback(callbackHandler)
	nodeB.RegisterRawRequestCallback(callbackHandler)
	nodeC.RegisterRawRequestCallback(callbackHandler)

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fmt.Printf("Worker %d starting\n", i)
			for j := 0; j < b.N*1000000; j++ {
				switch {
				case i < 2:
					nodeA.SendRawRequest(nodeB.NodeID(), "test", []byte("Test: A -> B"))
				case i < 4:
					nodeB.SendRawRequest(nodeC.NodeID(), "test", []byte("Test: B -> C"))
				case i < 6:
					nodeC.SendRawRequest(nodeA.NodeID(), "test", []byte("Test: C -> A"))
				}
			}
			fmt.Printf("Worker %d done\n", i)
		}(i)
	}
	wg.Wait()
}
