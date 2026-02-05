package marshaler

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// marshalFieldInfo holds pre-computed metadata for a struct field used during marshaling.
type marshalFieldInfo struct {
	jsonTag     string
	isSlice     bool
	slicePrefix string
	typeChar    string
}

type marshalTypeInfo struct {
	fields []marshalFieldInfo
}

var marshalCache sync.Map // reflect.Type -> *marshalTypeInfo

func getMarshalTypeInfo(t reflect.Type) *marshalTypeInfo {
	if cached, ok := marshalCache.Load(t); ok {
		return cached.(*marshalTypeInfo)
	}

	info := &marshalTypeInfo{
		fields: make([]marshalFieldInfo, t.NumField()),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fi := &info.fields[i]
		fi.jsonTag = field.Tag.Get("json")

		if field.Type.Kind() == reflect.Slice {
			fi.isSlice = true
			fi.slicePrefix = strings.TrimSuffix(fi.jsonTag, "XXX")
			switch field.Type.Elem().Kind() {
			case reflect.String:
				fi.typeChar = "s"
			case reflect.Int:
				fi.typeChar = "d"
			default:
				panic(fmt.Errorf("unsupported slice element type: %v", field.Type.Elem()))
			}
		} else {
			if fi.jsonTag == "" {
				fi.jsonTag = field.Name
			}
			switch field.Type.Kind() {
			case reflect.String:
				fi.typeChar = "s"
			case reflect.Int:
				fi.typeChar = "d"
			default:
				panic(fmt.Errorf("unsupported type: %v", field.Type))
			}
		}
	}

	marshalCache.Store(t, info)
	return info
}

// unmarshalFieldInfo holds pre-computed metadata for a struct field used during unmarshaling.
type unmarshalFieldInfo struct {
	index       int
	jsonTag     string
	isDynamic   bool
	fieldPrefix string
	kind        reflect.Kind
	elemKind    reflect.Kind
}

type unmarshalTypeInfo struct {
	fields []unmarshalFieldInfo
}

var unmarshalCache sync.Map // reflect.Type -> *unmarshalTypeInfo

func getUnmarshalTypeInfo(t reflect.Type) *unmarshalTypeInfo {
	if cached, ok := unmarshalCache.Load(t); ok {
		return cached.(*unmarshalTypeInfo)
	}

	info := &unmarshalTypeInfo{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		fi := unmarshalFieldInfo{
			index:   i,
			jsonTag: jsonTag,
			kind:    field.Type.Kind(),
		}

		if strings.Contains(jsonTag, "XXX") {
			fi.isDynamic = true
			fi.fieldPrefix = strings.TrimSuffix(jsonTag, "XXX")
		}

		if field.Type.Kind() == reflect.Slice {
			fi.elemKind = field.Type.Elem().Kind()
		}

		info.fields = append(info.fields, fi)
	}

	unmarshalCache.Store(t, info)
	return info
}
