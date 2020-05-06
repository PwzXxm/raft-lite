package sm

type StateMachine interface {
	ApplyAction(action interface{}) error
	Query(req interface{}) (interface{}, error)
}
