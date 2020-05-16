package sm

type StateMachine interface {
	Reset()
	ApplyAction(action interface{}) error
	Query(req interface{}) (interface{}, error)
	TakeSnapshot() ([]byte, error)
	ResetWithSnapshot(b []byte) error
}
