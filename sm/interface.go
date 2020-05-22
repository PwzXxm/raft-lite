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
	Reset()                                     // reset state machine
	ApplyAction(action interface{}) error       // apply action to state machine
	Query(req interface{}) (interface{}, error) // query from state machine
	TakeSnapshot() ([]byte, error)              // take snapshot of the state machine
	ResetWithSnapshot(b []byte) error           // reset the state machine with given snapshot
}
