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

// Package pstorage contains three kinds of PersistentStorage
// 1. File based persistent storage
// 2. Memory based persistent storage
// 3. Hybird persistent storage
package pstorage

// PersistentStorage can save and load the data correspondingly
type PersistentStorage interface {
	Save(data interface{}) error
	Load(data interface{}) (bool, error)
}
