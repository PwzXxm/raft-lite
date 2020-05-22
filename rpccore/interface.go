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

// Package rpccore provides a abstract layer of the low level rpc implementation.
// Serialization && dispatcher should be implemented in the upper level.
// There are two implementations, one is based on TCP and the other one is
// a mocked version based on channel for testing.
package rpccore

// NodeID representing the ID of the channel node
type NodeID string

// Callback representing the function that handels the RPC call
type Callback func(source NodeID, method string, data []byte) ([]byte, error)

// Node representing the node
type Node interface {
	// NodeID gets the node's ID
	NodeID() NodeID

	// SendRawRequest invokes an RPC method on the target node
	SendRawRequest(target NodeID, method string, data []byte) ([]byte, error)

	// RegisterRawRequestCallback let nodes to register methods
	// that will be called when receiving a RPC
	RegisterRawRequestCallback(callback Callback)
}
