package rpccore

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func init() {
	fmt.Println("* rpc core test *")
}

func TestNewChanAddress(t *testing.T) {
	nodeID := "test"
	addr := NewChanAddress(nodeID)

	if addr.NodeID() != NodeID(nodeID) {
		t.Errorf("ChanAddress NodeID is %v; want %v", addr.NodeID(), nodeID)
	}
}

func TestNewNode(t *testing.T) {
	network := NewChanNetwork(time.Second)

	addrA := NewChanAddress("node")
	addrB := NewChanAddress("node")

	_, err := network.NewNode(addrA)
	if err != nil {
		t.Errorf("Node A should have no error")
	}

	_, err = network.NewNode(addrB)

	if err == nil {
		t.Errorf("Node B should have same ID with A")
	}
}

func TestCommunication(t *testing.T) {
	network := NewChanNetwork(time.Second)

	addrA := NewChanAddress("nodeA")
	addrB := NewChanAddress("nodeB")
	addrC := NewChanAddress("nodeC")

	nodeA, _ := network.NewNode(addrA)
	nodeB, _ := network.NewNode(addrB)
	nodeC, _ := network.NewNode(addrC)

	nodeB.RegisterRawRequestCallback(func(source NodeID, method string, data []byte) ([]byte, error) {
		if string(data) == "Test: A -> B" {
			return []byte(string(source)), nil
		} else {
			return []byte(string(source)), errors.New("Incorrect data")
		}
	})

	data := []byte("Test: A -> B")
	_, err := nodeA.SendRawRequest(NodeID("nodeB"), "test", data)
	if err != nil {
		t.Errorf("Node A should receive callback.\n%+v", err)
	}

	data = []byte("Test: C -> B")
	_, err = nodeC.SendRawRequest(NodeID("nodeB"), "test", data)
	if err == nil {
		t.Errorf("Node C should receive error")
	}
}

func TestTimeoutAndDelayGenerator(t *testing.T) {
	addrA := NewChanAddress("nodeA")
	addrB := NewChanAddress("nodeB")
	addrC := NewChanAddress("nodeC")

	network := NewChanNetwork(time.Second)
	network.SetDelayGenerator(func(source, target NodeID) time.Duration {
		// block all message from/to node A
		if source == addrA.NodeID() || target == addrA.NodeID() {
			return 2 * time.Second
		}
		return 0
	})

	nodeA, _ := network.NewNode(addrA)
	nodeB, _ := network.NewNode(addrB)
	nodeC, _ := network.NewNode(addrC)

	callback := func(source NodeID, method string, data []byte) ([]byte, error) {
		return []byte("OK"), nil
	}
	nodeA.RegisterRawRequestCallback(callback)
	nodeB.RegisterRawRequestCallback(callback)

	_, err := nodeC.SendRawRequest(addrA.NodeID(), "", nil)
	if err == nil {
		t.Errorf("Send raw request to A should fail.")
	}

	_, err = nodeC.SendRawRequest(addrB.NodeID(), "", nil)
	if err != nil {
		t.Errorf("Send raw request to B shouldn't fail.\n%+v", err)
	}
}

func BenchmarkCommunication(b *testing.B) {
	network := NewChanNetwork(time.Second)

	addrA := NewChanAddress("nodeA")
	addrB := NewChanAddress("nodeB")
	addrC := NewChanAddress("nodeC")

	nodeA, _ := network.NewNode(addrA)
	nodeB, _ := network.NewNode(addrB)
	nodeC, _ := network.NewNode(addrC)

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
					nodeA.SendRawRequest(NodeID("nodeB"), "test", []byte("Test: A -> B"))
				case i < 4:
					nodeB.SendRawRequest(NodeID("nodeC"), "test", []byte("Test: B -> C"))
				case i < 6:
					nodeC.SendRawRequest(NodeID("nodeA"), "test", []byte("Test: C -> A"))
				}
			}
			fmt.Printf("Worker %d done\n", i)
		}(i)
	}
	wg.Wait()
}
