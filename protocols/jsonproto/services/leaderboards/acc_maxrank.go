package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccMaxrankGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	AccID       string `json:"acc_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
}

type AccMaxrankGetResponse struct {
	MaxRank int `json:"max_rank"`
}

type AccMaxrankGetService struct {
}

func (service AccMaxrankGetService) Path() string {
	return "leaderboards/acc_maxrank/get"
}

func (service AccMaxrankGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccMaxrankGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID000))

	if !validPIDres {
		log.Println("Client is attempting to get leaderboards without a valid server-assigned PID, rejecting call")
		return "", err
	}

	accomplishmentsCollection := database.Collection("accomplishments")

	var accomplishments models.Accomplishments
	err = accomplishmentsCollection.FindOne(context.TODO(), bson.M{"acc_id": req.AccID}).Decode(&accomplishments)

	if err != nil {
		return marshaler.MarshalResponse(service.Path(), []AccMaxrankGetResponse{{
			0,
		}})
	}

	// return the number of scores, aka the "max rank"
	res := []AccMaxrankGetResponse{{
		len(getAccomplishmentField(req.AccID, accomplishments)),
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
