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
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/sasha-s/go-deadlock"
)

// A ChanNode representing a node using channel
type ChanNode struct {
	id       NodeID
	network  *ChanNetwork
	callback Callback
	lock     deadlock.RWMutex
}

// NodeID gets the node's ID
func (node *ChanNode) NodeID() NodeID {
	return node.id
}

// SendRawRequest invokes an RPC method on the target node
func (node *ChanNode) SendRawRequest(target NodeID, method string, data []byte) ([]byte, error) {
	node.network.lock.RLock()
	reqChan, ok := node.network.nodeChannelMap[target]
	deadline := time.Now().Add(node.network.timeout)
	node.network.lock.RUnlock()
	if ok {
		// start sending request
		resChan := make(chan *chanResMsg)
		req := chanReqMsg{source: node.id, method: method, data: data,
			resChan: resChan, deadline: deadline}
		reqChan <- &req
		remainingTime := deadline.Sub(time.Now())
		if remainingTime <= 0 {
			return nil, errors.New("Request timeout")
		}

		// wait for response
		select {
		case res := <-resChan:
			return res.data, res.err
		case <-time.After(remainingTime):
			return nil, errors.New("Request timeout")
		}
	} else {
		err := errors.New(fmt.Sprintf("Unable to find target node: %v", target))
		return nil, err
	}
}

// RegisterRawRequestCallback let nodes to register methods
// that will be called when receiving a RPC
func (node *ChanNode) RegisterRawRequestCallback(callback Callback) {
	node.lock.Lock()
	node.callback = callback
	node.lock.Unlock()
}

// ChanNetwork representing the network information
// including available nodes, timeout and delays
type ChanNetwork struct {
	lock           deadlock.RWMutex
	nodeChannelMap map[NodeID](chan *chanReqMsg)
	timeout        time.Duration
	delayGenerator DelayGenerator
}

// NewChanNetwork creates a newwork using channels
func NewChanNetwork(timeout time.Duration) *ChanNetwork {
	n := new(ChanNetwork)
	n.nodeChannelMap = make(map[NodeID](chan *chanReqMsg))
	n.timeout = timeout
	n.delayGenerator = func(source, target NodeID) time.Duration {
		return 0
	}
	return n
}

// Shutdown shuts down the network by closing all channels
func (n *ChanNetwork) Shutdown() {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, channel := range n.nodeChannelMap {
		close(channel)
	}
	n.nodeChannelMap = nil
}

// NewNode creates a new node in the network and starts to reads from the channel
func (n *ChanNetwork) NewNode(nodeID NodeID) (*ChanNode, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.nodeChannelMap[nodeID]; ok {
		err := errors.New(fmt.Sprintf(
			"Node with same ID already exists. NodeID: %v.", nodeID))
		return nil, err
	}
	node := new(ChanNode)
	node.network = n
	node.id = nodeID
	node.callback = func(source NodeID, method string, data []byte) ([]byte, error) {
		return nil, errors.New("No callback function provided")
	}
	// don't need buffer here since the send is blocking anyway.
	nodeChannel := make(chan *chanReqMsg)
	n.nodeChannelMap[node.id] = nodeChannel

	// start listening loop
	go func() {
		for req := range nodeChannel {
			if node.callback == nil {
				log.Fatalf("Cannot find Callback for NodeID: %v.", node.id)
			}
			node.lock.RLock()
			callback := node.callback
			node.lock.RUnlock()
			// start a new goroutine for handling this request
			go func(req *chanReqMsg) {
				// delay (sender to receiver)
				n.lock.RLock()
				delay := n.delayGenerator(req.source, nodeID)
				n.lock.RUnlock()
				time.Sleep(delay)
				remainingTime := req.deadline.Sub(time.Now())
				if remainingTime <= 0 {
					req.resChan <- &chanResMsg{data: nil,
						err: errors.New("Request timeout")}
				} else {
					// invoke callback
					data, err := callback(req.source, req.method, req.data)
					// delay (receiver to sender)
					n.lock.RLock()
					delay := n.delayGenerator(nodeID, req.source)
					n.lock.RUnlock()
					if delay <= remainingTime {
						time.Sleep(delay)
						req.resChan <- &chanResMsg{data: data, err: err}
					}
				}

			}(req)
		}
	}()
	return node, nil
}

// DelayGenerator generates delays between source and target node
type DelayGenerator func(source NodeID, target NodeID) time.Duration

// SetDelayGenerator updates the state of the generator
func (n *ChanNetwork) SetDelayGenerator(delayGenerator DelayGenerator) {
	n.lock.Lock()
	n.delayGenerator = delayGenerator
	n.lock.Unlock()
}

// we are passing pointer in the channel, we should treat those two
// those structs should be immutable
type chanReqMsg struct {
	source   NodeID
	method   string
	data     []byte
	resChan  chan *chanResMsg
	deadline time.Time
}

type chanResMsg struct {
	err  error
	data []byte
}
