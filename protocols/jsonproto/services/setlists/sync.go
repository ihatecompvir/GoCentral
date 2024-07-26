package setlists

import (
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
	return marshaler.GenerateEmptyJSONResponse("setlists/sync"), nil
}
