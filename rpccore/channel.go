package rpccore

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type ChanAddress struct {
	nodeID NodeID
}

func NewChanAddress(nodeID string) ChanAddress {
	addr := ChanAddress{}
	addr.nodeID = NodeID(nodeID)
	return addr
}

func (addr ChanAddress) NodeID() NodeID {
	return addr.nodeID
}

type ChanNode struct {
	id       NodeID
	network  *ChanNetwork
	callback Callback
}

func (node *ChanNode) NodeID() NodeID {
	return node.id
}

func (node *ChanNode) SendRawRequest(target NodeID, method string, data []byte) ([]byte, error) {
	node.network.lock.RLock()
	reqChan, ok := node.network.nodeChannelMap[target]
	deadline := time.Now().Add(node.network.timeout)
	node.network.lock.RUnlock()
	if ok {
		resChan := make(chan *resMsg)
		req := reqMsg{source: node.id, method: method, data: data,
			resChan: resChan, deadline: deadline}
		reqChan <- &req
		remainingTime := deadline.Sub(time.Now())
		if remainingTime <= 0 {
			return nil, errors.New("Request timeout.")
		}
		select {
		case res := <-resChan:
			return res.data, res.err
		case <-time.After(remainingTime):
			return nil, errors.New("Request timeout.")
		}
	} else {
		err := errors.New(fmt.Sprintf(
			"Unable to find target node: %v.", target))
		return nil, err
	}
}

func (node *ChanNode) RegisterRawRequestCallback(callback Callback) {
	node.callback = callback
}

type ChanNetwork struct {
	lock           sync.RWMutex
	nodeChannelMap map[NodeID](chan *reqMsg)
	timeout        time.Duration
	delayGenerator DelayGenerator
}

func NewChanNetwork(timeout time.Duration) *ChanNetwork {
	n := new(ChanNetwork)
	n.lock = sync.RWMutex{}
	n.nodeChannelMap = make(map[NodeID](chan *reqMsg))
	n.timeout = timeout
	n.delayGenerator = func(source, target NodeID) time.Duration {
		return 0
	}
	return n
}

func (n *ChanNetwork) NewNode(addr ChanAddress) (*ChanNode, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.nodeChannelMap[addr.NodeID()]; ok {
		err := errors.New(fmt.Sprintf(
			"Node with same ID already exists. NodeID: %v.", addr.NodeID()))
		return nil, err
	}
	node := new(ChanNode)
	node.network = n
	node.id = addr.NodeID()
	// don't need buffer here since the send is blocking anyway.
	nodeChannel := make(chan *reqMsg)
	n.nodeChannelMap[node.id] = nodeChannel

	// start listening loop
	go func() {
		for req := range nodeChannel {
			if node.callback == nil {
				log.Fatalf("Cannot find Callback for NodeID: %v.", node.id)
			}
			// start a new goroutine for handling this request
			go func(req *reqMsg) {
				// delay (sender to receiver)
				n.lock.RLock()
				delay := n.delayGenerator(req.source, addr.NodeID())
				n.lock.RUnlock()
				time.Sleep(delay)
				remainingTime := req.deadline.Sub(time.Now())
				if remainingTime <= 0 {
					req.resChan <- &resMsg{data: nil,
						err: errors.New("Request timeout.")}
				} else {
					// invoke callback
					data, err := node.callback(req.source, req.method, req.data)
					// delay (receiver to sender)
					n.lock.RLock()
					delay := n.delayGenerator(addr.NodeID(), req.source)
					n.lock.RUnlock()
					if delay <= remainingTime {
						time.Sleep(delay)
						req.resChan <- &resMsg{data: data, err: err}
					}
				}

			}(req)
		}
	}()
	return node, nil
}

type DelayGenerator func(source NodeID, target NodeID) time.Duration

func (n *ChanNetwork) SetDelayGenerator(delayGenerator DelayGenerator) {
	n.lock.Lock()
	n.delayGenerator = delayGenerator
	n.lock.Unlock()
}

// we are passing pointer in the channel, we should treat those two
// structs as immutable
type reqMsg struct {
	source   NodeID
	method   string
	data     []byte
	resChan  chan *resMsg
	deadline time.Time
}

type resMsg struct {
	err  error
	data []byte
}
