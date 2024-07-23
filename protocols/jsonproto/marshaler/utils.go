package marshaler

import "encoding/json"

// CombineJSONMethods combines multiple json method responses into a singular one
// used by the game when the structure of the JSON needs to be different in each response (i.e. setlists, battles, etc.)
func CombineJSONMethods(jsonStrings []string) (string, error) {
	var finalOutput [][]interface{}

	for _, jsonString := range jsonStrings {
		var tempOutput [][]interface{}
		if err := json.Unmarshal([]byte(jsonString), &tempOutput); err != nil {
			return "", err
		}
		finalOutput = append(finalOutput, tempOutput...)
	}

	output, err := json.Marshal(finalOutput)
	if err != nil {
		return "", err
	}

	return string(output), nil
}
