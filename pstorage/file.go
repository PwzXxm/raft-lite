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
	"github.com/sasha-s/go-deadlock"
	"os"
	"github.com/natefinch/atomic"
)

type FileBased struct {
	lock     deadlock.Mutex
	filepath string
}

// initialize file based persistent storage
func NewFileBasedPersistentStorage(filepath string) *FileBased {
	return &FileBased{filepath: filepath}
}

// save persistent storage to file
func (f *FileBased) Save(data interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return err
	}
	return atomic.WriteFile(f.filepath, &buf)
}

// load persistent storage from file
func (f *FileBased) Load(data interface{}) (bool, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if _, err := os.Stat(f.filepath); err == nil {
		// file exists
		f, err := os.Open(f.filepath)
		if err != nil {
			return false, err
		}
		defer f.Close()

		dec := gob.NewDecoder(f)
		return true, dec.Decode(data)

	} else if os.IsNotExist(err) {
		// file does not exist
		return false, nil
	} else {
		return false, err
	}
}
