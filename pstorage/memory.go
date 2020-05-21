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

func NewMemoryBasedPersistentStorage() *MemoryBased {
	return &MemoryBased{}
}

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

func (f *MemoryBased) Load(data interface{}) (bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if len(f.data) == 0 {
		return false, nil
	}

	dec := gob.NewDecoder(bytes.NewBuffer(f.data))

	return true, dec.Decode(data)
}
