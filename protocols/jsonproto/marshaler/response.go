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
	tp := v.Type()

	if tp.Kind() == reflect.Slice {
		panic(fmt.Errorf("slices should not be passed here:%+v", v))
	}

	fieldCount := v.NumField()

	names := make([]string, 0, fieldCount)
	var types strings.Builder

	for i := 0; i < fieldCount; i++ {
		field := tp.Field(i)

		if field.Type.Kind() == reflect.Slice {
			for j := 0; j < v.Field(i).Len(); j++ {
				name := fmt.Sprintf("%s%03d", strings.TrimSuffix(field.Tag.Get("json"), "XXX"), j)
				names = append(names, name)
				switch v.Field(i).Index(j).Interface().(type) {
				case string:
					types.WriteString("s")
				case int:
					types.WriteString("d")
				default:
					panic(fmt.Errorf("unsupported type in slice:%+v\n", field.Type))
				}
			}
		} else {
			name := field.Tag.Get("json")
			if name == "" {
				name = field.Name
			}

			names = append(names, name)

			switch v.Field(i).Interface().(type) {
			case string:
				types.WriteString("s")
			case int:
				types.WriteString("d")
			default:
				panic(fmt.Errorf("unsupported type:%+v\n", field.Type))
			}
		}
	}

	return names, types.String()
}

func buildValuesList(v reflect.Value) []interface{} {
	if v.Type().Kind() == reflect.Slice {
		panic(fmt.Errorf("slices should not be passed here:%+v", v))
	}

	fieldCount := v.NumField()

	values := make([]interface{}, 0, fieldCount)

	for i := 0; i < fieldCount; i++ {
		if v.Field(i).Type().Kind() == reflect.Slice {
			for j := 0; j < v.Field(i).Len(); j++ {
				values = append(values, v.Field(i).Index(j).Interface())
			}
		} else {
			values = append(values, v.Field(i).Interface())
		}
	}

	return values
}
