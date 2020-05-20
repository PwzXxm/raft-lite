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

// TODO: github.com/valyala/gorpc looks pretty good, but the last commit of it
// is four years ago

func init() {
	gob.Register(tcpReqMsg{})
	gob.Register(tcpResMsg{})
	gob.Register(sm.TSMAction{})

	// ignore all log printed by [gorpc]
	gorpc.SetErrorLogger(func(format string, args ...interface{}) {})
}

type TCPNetwork struct {
	lock        sync.RWMutex
	nodeAddrMap map[NodeID]string
	timeout     time.Duration
}

func NewTCPNetwork(timeout time.Duration) *TCPNetwork {
	n := new(TCPNetwork)
	n.nodeAddrMap = make(map[NodeID]string)
	n.timeout = timeout
	return n
}

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

func (n *TCPNetwork) NewLocalNode(nodeID NodeID, remoteAddr, listenAddr string) (*TCPNode, error) {

	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.nodeAddrMap[nodeID]; ok {
		return nil, errors.New(fmt.Sprintf(
			"Node with same ID already exists. NodeID: %v.", nodeID))
	}

	defaultCallback := func(source NodeID, method string, data []byte) ([]byte, error) {
		return nil, errors.New("No callback function provided.")
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
	// TODO: support graceful shutdown or at least clean shutdown
	n.nodeAddrMap[nodeID] = remoteAddr
	return node, nil
}

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
	return node, nil
}

type TCPNode struct {
	id             NodeID
	network        *TCPNetwork
	callback       Callback
	clientMap      map[NodeID]*gorpc.Client
	clientOnlyMode bool
	lock           sync.RWMutex
}

func (node *TCPNode) NodeID() NodeID {
	return node.id
}

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
	} else {
		var err error = nil
		if res.(tcpResMsg).Err != "" {
			err = errors.New(res.(tcpResMsg).Err)
		}
		return res.(tcpResMsg).Data, err
	}
}

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
