package setlists

import (
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
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

func (service SetlistSyncService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req SetlistSyncRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		users := database.Collection("users")
		var user models.User
		err = users.FindOne(nil, bson.M{"pid": req.PID}).Decode(&user)
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting setlist synchronization")
		log.Println("Database PID : ", user.PID)
		client.SetPlayerID(user.PID)
		log.Println("Client PID : ", client.PlayerID())
	}

	res := []SetlistSyncResponse{{
		req.PID,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
