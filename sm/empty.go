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
