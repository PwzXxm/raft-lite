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

func (t *TSM) ApplyAction(action interface{}) error {
	tsmAction := action.(TSMAction)
	switch tsmAction.action {
	case tsmActionSet:
		t.data[tsmAction.target] = tsmAction.value
	case tsmActionIncr:
		v, ok := t.data[tsmAction.target]
		if !ok {
			return errors.Errorf("invalid key: %v", tsmAction.target)
		}
		t.data[tsmAction.target] = v + tsmAction.value
	case tsmActionMove:
		sv, ok := t.data[tsmAction.source]
		if !ok {
			return errors.Errorf("invalid key for source: %v", tsmAction.source)
		}
		tv, ok := t.data[tsmAction.target]
		if !ok {
			return errors.Errorf("invalid key for target: %v", tsmAction.target)
		}
		t.data[tsmAction.source] = sv - tsmAction.value
		t.data[tsmAction.target] = tv + tsmAction.value
	}
	return nil
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

// I really don't like the non-type-safe approach below, but I couldn't find a
// better way. There is an another approach that use anonymous functions as
// [TSMAction] but it can't be serialized.

type TSMAction struct {
	action tsmActionType
	target string
	source string // useless for [tsmActionSet] and [tsmActionIncr]
	value  int
}

type tsmActionType int

const (
	tsmActionSet tsmActionType = iota
	tsmActionIncr
	tsmActionMove
)

func TSMActionSetValue(key string, value int) TSMAction {
	return TSMAction{
		action: tsmActionSet,
		target: key,
		value:  value,
	}
}

func TSMActionIncrValue(key string, value int) TSMAction {
	return TSMAction{
		action: tsmActionIncr,
		target: key,
		value:  value,
	}
}

func TSMActionMoveValue(source, target string, value int) TSMAction {
	return TSMAction{
		action: tsmActionMove,
		source: source,
		target: target,
		value:  value,
	}
}
