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
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/PwzXxm/raft-lite/sm"
	"github.com/pkg/errors"
	"github.com/valyala/gorpc"
)

func init() {
	gob.Register(tcpReqMsg{})
	gob.Register(tcpResMsg{})
	gob.Register(sm.TSMAction{})

	// ignore all log printed by [gorpc]
	gorpc.SetErrorLogger(func(format string, args ...interface{}) {})
}

// TCPNetwork representing the network using TCP
// including all the nodes exisiting in the current network
type TCPNetwork struct {
	lock        sync.RWMutex
	nodeAddrMap map[NodeID]string
	timeout     time.Duration
	localNodes  map[NodeID]*TCPNode
}

// NewTCPNetwork creates a new TCP newwork for other nodes to join in
func NewTCPNetwork(timeout time.Duration) *TCPNetwork {
	n := new(TCPNetwork)
	n.nodeAddrMap = make(map[NodeID]string)
	n.localNodes = make(map[NodeID]*TCPNode)
	n.timeout = timeout
	return n
}

// Shutdown the TCP network by stopping all nodes
func (n *TCPNetwork) Shutdown() {
	n.lock.Lock()
	defer n.lock.Unlock()
	for _, node := range n.localNodes {
		node.lock.Lock()
		for _, client := range node.clientMap {
			client.Stop()
		}
		if !node.clientOnlyMode {
			node.s.Stop()
			node.s = nil
		}
		node.clientMap = nil
		node.lock.Unlock()
	}
	n.localNodes = nil
	n.nodeAddrMap = nil
}

// NewRemoteNode adds a new remote TCP node in the current network
func (n *TCPNetwork) NewRemoteNode(nodeID NodeID, addr string) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.nodeAddrMap[nodeID]; ok {
		return errors.New(fmt.Sprintf(
			"Node with same ID already exists. NodeID: %v.", nodeID))
	}
	n.nodeAddrMap[nodeID] = addr
	return nil
}

// NewLocalNode starts a new server at local and starts to listening on given address and port
func (n *TCPNetwork) NewLocalNode(nodeID NodeID, remoteAddr, listenAddr string) (*TCPNode, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.nodeAddrMap[nodeID]; ok {
		return nil, errors.New(fmt.Sprintf(
			"Node with same ID already exists. NodeID: %v.", nodeID))
	}

	defaultCallback := func(source NodeID, method string, data []byte) ([]byte, error) {
		return nil, errors.New("No callback function provided")
	}

	node := &TCPNode{
		id:             nodeID,
		network:        n,
		callback:       defaultCallback,
		clientMap:      make(map[NodeID]*gorpc.Client),
		clientOnlyMode: false,
	}

	s := &gorpc.Server{
		Addr: listenAddr,
		Handler: func(clientAddr string, request interface{}) interface{} {
			node.lock.RLock()
			callback := node.callback
			node.lock.RUnlock()
			req := request.(tcpReqMsg)
			data, err := callback(req.Source, req.Method, req.Data)
			errStr := ""
			if err != nil {
				errStr = fmt.Sprintf("%v", err)
			}
			return &tcpResMsg{Data: data, Err: errStr}
		},
	}
	if err := s.Start(); err != nil {
		return nil, err
	}
	node.s = s
	n.nodeAddrMap[nodeID] = remoteAddr
	n.localNodes[nodeID] = node
	return node, nil
}

// NewLocalClientOnlyNode creates a TCP node representing the client
// it does not act as a server
func (n *TCPNetwork) NewLocalClientOnlyNode(nodeID NodeID) (*TCPNode, error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.nodeAddrMap[nodeID]; ok {
		return nil, errors.New(fmt.Sprintf(
			"Node with same ID already exists. NodeID: %v.", nodeID))
	}

	node := &TCPNode{
		id:             nodeID,
		network:        n,
		clientMap:      make(map[NodeID]*gorpc.Client),
		clientOnlyMode: true,
	}
	n.localNodes[nodeID] = node
	return node, nil
}

// A TCPNode representing a node within the current TCP network
// and the client associate with it if it is not in client only mode
type TCPNode struct {
	id             NodeID
	network        *TCPNetwork
	callback       Callback
	clientMap      map[NodeID]*gorpc.Client
	clientOnlyMode bool
	lock           sync.RWMutex
	s              *gorpc.Server
}

// NodeID gets the node's ID
func (node *TCPNode) NodeID() NodeID {
	return node.id
}

// SendRawRequest invokes an RPC method on the target node
func (node *TCPNode) SendRawRequest(target NodeID, method string, data []byte) ([]byte, error) {
	node.lock.RLock()
	client, ok := node.clientMap[target]
	node.lock.RUnlock()
	if !ok {
		// first time sending to this target, create a new client
		// Double-checked locking (Write lock)
		node.lock.Lock()
		client, ok = node.clientMap[target]
		if !ok {
			node.network.lock.RLock()
			addr, ok := node.network.nodeAddrMap[target]
			node.network.lock.RUnlock()
			if ok {
				client = &gorpc.Client{Addr: addr, RequestTimeout: node.network.timeout}
				client.Start()
				node.clientMap[target] = client
			} else {
				client = nil
			}
		}
		node.lock.Unlock()
	}
	if client == nil {
		err := errors.New(fmt.Sprintf(
			"Unable to find target node: %v.", target))
		return nil, err
	}
	res, err := client.Call(&tcpReqMsg{Source: node.id, Method: method, Data: data})
	if err != nil {
		return nil, err
	}

	if res.(tcpResMsg).Err != "" {
		err = errors.New(res.(tcpResMsg).Err)
	}
	return res.(tcpResMsg).Data, err
}

// RegisterRawRequestCallback let nodes to register methods that will be called when receiving a RPC
func (node *TCPNode) RegisterRawRequestCallback(callback Callback) {
	node.lock.Lock()
	node.callback = callback
	node.lock.Unlock()
}

type tcpReqMsg struct {
	Source NodeID
	Method string
	Data   []byte
}

type tcpResMsg struct {
	Err  string
	Data []byte
}
