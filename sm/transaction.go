package sm

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"reflect"

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
	fmt.Printf("Apply actions\n")
	tsmAction := action.(TSMAction)
	// check duplicate
	lastID, ok := t.latestRequestID[tsmAction.clientID]
	if ok {
		if lastID == tsmAction.requestID {
			// duplicate request, ignore it
			return nil
		}
	}
	t.latestRequestID[tsmAction.clientID] = tsmAction.requestID

	// execute action
	switch tsmAction.action {
	case tsmActionSet:
		fmt.Printf("Action1\n")
		t.data[tsmAction.target] = tsmAction.value
	case tsmActionIncr:
		fmt.Printf("Action2\n")
		v, ok := t.data[tsmAction.target]
		if !ok {
			return errors.Errorf("invalid key: %v", tsmAction.target)
		}
		t.data[tsmAction.target] = v + tsmAction.value
	case tsmActionMove:
		fmt.Printf("Action3\n")
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
func (t *TSM) Query(req interface{}) interface{} {
	query := req.(TSMQuery)
	switch query.query {
	case tsmQueryData:
		key := query.key
		v, ok := t.data[key]
		if !ok {
			return nil
		}
		return v
	case tsmQueryLatestRequest:
		client := query.key
		v, ok := t.latestRequestID[client]
		if !ok {
			return nil
		}
		return v
	}
	return nil
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
	newTSM, err := decodeTSMFromBytes(b)
	if err == nil {
		t.data = newTSM.data
		t.latestRequestID = newTSM.latestRequestID
	}
	return err
}

func decodeTSMFromBytes(b []byte) (TSM, error) {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	var newTSM TSM
	err := dec.Decode(&newTSM)
	return newTSM, err
}

// I really don't like the non-type-safe approach below, but I couldn't find a
// better way. There is an another approach that use anonymous functions as
// [TSMAction] but it can't be serialized.

func init() {
	gob.Register(TSMAction{})
}

type TSMAction struct {
	action tsmActionType
	target string
	source string // useless for [tsmActionSet] and [tsmActionIncr]
	value  int
	// request info
	clientID  string
	requestID uint32
}

type tsmActionType int

const (
	tsmActionSet tsmActionType = iota
	tsmActionIncr
	tsmActionMove
)

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
		action:    tsmActionSet,
		target:    key,
		value:     value,
		clientID:  b.clientID,
		requestID: b.newRequestID(),
	}
}

func (b *TSMActionBuilder) TSMActionIncrValue(key string, value int) TSMAction {
	return TSMAction{
		action:    tsmActionIncr,
		target:    key,
		value:     value,
		clientID:  b.clientID,
		requestID: b.newRequestID(),
	}
}

func (b *TSMActionBuilder) TSMActionMoveValue(source, target string, value int) TSMAction {
	return TSMAction{
		action:    tsmActionMove,
		source:    source,
		target:    target,
		value:     value,
		clientID:  b.clientID,
		requestID: b.newRequestID(),
	}
}

type TSMQuery struct {
	query tsmQueryType
	key   string
}

type tsmQueryType int

const (
	tsmQueryData tsmQueryType = iota
	tsmQueryLatestRequest
)

func NewTSMDataQuery(key string) TSMQuery {
	return TSMQuery{
		query: tsmQueryData,
		key:   key,
	}
}

func NewTSMLatestRequestQuery(clientID string) TSMQuery {
	return TSMQuery{
		query: tsmQueryLatestRequest,
		key:   clientID,
	}
}

func TSMIsSnapshotEqual(b1 []byte, b2 []byte) (bool, error) {
	tSM1, err := decodeTSMFromBytes(b1)
	if err != nil {
		return false, err
	}
	tSM2, err := decodeTSMFromBytes(b2)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(tSM1, tSM2), nil
}
