package sm

type StateMachine interface {
	Reset()
	ApplyAction(action interface{}) error
	Query(req interface{}) (interface{}, error)
}
