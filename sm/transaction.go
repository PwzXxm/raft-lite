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
	Data            map[string]int
	LatestRequestID map[string]uint32
}

func NewTransactionStateMachine() *TSM {
	t := new(TSM)
	t.Data = make(map[string]int)
	t.LatestRequestID = make(map[string]uint32)
	return t
}

func (t *TSM) Reset() {
	t.Data = make(map[string]int)
	t.LatestRequestID = make(map[string]uint32)
}

func (t *TSM) ApplyAction(action interface{}) error {
	fmt.Printf("Apply actions\n")
	tsmAction := action.(TSMAction)
	// check duplicate
	lastID, ok := t.LatestRequestID[tsmAction.ClientID]
	if ok {
		if lastID == tsmAction.RequestID {
			// duplicate request, ignore it
			return nil
		}
	}
	t.LatestRequestID[tsmAction.ClientID] = tsmAction.RequestID

	// execute action
	switch tsmAction.Action {
	case tsmActionSet:
		fmt.Printf("Action1\n")
		t.Data[tsmAction.Target] = tsmAction.Value
	case tsmActionIncr:
		fmt.Printf("Action2\n")
		v, ok := t.Data[tsmAction.Target]
		if !ok {
			return errors.Errorf("invalid key: %v", tsmAction.Target)
		}
		t.Data[tsmAction.Target] = v + tsmAction.Value
	case tsmActionMove:
		fmt.Printf("Action3\n")
		sv, ok := t.Data[tsmAction.Source]
		if !ok {
			return errors.Errorf("invalid key for source: %v", tsmAction.Source)
		}
		tv, ok := t.Data[tsmAction.Target]
		if !ok {
			return errors.Errorf("invalid key for target: %v", tsmAction.Target)
		}
		t.Data[tsmAction.Source] = sv - tsmAction.Value
		t.Data[tsmAction.Target] = tv + tsmAction.Value
	}
	return nil
}

// return the value of the given key.
// key is string and return value is int
// TODO: Temp fakes func
func (t *TSM) Query(req interface{}) (interface{}, error) {
	query := req.(TSMQuery)
	switch query.Query {
	case tsmQueryData:
		key := query.Key
		v, ok := t.Data[key]
		if !ok {
			return nil, nil
		}
		return v, nil
	case tsmQueryLatestRequest:
		client := query.Key
		v, ok := t.LatestRequestID[client]
		if !ok {
			return nil, nil
		}
		return v, nil
	}
	return nil, nil
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
		t.Data = newTSM.Data
		t.LatestRequestID = newTSM.LatestRequestID
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
