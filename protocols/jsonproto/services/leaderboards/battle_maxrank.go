package leaderboard

import (
	"context"
	"log"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type BattleMaxrankGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	BattleID    int    `json:"battle_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
}

type BattleMaxrankGetResponse struct {
	MaxRank int `json:"max_rank"`
}

type BattleMaxrankGetService struct {
}

func (service BattleMaxrankGetService) Path() string {
	return "leaderboards/battle_maxrank/get"
}

func (service BattleMaxrankGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req BattleMaxrankGetRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID000))

	if !validPIDres {
		log.Println("Client is attempting to get leaderboards without a valid server-assigned PID, rejecting call")
		return "", err
	}

	scoresCollection := database.Collection("scores")

	numScores, err := scoresCollection.CountDocuments(context.TODO(), bson.M{"battle_id": req.BattleID})
	if err != nil {
		return marshaler.MarshalResponse(service.Path(), []BattleMaxrankGetResponse{{
			0,
		}})
	}

	res := []BattleMaxrankGetResponse{{
		int(numScores),
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
