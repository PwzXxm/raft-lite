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

// Package utils have all ad-hoc functions such as finding min, max and random
package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Min find the minimum value of the int type
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max find the maximum value of the int type
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Random an integer within the range, both sides are inclusive
func Random(a, b int) int {
	return rand.Intn(b-a+1) + a
}

// RandomFloat returns a random float, left inclusive
func RandomFloat(a, b float64) float64 {
	return a + rand.Float64()*(b-a)
}

// RandomTime returns a random time duration, both inclusive
func RandomTime(a, b time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(b-a+1)) + int64(a))
}

// RandomBool returns a boolean value based on a given probablity
func RandomBool(prob float64) bool {
	return rand.Float64() < prob
}

// PrintUsage print usage messages to the stdin
func PrintUsage(m map[string]string) {
	fmt.Println("Usage: <cmd> <args> ...")
	var longest int = -1

	rst := make([]string, 0, len(m))
	for cmd := range m {
		l := len(cmd)
		if longest < l {
			longest = l
		}
		rst = append(rst, cmd)
	}

	sort.Strings(rst)

	fmt.Printf("\nCommands:\n")
	for _, cmd := range rst {
		fmt.Printf("\t%-*v %v\n", longest, cmd, m[cmd])
	}
}

// ReadClientFromJSON reads JSON file and unmarshal
func ReadClientFromJSON(v interface{}, filepath string) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
