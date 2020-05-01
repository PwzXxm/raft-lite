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

func TestNewNode(t *testing.T) {
	network := NewChanNetwork(time.Second)

	_, err := network.NewNode(NodeID("node"))
	if err != nil {
		t.Errorf("Node A should have no error")
	}

	_, err = network.NewNode(NodeID("node"))

	if err == nil {
		t.Errorf("Node B should have same ID with A")
	}
}

func TestCommunication(t *testing.T) {
	network := NewChanNetwork(time.Second)

	nodeA, _ := network.NewNode("nodeA")
	nodeB, _ := network.NewNode("nodeB")
	nodeC, _ := network.NewNode("nodeC")

	nodeB.RegisterRawRequestCallback(func(source NodeID, method string, data []byte) ([]byte, error) {
		if string(data) == "Test: A -> B" {
			return []byte(string(source)), nil
		} else {
			return []byte(string(source)), errors.New("Incorrect data")
		}
	})

	data := []byte("Test: A -> B")
	_, err := nodeA.SendRawRequest(nodeB.NodeID(), "test", data)
	if err != nil {
		t.Errorf("Node A should receive callback.\n%+v", err)
	}

	data = []byte("Test: C -> B")
	_, err = nodeC.SendRawRequest(nodeB.NodeID(), "test", data)
	if err == nil {
		t.Errorf("Node C should receive error")
	}
}

func TestTimeoutAndDelayGenerator(t *testing.T) {
	network := NewChanNetwork(time.Second)

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

func BenchmarkCommunication(b *testing.B) {
	network := NewChanNetwork(time.Second)

	nodeA, _ := network.NewNode("nodeA")
	nodeB, _ := network.NewNode("nodeB")
	nodeC, _ := network.NewNode("nodeC")

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
