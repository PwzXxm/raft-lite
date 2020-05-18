package utils

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
	"io/ioutil"
	"encoding/json"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Random an integer within the range
func Random(a, b int) int {
	return rand.Intn(b-a+1) + a
}

func RandomFloat(a, b float64) float64 {
	return a + rand.Float64()*(b-a)
}

func RandomTime(a, b time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(b-a+1)) + int64(a))
}

func RandomBool(prob float64) bool {
	return rand.Float64() < prob
}

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

func ReadClientFromJSON(v interface{}, filepath string) (interface{}, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal(data, &v)
	if err != nil {
		return v, err
	}
	return v, nil
}
