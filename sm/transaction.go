/*
 * Project: raft-lite
 * ---------------------
 * Authors:
 *   Minjian Chen 813534
 *   Shijie Liu   813277
 *   Weizhi Xu    752454
 *   Wenqing Xue  813044
 *   Zijun Chen   813190
 */

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

// TSM contains the transaction data and information about last request info
// This state machine is not thread-safe
type TSM struct {
	Data              map[string]int
	LatestRequestInfo map[string]TSMRequestInfo
}

// TSMRequestInfo TSM request stucture
type TSMRequestInfo struct {
	RequestID uint32
	Err       *string
}

// NewTransactionStateMachine new state machine
func NewTransactionStateMachine() *TSM {
	t := new(TSM)
	t.Reset()
	return t
}

// Reset reset state machine
func (t *TSM) Reset() {
	t.Data = make(map[string]int)
	t.LatestRequestInfo = make(map[string]TSMRequestInfo)
}

// ApplyAction apply action to state machine
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

// Query return the value of the given key.
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

// TakeSnapshot take current state machine to a snapshot
func (t *TSM) TakeSnapshot() ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)

	err := enc.Encode(t)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ResetWithSnapshot reset the state machine by the given snapshot
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

// TSMAction the state machine action structure
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

// GetRequestID get the request ID
func (a TSMAction) GetRequestID() uint32 {
	return a.RequestID
}

// TSMActionBuilder the state machine action builder structure
type TSMActionBuilder struct {
	clientID      string
	lastRequestID uint32
}

// NewTSMActionBuilder reture a new state machine action builder
func NewTSMActionBuilder(clientID string) *TSMActionBuilder {
	return &TSMActionBuilder{clientID: clientID, lastRequestID: rand.Uint32()}
}

func (b *TSMActionBuilder) newRequestID() uint32 {
	b.lastRequestID++
	return b.lastRequestID
}

// TSMActionSetValue the set value action
func (b *TSMActionBuilder) TSMActionSetValue(key string, value int) TSMAction {
	return TSMAction{
		Action:    tsmActionSet,
		Target:    key,
		Value:     value,
		ClientID:  b.clientID,
		RequestID: b.newRequestID(),
	}
}

// TSMActionIncrValue the increment value action
func (b *TSMActionBuilder) TSMActionIncrValue(key string, value int) TSMAction {
	return TSMAction{
		Action:    tsmActionIncr,
		Target:    key,
		Value:     value,
		ClientID:  b.clientID,
		RequestID: b.newRequestID(),
	}
}

// TSMActionMoveValue the move value action
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

// TSMQuery state machine query structure
type TSMQuery struct {
	Query tsmQueryType
	Key   string
}

type tsmQueryType int

const (
	tsmQueryData tsmQueryType = iota
	tsmQueryLatestRequest
)

// NewTSMDataQuery make new state machine data query
func NewTSMDataQuery(key string) TSMQuery {
	return TSMQuery{
		Query: tsmQueryData,
		Key:   key,
	}
}

// NewTSMLatestRequestQuery make new latest request query
func NewTSMLatestRequestQuery(clientID string) TSMQuery {
	return TSMQuery{
		Query: tsmQueryLatestRequest,
		Key:   clientID,
	}
}

// TSMIsSnapshotEqual check whether two state machine snapshot is equal
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

// TSMToStringHuman convert the state machine snapshot to human readable string
func TSMToStringHuman(b []byte) string {
	tSM, err := decodeTSMFromBytes(b)
	if err != nil {
		return fmt.Sprintf("Unable to decode snapshot: %v", err)
	}
	return fmt.Sprint(tSM)
}
