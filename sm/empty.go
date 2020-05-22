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

package sm

// ESM dummy strcut
type ESM struct {
}

// NewEmptyStateMachine new state machine
func NewEmptyStateMachine() *ESM {
	return new(ESM)
}

// Reset dummy
func (t *ESM) Reset() {
}

// ApplyAction dummy
func (t *ESM) ApplyAction(action interface{}) error {
	return nil
}

// Query dummy
func (t *ESM) Query(req interface{}) (interface{}, error) {
	return nil, nil
}

// TakeSnapshot dummy
func (t *ESM) TakeSnapshot() ([]byte, error) {
	return nil, nil
}

// ResetWithSnapshot dummy
func (t *ESM) ResetWithSnapshot(b []byte) error {
	return nil
}
