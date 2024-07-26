package setlists

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type SetlistSyncRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PIDs        []int  `json:"pidXXX"`
}

type SetlistSyncResponse struct {
	PID     int `json:"pid"`
	Creator int `json:"creator"`
}

type SetlistSyncService struct {
}

func (service SetlistSyncService) Path() string {
	return "setlists/sync"
}

func (service SetlistSyncService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req SetlistSyncRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PIDs[0] == 0 {
		// it is a machine, not a player, so just respond with a blank response
		return marshaler.MarshalResponse(service.Path(), []SetlistSyncResponse{{0, 0}})
	}
	if req.PIDs[0] != int(client.PlayerID()) {
		log.Printf("Client-supplied PID %v did not match server-assigned PID %v, rejecting setlist synchronization", req.PIDs[0], client.PlayerID())
		return "", err
	}

	res := []SetlistSyncResponse{{
		req.PIDs[0],
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
