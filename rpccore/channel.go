package rpccore

import (
	"fmt"
	"log"

	"github.com/pkg/errors"
)

type ChanAddress struct {
	nodeID NodeID
}

func NewChanAddress(nodeID string) *ChanAddress {
	addr := new(ChanAddress)
	addr.nodeID = NodeID(nodeID)
	return addr
}

func (addr *ChanAddress) NodeID() NodeID {
	return addr.nodeID
}

type ChanNode struct {
	id       NodeID
	network  *ChanNetwork
	callback Callback
}

func (node *ChanNode) SendRawRequest(target NodeID, method string, data []byte) ([]byte, error) {
	if reqChan, ok := node.network.nodeChannelMap[target]; ok {
		resChan := make(chan *resMsg)
		req := reqMsg{source: node.id, method: method, data: data, resChan: resChan}
		reqChan <- &req
		res := <-resChan
		return res.data, res.err
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
	nodeChannelMap map[NodeID](chan *reqMsg)
}

func NewChanNetwork() *ChanNetwork {
	n := new(ChanNetwork)
	n.nodeChannelMap = make(map[NodeID](chan *reqMsg))
	return n
}

func (n *ChanNetwork) NewNode(addr ChanAddress) (*ChanNode, error) {
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
				data, err := node.callback(req.source, req.method, req.data)
				res := resMsg{data: data, err: err}
				req.resChan <- &res
			}(req)
		}
	}()
	return node, nil
}

// we are passing pointer in the channel, we should treat those two
// structs as immutable
type reqMsg struct {
	source  NodeID
	method  string
	data    []byte
	resChan chan *resMsg
}

type resMsg struct {
	err  error
	data []byte
}
