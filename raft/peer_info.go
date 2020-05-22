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

package raft

import (
	"fmt"
)

// GetInfo returns an info map with string key and string value
func (p *Peer) GetInfo() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return fmt.Sprintf("%+v\nsnapshot:%+v\n\n", p, p.snapshot)
}
