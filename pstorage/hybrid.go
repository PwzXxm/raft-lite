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
	"os"
	"sync"
	"time"

	"github.com/natefinch/atomic"
	"github.com/sirupsen/logrus"
)

// HybridPStorage will save data into memory first, then write to disk
// periodically (so the IO is reduced). Please make sure [Stop] or [Flush] is
// called before quiting so the file on disk is up to date.

type Hybrid struct {
	lock     sync.Mutex
	filepath string
	data     []byte
	changed  bool
	logger   *logrus.Entry
	stop     chan struct{}
}

// initialize hybrid persistent storage
func NewHybridPersistentStorage(filepath string, interval time.Duration, logger *logrus.Entry) *Hybrid {
	h := new(Hybrid)
	h.stop = make(chan struct{})
	h.filepath = filepath
	h.logger = logger
	go func() {
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-ticker.C:
				err := h.Flush()
				if err != nil {
					if logger != nil {
						logger.Errorf("[HybridPStorage] Unable to flush: %v", err)
					}
				}
			case <-h.stop:
				ticker.Stop()
				return
			}
		}
	}()
	return h
}

// Save the hybrid version of save function
func (h *Hybrid) Save(data interface{}) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return err
	}
	h.changed = true
	h.data = buf.Bytes()
	return nil
}

// Load the hybrid version of load function
func (h *Hybrid) Load(data interface{}) (bool, error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if len(h.data) != 0 {
		dec := gob.NewDecoder(bytes.NewBuffer(h.data))
		return true, dec.Decode(data)
	}

	// try to load from file
	if _, err := os.Stat(h.filepath); err == nil {
		// file exists
		f, err := os.Open(h.filepath)
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

// Flush the hybrid version of flush
func (h *Hybrid) Flush() error {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.changed {
		err := atomic.WriteFile(h.filepath, bytes.NewBuffer(h.data))
		if err != nil {
			return err
		}
		h.changed = false
	}
	return nil
}

// Stop the hybrid version of stop
func (h *Hybrid) Stop() error {
	h.stop <- struct{}{}
	return h.Flush()
}
