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

type ESM struct {
}

func NewEmptyStateMachine() *ESM {
	return new(ESM)
}

func (t *ESM) Reset() {
}

func (t *ESM) ApplyAction(action interface{}) error {
	return nil
}

func (t *ESM) Query(req interface{}) (interface{}, error) {
	return nil, nil
}

func (t *ESM) TakeSnapshot() ([]byte, error) {
	return nil, nil
}

func (t *ESM) ResetWithSnapshot(b []byte) error {
	return nil
}
