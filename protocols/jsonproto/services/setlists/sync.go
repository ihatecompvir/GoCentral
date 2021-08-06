package setlists

import (
	"rb3server/protocols/jsonproto/marshaler"

	"go.mongodb.org/mongo-driver/mongo"
)

type SetlistSyncRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid000"`
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

func (service SetlistSyncService) Handle(data string, database *mongo.Database) (string, error) {
	var req SetlistSyncRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	res := []SetlistSyncResponse{{
		req.PID,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
