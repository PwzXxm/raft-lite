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

package pstorage

import (
	"bytes"
	"encoding/gob"
	"sync"
)

type MemoryBased struct {
	lock sync.Mutex
	data []byte
}

// initialize memory based persistent storage
func NewMemoryBasedPersistentStorage() *MemoryBased {
	return &MemoryBased{}
}

// save persistent storage to memory
func (f *MemoryBased) Save(data interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err == nil {
		f.data = buf.Bytes()
	}

	return err
}

// load persistent storage from memory
func (f *MemoryBased) Load(data interface{}) (bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if len(f.data) == 0 {
		return false, nil
	}

	dec := gob.NewDecoder(bytes.NewBuffer(f.data))

	return true, dec.Decode(data)
}
