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
		fmt.Printf("Action1\n")
		t.data[tsmAction.Target] = tsmAction.Value
	case tsmActionIncr:
		fmt.Printf("Action2\n")
		v, ok := t.data[tsmAction.Target]
		if !ok {
			return errors.Errorf("invalid key: %v", tsmAction.Target)
		}
		t.data[tsmAction.Target] = v + tsmAction.Value
	case tsmActionMove:
		fmt.Printf("Action3\n")
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
func (t *TSM) Query(req interface{}) interface{} {
	query := req.(TSMQuery)
	switch query.Query {
	case tsmQueryData:
		key := query.Key
		v, ok := t.data[key]
		if !ok {
			return nil
		}
		return v
	case tsmQueryLatestRequest:
		client := query.Key
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
