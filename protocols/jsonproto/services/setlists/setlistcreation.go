package setlists

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type SetlistCreationRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type SetlistCreationResponse struct {
	PID     int `json:"pid"`
	Creator int `json:"creator"`
}

type SetlistCreationService struct {
}

func (service SetlistCreationService) Path() string {
	return "misc/get_accounts_setlist_creation_status"
}

func (service SetlistCreationService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req SetlistCreationRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting setlist creation request")
		return "", err
	}

	res := []SetlistCreationResponse{{
		req.PID,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
