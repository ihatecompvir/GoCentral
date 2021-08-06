package entities

import (
	"rb3server/protocols/jsonproto/marshaler"

	"go.mongodb.org/mongo-driver/mongo"
)

type CharacterUpdateRequest struct {
	Name        string `json:"name"`
	Region      string `json:"region"`
	Flags       int    `json:"flags"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	GUID        string `json:"guid"`
	CharData    string `json:"char_data"`
}

type CharacterUpdateResponse struct {
	Test int `json:"test"`
}

type CharacterUpdateService struct {
}

func (service CharacterUpdateService) Path() string {
	return "entities/character/update"
}

func (service CharacterUpdateService) Handle(data string, database *mongo.Database) (string, error) {
	var req CharacterUpdateRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	// Spoof account linking status, 12345 pid
	res := []CharacterUpdateResponse{{
		1,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
