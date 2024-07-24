package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

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

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for accomplishment leaderboards")
		return "", err
	}

	setlistsCollection := database.Collection("setlists")

	// first verify that the setlist with the req.BattleID exists
	var setlist models.Setlist
	err = setlistsCollection.FindOne(context.TODO(), bson.M{"setlist_id": req.BattleID}).Decode(&setlist)

	if err != nil {
		return marshaler.MarshalResponse(service.Path(), []BattleMaxrankGetResponse{{
			0,
		}})
	}

	// return the number of scores, aka the "max rank"
	res := []BattleMaxrankGetResponse{{
		len(setlist.BattleScores),
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
