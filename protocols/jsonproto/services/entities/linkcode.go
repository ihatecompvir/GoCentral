package entities

import (
	"rb3server/protocols/jsonproto/marshaler"

	"go.mongodb.org/mongo-driver/mongo"
)

type GetLinkcodeRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type GetLinkcodeResponse struct {
	Code string `json:"code"`
}

type GetLinkcodeService struct {
}

func (service GetLinkcodeService) Path() string {
	return "entities/linkcode/get"
}

func (service GetLinkcodeService) Handle(data string, database *mongo.Database) (string, error) {
	var req GetLinkcodeRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	// Spoof account linking status, 12345 pid
	res := []GetLinkcodeResponse{{
		"Not supported",
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
