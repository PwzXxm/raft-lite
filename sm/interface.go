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

// Package sm contains two state machines, one is empty
// and the other is transaction state machines
package sm

// StateMachine records the current state and the operations of the state machine
type StateMachine interface {
	Reset()
	ApplyAction(action interface{}) error
	Query(req interface{}) (interface{}, error)
	TakeSnapshot() ([]byte, error)
	ResetWithSnapshot(b []byte) error
}
