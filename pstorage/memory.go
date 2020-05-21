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
	data bytes.Buffer
}

// initial object
func NewMemoryBasedPersistentStorage() *MemoryBased {
	return &MemoryBased{}
}

// save persistent storage to memory
func (f *MemoryBased) Save(data interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	enc := gob.NewEncoder(&f.data)
	return enc.Encode(data)
}

// load persistent storage from memory
func (f *MemoryBased) Load(data interface{}) (bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.data.Len() == 0 {
		return false, nil
	}
	dec := gob.NewDecoder(&f.data)
	return true, dec.Decode(data)
}
