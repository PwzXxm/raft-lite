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
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
)

type testStruct struct {
	Str string
	Int int
}

// test memory based persistent storage initialization
func TestMemoryBased(t *testing.T) {
	m := NewMemoryBasedPersistentStorage()
	testPersistentStorage(t, m)
}

// test file based persistent storage initialization
func TestFileBased(t *testing.T) {
	file, err := ioutil.TempFile("", "tests")
	if err != nil {
		log.Fatal(err)
	}
	// a little hacky
	os.Remove(file.Name())
	defer os.Remove(file.Name())

	m := NewFileBasedPersistentStorage(file.Name())
	testPersistentStorage(t, m)
}

// test hybrid based persistent storage initialization
func TestHybridBased(t *testing.T) {
	file, err := ioutil.TempFile("", "tests")
	if err != nil {
		log.Fatal(err)
	}
	// a little hacky
	os.Remove(file.Name())
	defer os.Remove(file.Name())

	m := NewHybridPersistentStorage(file.Name(), time.Second, nil)
	testPersistentStorage(t, m)
	_ = m.Stop()
}

// check with errors
func checkNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Shouldn't be an error: %+v", errors.WithStack(err))
	}
}

// test save and load persistent storage
func testPersistentStorage(t *testing.T, p PersistentStorage) {
	var data testStruct
	hasData, err := p.Load(&data)
	checkNoError(t, err)
	if hasData {
		t.Error("Should be empty.")
	}
	data.Int = 123
	data.Str = "ABC"
	// test save
	err = p.Save(data)
	checkNoError(t, err)
	// test load
	var data2 testStruct
	hasData, err = p.Load(&data2)
	checkNoError(t, err)
	if !hasData {
		t.Error("Shouldn't be empty.")
	}
	if data != data2 {
		t.Errorf("Data should be the same, data1: %v, data2: %v", data, data2)
	}
}
