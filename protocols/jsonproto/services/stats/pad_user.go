package stats

import (
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type StatsPadRequest struct {
	Name        string `json:"name"`
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	Pad0        string `json:"pad_0"`
}

type StatsPadResponse struct {
	PID     int `json:"pid"`
	Creator int `json:"creator"`
}

type StatsPadService struct {
}

func (service StatsPadService) Path() string {
	return "stats/pad_user"
}

func (service StatsPadService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req StatsPadRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	res := []StatsPadResponse{{
		123,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
