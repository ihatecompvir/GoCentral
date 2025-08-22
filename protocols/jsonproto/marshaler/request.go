package marshaler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// gets the request name from the json data
func GetRequestName(data string) (string, error) {
	var out [][]interface{}
	err := json.Unmarshal([]byte(data), &out)
	if err != nil {
		return "", err
	}

	if len(out) != 1 {
		return "", fmt.Errorf("received bad length:%d\n", len(out))
	}

	first := out[0]
	if len(first) != 2 {
		return "", fmt.Errorf("received bad length:%d\n", len(first))
	}

	name, ok := first[0].(string)
	if !ok {
		panic("bad name")
	}

	return name, nil
}

// extracts the index from dynamic fields (pid000, pid001, etc.)
func extractIndex(key, prefix string) (int, error) {
	re := regexp.MustCompile(`\d+$`)
	indexStr := re.FindString(key[len(prefix):])
	var index int
	_, err := fmt.Sscanf(indexStr, "%d", &index)
	return index, err
}

// Converts JSON list to struct
func UnmarshalRequest(data string, out interface{}) error {
	normalized, err := normalizeJson(data)
	if err != nil {
		return err
	}

	outValue := reflect.ValueOf(out).Elem()
	outType := outValue.Type()

	dynamicFields := make(map[string]map[int]interface{})

	for i := 0; i < outType.NumField(); i++ {
		field := outType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Handle static fields
		if value, ok := normalized[jsonTag]; ok {
			fieldValue := outValue.Field(i)
			if fieldValue.CanSet() {
				switch fieldValue.Kind() {
				case reflect.Int:
					if v, ok := value.(float64); ok {
						fieldValue.SetInt(int64(v))
					}
				case reflect.String:
					if v, ok := value.(string); ok {
						fieldValue.SetString(v)
					}
				default:
					fieldValue.Set(reflect.ValueOf(value))
				}
			}
			delete(normalized, jsonTag)
		}

		// Handle dynamic fields (pidXXX, role_idXXX, etc.)
		if strings.Contains(jsonTag, "XXX") {
			fieldNamePrefix := strings.TrimSuffix(jsonTag, "XXX")
			for key, value := range normalized {
				if strings.HasPrefix(key, fieldNamePrefix) {
					index, err := extractIndex(key, fieldNamePrefix)
					if err != nil {
						continue
					}
					if _, ok := dynamicFields[jsonTag]; !ok {
						dynamicFields[jsonTag] = make(map[int]interface{})
					}
					dynamicFields[jsonTag][index] = value
					delete(normalized, key)
				}
			}
		}
	}

	for i := 0; i < outType.NumField(); i++ {
		field := outType.Field(i)
		jsonTag := field.Tag.Get("json")
		if values, ok := dynamicFields[jsonTag]; ok {
			fieldValue := outValue.Field(i)
			if fieldValue.Kind() == reflect.Slice {
				keys := make([]int, 0, len(values))
				for k := range values {
					keys = append(keys, k)
				}
				sort.Ints(keys)
				for _, k := range keys {
					value := values[k]
					switch fieldValue.Type().Elem().Kind() {
					case reflect.Int:
						if v, ok := value.(float64); ok {
							fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(int(v))))
						}
					case reflect.String:
						if v, ok := value.(string); ok {
							fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(v)))
						}
					default:
						fieldValue.Set(reflect.Append(fieldValue, reflect.ValueOf(value)))
					}
				}
			}
		}
	}

	return nil
}

func normalizeJson(data string) (map[string]interface{}, error) {
	var out [][]interface{}
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil, err
	}

	if len(out) != 1 {
		return nil, fmt.Errorf("received bad length:%d", len(out))
	}

	outer := out[0]
	if len(outer) != 2 {
		return nil, fmt.Errorf("received bad outer length:%d", len(outer))
	}

	inner, ok := outer[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad inner")
	}
	if len(inner) < 2 {
		return nil, fmt.Errorf("bad inner length:%d", len(inner))
	}

	fields, ok := inner[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad fields")
	}
	values, ok := inner[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad values")
	}

	flen, vlen := len(fields), len(values)
	if flen != vlen {
		return nil, fmt.Errorf("fields/values length mismatch: %d vs %d", flen, vlen)
	}

	m := make(map[string]interface{}, flen)
	for i := 0; i < flen; i++ {
		field, ok := fields[i].(string)
		if !ok {
			return nil, fmt.Errorf("bad field name at index %d", i)
		}
		m[field] = values[i]
	}

	return m, nil
}
