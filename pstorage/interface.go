/*
 * Project: raft-lite
 * ---------------------
 * Authors:
 * Minjian Chen 813534
 * Shijie Liu   813277
 * Weizhi Xu    752454
 * Wenqing Xue  813044
 * Zijun Chen   813190
 */

package pstorage

type PersistentStorage interface {
	Save(data interface{}) error
	Load(data interface{}) (bool, error)
}
