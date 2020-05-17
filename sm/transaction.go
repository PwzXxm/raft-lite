package sm

import (
	"bytes"
	"encoding/gob"
	"math/rand"

	"github.com/pkg/errors"
)

// This state machine is not thread-safe
type TSM struct {
	data            map[string]int
	latestRequestID map[string]uint32
}

func NewTransactionStateMachine() *TSM {
	t := new(TSM)
	t.data = make(map[string]int)
	t.latestRequestID = make(map[string]uint32)
	return t
}

func (t *TSM) Reset() {
	t.data = make(map[string]int)
	t.latestRequestID = make(map[string]uint32)
}

func (t *TSM) ApplyAction(action interface{}) error {
	tsmAction := action.(TSMAction)
	// check duplicate
	lastID, ok := t.latestRequestID[tsmAction.ClientID]
	if ok {
		if lastID == tsmAction.RequestID {
			// duplicate request, ignore it
			return nil
		}
	}
	t.latestRequestID[tsmAction.ClientID] = tsmAction.RequestID

	// execute action
	switch tsmAction.Action {
	case tsmActionSet:
		t.data[tsmAction.Target] = tsmAction.Value
	case tsmActionIncr:
		v, ok := t.data[tsmAction.Target]
		if !ok {
			return errors.Errorf("invalid key: %v", tsmAction.Target)
		}
		t.data[tsmAction.Target] = v + tsmAction.Value
	case tsmActionMove:
		sv, ok := t.data[tsmAction.Source]
		if !ok {
			return errors.Errorf("invalid key for source: %v", tsmAction.Source)
		}
		tv, ok := t.data[tsmAction.Target]
		if !ok {
			return errors.Errorf("invalid key for target: %v", tsmAction.Target)
		}
		t.data[tsmAction.Source] = sv - tsmAction.Value
		t.data[tsmAction.Target] = tv + tsmAction.Value
	}
	return nil
}

// return the value of the given key.
// key is string and return value is int
func (t *TSM) Query(req interface{}) (interface{}, error) {
	query := req.(TSMQuery)
	switch query.Query {
	case tsmQueryData:
		key := query.Key
		v, ok := t.data[key]
		if !ok {
			return nil, errors.New("key does not exist")
		}
		return v, nil
	case tsmQueryLatestRequest:
		client := query.Key
		v, ok := t.latestRequestID[client]
		if !ok {
			return nil, errors.New("key does not exist")
		}
		return v, nil
	}
	return nil, errors.New("invalid query")
}

func (t *TSM) TakeSnapshot() ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)

	err := enc.Encode(t)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *TSM) ResetWithSnapshot(b []byte) error {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	var newTSM TSM
	err := dec.Decode(&newTSM)
	if err == nil {
		t.data = newTSM.data
		t.latestRequestID = newTSM.latestRequestID
	}
	return err
}

// I really don't like the non-type-safe approach below, but I couldn't find a
// better way. There is an another approach that use anonymous functions as
// [TSMAction] but it can't be serialized.

type TSMAction struct {
	Action tsmActionType
	Target string
	Source string // useless for [tsmActionSet] and [tsmActionIncr]
	Value  int
	// request info
	ClientID  string
	RequestID uint32
}

type tsmActionType int

const (
	tsmActionSet tsmActionType = iota
	tsmActionIncr
	tsmActionMove
)

func (a TSMAction) GetRequestID() uint32 {
	return a.RequestID
}

type TSMActionBuilder struct {
	clientID      string
	lastRequestID uint32
}

func NewTSMActionBuilder(clientID string) *TSMActionBuilder {
	return &TSMActionBuilder{clientID: clientID, lastRequestID: rand.Uint32()}
}

func (b *TSMActionBuilder) newRequestID() uint32 {
	b.lastRequestID += 1
	return b.lastRequestID
}

func (b *TSMActionBuilder) TSMActionSetValue(key string, value int) TSMAction {
	return TSMAction{
		Action:    tsmActionSet,
		Target:    key,
		Value:     value,
		ClientID:  b.clientID,
		RequestID: b.newRequestID(),
	}
}

func (b *TSMActionBuilder) TSMActionIncrValue(key string, value int) TSMAction {
	return TSMAction{
		Action:    tsmActionIncr,
		Target:    key,
		Value:     value,
		ClientID:  b.clientID,
		RequestID: b.newRequestID(),
	}
}

func (b *TSMActionBuilder) TSMActionMoveValue(source, target string, value int) TSMAction {
	return TSMAction{
		Action:    tsmActionMove,
		Source:    source,
		Target:    target,
		Value:     value,
		ClientID:  b.clientID,
		RequestID: b.newRequestID(),
	}
}

type TSMQuery struct {
	Query tsmQueryType
	Key   string
}

type tsmQueryType int

const (
	tsmQueryData tsmQueryType = iota
	tsmQueryLatestRequest
)

func NewTSMDataQuery(key string) TSMQuery {
	return TSMQuery{
		Query: tsmQueryData,
		Key:   key,
	}
}

func NewTSMLatestRequestQuery(clientID string) TSMQuery {
	return TSMQuery{
		Query: tsmQueryLatestRequest,
		Key:   clientID,
	}
}
