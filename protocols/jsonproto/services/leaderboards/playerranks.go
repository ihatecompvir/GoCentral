package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PlayerranksGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	RoleID      int    `json:"role_id"`
	SongIDs     []int  `json:"song_idXXX"`
}

type PlayerranksGetResponse struct {
	SongID       int `json:"song_id"`
	Rank         int `json:"rank"`
	IsPercentile int `json:"is_percentile"`
}

type PlayerranksGetService struct {
}

func (service PlayerranksGetService) Path() string {
	return "leaderboards/playerranks/get"
}

func (service PlayerranksGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req PlayerranksGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for player rank range leaderboards")
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	scoresCollection := database.Collection("scores")

	res := []PlayerranksGetResponse{}

	for _, id := range req.SongIDs {
		var playerScore models.Score
		err = scoresCollection.FindOne(context.TODO(), bson.M{"song_id": id, "role_id": req.RoleID, "pid": req.PID}).Decode(&playerScore)
		if err != nil && err != mongo.ErrNoDocuments {
			log.Println(err)
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}

		if err == mongo.ErrNoDocuments {
			err = scoresCollection.FindOne(context.TODO(), bson.M{"song_id": id, "role_id": req.RoleID}, &options.FindOneOptions{
				Sort: bson.M{"score": -1},
			}).Decode(&playerScore)
			if err != nil && err != mongo.ErrNoDocuments {
				log.Println(err)
				return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
			}
		}

		playerScoreIdx, err := scoresCollection.CountDocuments(context.TODO(), bson.M{"song_id": id, "role_id": req.RoleID, "score": bson.M{"$gt": playerScore.Score}})
		if err != nil {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}

		var response PlayerranksGetResponse
		response.SongID = id
		response.Rank = int(playerScoreIdx)
		response.IsPercentile = 0

		res = append(res, response)
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}

}
