package raft

import (
	"fmt"
	"reflect"
	"strconv"
)

// GetInfo
func (p *Peer) GetInfo() map[string]string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	mp := make(map[string]string)
	v := reflect.Indirect(reflect.ValueOf(p))
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		switch field.Kind() {
		case reflect.Int:
			mp[fieldName] = strconv.FormatInt(field.Int(), 10)
			break
		case reflect.Bool:
			mp[fieldName] = strconv.FormatBool(field.Bool())
			break
		case reflect.Ptr:
			if fieldName == "votedFor" {
				if field.IsNil() {
					mp[fieldName] = "nil"
				} else {
					mp[fieldName] = reflect.Indirect(field).String()
				}
			}
			break
		case reflect.Map:
			mp[fieldName] = getMapStr(field)
			break
		case reflect.Slice:
			mp[fieldName] = getSliceStr(field)
			break
		}
	}

	return mp
}

func getMapStr(field reflect.Value) string {
	var rst string = "{"
	it := field.MapRange()
	for it.Next() {
		key, val := it.Key(), it.Value()
		rst += "\n\t" + key.String() + ": " + strconv.FormatInt(val.Int(), 10)
	}
	rst += "}"
	return rst
}

func getSliceStr(field reflect.Value) string {
	var rst string = "["
	for i := 0; i < field.Len(); i++ {
		val := field.Index(i)

		if i != 0 {
			rst += ", "
		}

		switch val.Kind() {
		case reflect.String:
			rst += val.String()
			break
		case reflect.Struct:
			rst += "{"
			for j := 0; j < val.NumField(); j++ {
				if j != 0 {
					rst += ", "
				}

				subField := val.Field(j)
				subFieldName := val.Type().Field(j).Name

				rst += subFieldName + ": "
				switch subField.Kind() {
				case reflect.Int:
					rst += strconv.FormatInt(subField.Int(), 10)
					break
				case reflect.Interface:
					rst += fmt.Sprintf("%v", subField.Elem())
					break
				}
			}
			rst += "}"
			break
		}
	}
	rst += "]"
	return rst
}
