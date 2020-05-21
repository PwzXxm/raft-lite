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
	"os"
	"reflect"
	"strconv"
	"strings"
)

// GetInfo returns an info map of string key and string value
func (p *Peer) GetInfo() map[string]string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	mp := make(map[string]string)
	v := reflect.Indirect(reflect.ValueOf(p))
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		if fieldName == "mutex" || fieldName == "logger" || fieldName == "persistentStorage" {
			continue
		}
		mp[fieldName] = getFieldStr(v.Field(i), 2) + "\n"
	}

	return mp
}

// getFieldStr returns a field string of given values
func getFieldStr(v reflect.Value, lvl int) string {
	lvl += 2

	switch v.Kind() {
	case reflect.Int:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Chan:
		// ignore channel
		return "chan(ignored)"
	case reflect.Ptr:
		if v.IsNil() {
			return "nil"
		} else {
			return reflect.Indirect(v).String()
		}
	case reflect.String:
		return v.String()
	case reflect.Map:
		return getMapStr(v, lvl)
	case reflect.Slice:
		return getSliceStr(v, lvl)
	case reflect.Interface:
		return fmt.Sprintf("%v", v.Elem())
	case reflect.Struct:
		var rst string = ""
		rst += "{"
		for j := 0; j < v.NumField(); j++ {
			if j != 0 {
				rst += ", "
			}

			subField := v.Field(j)
			subFieldName := v.Type().Field(j).Name

			rst += subFieldName + ": "
			rst += getFieldStr(subField, lvl)
		}
		rst += "}"
		return rst
	}

	fmt.Fprintf(os.Stderr, "Unrecognized type %v", v.Type().String())
	return ""
}

// getMapStr returns a map string of given values
func getMapStr(field reflect.Value, lvl int) string {
	var rst string = "{"
	it := field.MapRange()
	for it.Next() {
		key, val := it.Key(), it.Value()
		rst += "\n" + strings.Repeat(" ", lvl) + getFieldStr(key, lvl) + ": " + getFieldStr(val, lvl)
	}
	rst += "\n" + strings.Repeat(" ", lvl-2) + "}"
	return rst
}

// getSliceStr returns a slide string of given values
func getSliceStr(field reflect.Value, lvl int) string {
	var rst string = "["
	for i := 0; i < field.Len(); i++ {
		val := field.Index(i)
		if i != 0 {
			rst += ", "
		}
		rst += getFieldStr(val, lvl)
	}
	rst += "]"
	return rst
}
