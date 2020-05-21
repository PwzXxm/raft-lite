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

func (h *Hybrid) Stop() error {
	h.stop <- struct{}{}
	return h.Flush()
}
