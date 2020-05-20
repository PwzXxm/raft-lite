package sm

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"reflect"

	"github.com/pkg/errors"
)

func init() {
	gob.Register(TSMAction{})
	gob.Register(TSMRequestInfo{})
}

// This state machine is not thread-safe
type TSM struct {
	Data              map[string]int
	LatestRequestInfo map[string]TSMRequestInfo
}

type TSMRequestInfo struct {
	RequestID uint32
	Err       *string
}

func NewTransactionStateMachine() *TSM {
	t := new(TSM)
	t.Reset()
	return t
}

func (t *TSM) Reset() {
	t.Data = make(map[string]int)
	t.LatestRequestInfo = make(map[string]TSMRequestInfo)
}

func (t *TSM) ApplyAction(action interface{}) error {
	tsmAction := action.(TSMAction)
	// check duplicate
	lastRequest, ok := t.LatestRequestInfo[tsmAction.ClientID]
	if ok {
		if lastRequest.RequestID == tsmAction.RequestID {
			// duplicate request, ignore it
			return nil
		}
	}

	var errStr string
	// execute action
	switch tsmAction.Action {
	case tsmActionSet:
		if tsmAction.Value < 0 {
			errStr = "negative value is not allowed"
			break
		}
		t.Data[tsmAction.Target] = tsmAction.Value
	case tsmActionIncr:
		v, ok := t.Data[tsmAction.Target]
		if !ok {
			errStr = fmt.Sprintf("invalid key: [%v]", tsmAction.Target)
			break
		}
		if v+tsmAction.Value < 0 {
			errStr = fmt.Sprintf("the value [%v] of key [%v] will be negative after this request, which is not allowed",
				v, tsmAction.Target)
			break
		}
		t.Data[tsmAction.Target] = v + tsmAction.Value
	case tsmActionMove:
		sv, ok := t.Data[tsmAction.Source]
		if !ok {
			errStr = fmt.Sprintf("invalid key for source: [%v]", tsmAction.Source)
			break
		}
		tv, ok := t.Data[tsmAction.Target]
		if !ok {
			errStr = fmt.Sprintf("invalid key for target: [%v]", tsmAction.Target)
			break
		}
		if sv-tsmAction.Value < 0 {
			errStr = fmt.Sprintf("the value [%v] in key [%v] will be negative after this request, which is not allowed",
				sv, tsmAction.Source)
			break
		}
		if tv+tsmAction.Value < 0 {
			errStr = fmt.Sprintf("the value [%v] in key [%v] will be negative after this request, which is not allowed",
				tv, tsmAction.Target)
			break
		}
		t.Data[tsmAction.Source] = sv - tsmAction.Value
		t.Data[tsmAction.Target] = tv + tsmAction.Value
	default:
		errStr = "invalid request"
	}
	var errStrp *string
	var err error
	if errStr != "" {
		errStrp = &errStr
		err = errors.New(errStr)
	}
	t.LatestRequestInfo[tsmAction.ClientID] = TSMRequestInfo{
		RequestID: tsmAction.RequestID,
		Err:       errStrp,
	}
	return err
}

// return the value of the given key.
// key is string and return value is int
func (t *TSM) Query(req interface{}) (interface{}, error) {
	query := req.(TSMQuery)
	switch query.Query {
	case tsmQueryData:
		key := query.Key
		v, ok := t.Data[key]
		if !ok {
			return nil, errors.New("key does not exist")
		}
		return v, nil
	case tsmQueryLatestRequest:
		client := query.Key
		v, ok := t.LatestRequestInfo[client]
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
	newTSM, err := decodeTSMFromBytes(b)
	if err == nil {
		t.Data = newTSM.Data
		t.LatestRequestInfo = newTSM.LatestRequestInfo
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

func TSMToStringHuman(b []byte) string {
	tSM, err := decodeTSMFromBytes(b)
	if err != nil {
		return fmt.Sprintf("Unable to decode snapshot: %v", err)
	}
	return fmt.Sprint(tSM)
}
