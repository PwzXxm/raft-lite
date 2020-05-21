package rpccore

// This package provides a abstract layer of the low level rpc implementation.
// Serialization && dispatcher should be implemented in the upper level.
// There are two implementations, one is based on TCP and the other one is
// a mocked version based on channel for testing.

type NodeID string
type Callback func(source NodeID, method string, data []byte) ([]byte, error)

type Node interface {
	NodeID() NodeID
	SendRawRequest(target NodeID, method string, data []byte) ([]byte, error)
	RegisterRawRequestCallback(callback Callback)
}
