package battles

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type LimitCheckRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type LimitCheckResponse struct {
	Success int `json:"success"`
}

type LimitCheckService struct {
}

func (service LimitCheckService) Path() string {
	return "battles/limit/check"
}

func (service LimitCheckService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req LimitCheckRequest

	res := []LimitCheckResponse{{0}}

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting battle limit check")
		return "", err
	}

	return marshaler.MarshalResponse(service.Path(), res)
}
