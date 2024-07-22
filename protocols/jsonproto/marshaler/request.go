package marshaler

import (
	"encoding/json"
	"fmt"
	"reflect"
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

// converts json list to struct
func UnmarshalRequest(data string, out interface{}) error {
	normalized, err := normalizeJson(data)
	if err != nil {
		return err
	}

	outValue := reflect.ValueOf(out).Elem()
	outType := outValue.Type()

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

		// Handle dynamic fields (pid000, pid001, etc.)
		if strings.Contains(jsonTag, "XXX") {
			fieldNamePrefix := strings.TrimSuffix(jsonTag, "XXX")
			for key, value := range normalized {
				if strings.HasPrefix(key, fieldNamePrefix) {
					fieldValue := outValue.Field(i)
					if fieldValue.CanSet() {
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
					delete(normalized, key)
				}
			}
		}
	}

	return nil
}

func normalizeJson(data string) (map[string]interface{}, error) {
	var out [][]interface{}
	err := json.Unmarshal([]byte(data), &out)
	if err != nil {
		return nil, err
	}

	if len(out) != 1 {
		return nil, fmt.Errorf("received bad length:%d\n", len(out))
	}

	outer := out[0]
	if len(outer) != 2 {
		return nil, fmt.Errorf("received bad length:%d\n", len(outer))
	}

	inner, ok := outer[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad inner")
	}

	fields, ok := inner[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad fields")
	}

	fieldLen := len(fields)

	values, ok := inner[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("bad values")
	}

	m := make(map[string]interface{}, fieldLen)
	for i := 0; i < fieldLen; i++ {
		field, ok := fields[i].(string)
		if !ok {
			return nil, fmt.Errorf("bad field name")
		}
		m[field] = values[i]
	}

	return m, nil
}
