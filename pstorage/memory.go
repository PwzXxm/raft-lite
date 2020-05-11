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

func NewMemoryBasedPersistentStorage() *MemoryBased {
	return &MemoryBased{}
}

func (f *MemoryBased) Save(data interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	enc := gob.NewEncoder(&f.data)
	return enc.Encode(data)
}

func (f *MemoryBased) Load(data interface{}) (bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.data.Len() == 0 {
		return false, nil
	}
	dec := gob.NewDecoder(&f.data)
	return true, dec.Decode(data)
}
