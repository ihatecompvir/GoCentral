package marshaler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// builds the response for the service
func MarshalResponse(path string, obj interface{}) (string, error) {
	var names []string
	var types string
	var values interface{}

	v := reflect.ValueOf(obj)

	if v.Type().Kind() == reflect.Slice {
		len := v.Len()
		if len < 1 {
			panic("too small")
		}

		names, types = buildNamesAndTypesLists(v.Index(0))

		temp := make([]interface{}, len)
		for i := 0; i < len; i++ {
			temp[i] = buildValuesList(v.Index(i))
		}
		values = temp

	} else {
		names, types = buildNamesAndTypesLists(v)
		values = buildValuesList(v)
	}

	res := [][]interface{}{{path, types, names, values}}

	out, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func buildNamesAndTypesLists(v reflect.Value) ([]string, string) {
	if v.Type().Kind() == reflect.Slice {
		panic(fmt.Errorf("slices should not be passed here:%+v", v))
	}

	info := getMarshalTypeInfo(v.Type())

	names := make([]string, 0, len(info.fields))
	var types strings.Builder

	for i, fi := range info.fields {
		if fi.isSlice {
			for j := 0; j < v.Field(i).Len(); j++ {
				names = append(names, fmt.Sprintf("%s%03d", fi.slicePrefix, j))
				types.WriteString(fi.typeChar)
			}
		} else {
			names = append(names, fi.jsonTag)
			types.WriteString(fi.typeChar)
		}
	}

	return names, types.String()
}

func buildValuesList(v reflect.Value) []interface{} {
	if v.Type().Kind() == reflect.Slice {
		panic(fmt.Errorf("slices should not be passed here:%+v", v))
	}

	info := getMarshalTypeInfo(v.Type())

	values := make([]interface{}, 0, len(info.fields))

	for i, fi := range info.fields {
		if fi.isSlice {
			for j := 0; j < v.Field(i).Len(); j++ {
				values = append(values, v.Field(i).Index(j).Interface())
			}
		} else {
			values = append(values, v.Field(i).Interface())
		}
	}

	return values
}
