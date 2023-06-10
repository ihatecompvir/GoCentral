package marshaler

import (
	"encoding/json"
	"fmt"
)

// gets the request name from the json data
func GetRequestName(data string) (string, error) {
	var out [][]interface{}
	err := json.Unmarshal([]byte(data), &out)
	if err != nil {
		return "", err
	}

	if len(out) != 1 {
		x := fmt.Errorf("received bad length:%d", len(out))
		return "", x
	}

	first := out[0]
	if len(first) != 2 {
		return "", fmt.Errorf("received bad length:%d", len(first))
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

	// convert map to json
	jsonString, err := json.Marshal(normalized)
	if err != nil {
		return err
	}

	// convert json to struct
	return json.Unmarshal(jsonString, &out)

}

func normalizeJson(data string) (map[string]interface{}, error) {

	var out [][]interface{}
	err := json.Unmarshal([]byte(data), &out)
	if err != nil {
		return nil, err
	}

	if len(out) != 1 {
		return nil, fmt.Errorf("received bad length:%d", len(out))
	}

	outer := out[0]
	if len(outer) != 2 {
		return nil, fmt.Errorf("received bad length:%d", len(outer))
	}

	inner, ok := outer[1].([]interface{})
	if !ok {
		panic("bad inner")
	}

	fields, ok := inner[0].([]interface{})
	if !ok {
		panic("bad fields")
	}

	fieldLen := len(fields)

	values, ok := inner[1].([]interface{})
	if !ok {
		panic("bad values")
	}

	m := make(map[string]interface{}, fieldLen)
	for i := 0; i < fieldLen; i++ {
		field, ok := fields[i].(string)
		if !ok {
			panic("bad field name")
		}
		m[field] = values[i]
	}

	return m, nil
}
