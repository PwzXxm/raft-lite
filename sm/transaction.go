package sm

import (
	"github.com/pkg/errors"
)

type TSM struct {
	data map[string]int
}

func NewTransactionStateMachine() *TSM {
	t := new(TSM)
	t.data = make(map[string]int)
	return t
}

type TSMAction func(t *TSM) error

func (t *TSM) ApplyAction(action interface{}) error {
	return action.(TSMAction)(t)
}

// return the value of the given key.
// key is string and return value is int
func (t *TSM) Query(req interface{}) (interface{}, error) {
	key := req.(string)
	v, ok := t.data[key]
	if !ok {
		return nil, errors.Errorf("invalid key: %v", key)
	}
	return v, nil
}

func TSMActionSetValue(key string, value int) TSMAction {
	return func(t *TSM) error {
		t.data[key] = value
		return nil
	}
}

func TSMActionIncrValue(key string, value int) TSMAction {
	return func(t *TSM) error {
		v, ok := t.data[key]
		if !ok {
			return errors.Errorf("invalid key: %v", key)
		}
		t.data[key] = v + value
		return nil
	}
}

func TSMActionMoveValue(source, target string, value int) TSMAction {
	return func(t *TSM) error {
		sv, ok := t.data[source]
		if !ok {
			return errors.Errorf("invalid key for source: %v", source)
		}
		tv, ok := t.data[target]
		if !ok {
			return errors.Errorf("invalid key for target: %v", target)
		}
		t.data[source] = sv - value
		t.data[target] = tv + value
		return nil
	}
}
