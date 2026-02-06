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

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to get leaderboards without a valid server-assigned PID, rejecting call")
		return "", err
	}

	scoresCollection := database.Collection("scores")

	res := []PlayerranksGetResponse{}

	if len(req.SongIDs) == 0 {
		log.Println("No song IDs provided for playerranks get request")
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	playerScoresFilter := bson.M{
		"pid":     req.PID,
		"role_id": req.RoleID,
		"song_id": bson.M{"$in": req.SongIDs},
	}
	cursor, err := scoresCollection.Find(context.TODO(), playerScoresFilter)
	if err != nil {
		log.Println(err)
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}
	defer cursor.Close(context.TODO())

	playerScoresMap := make(map[int]int)
	for cursor.Next(context.Background()) {
		var score models.Score
		cursor.Decode(&score)
		playerScoresMap[score.SongID] = score.Score
	}

	for _, id := range req.SongIDs {
		playerScore := playerScoresMap[id]
		rank, err := scoresCollection.CountDocuments(context.TODO(), bson.M{"song_id": id, "role_id": req.RoleID, "score": bson.M{"$gt": playerScore}})
		if err != nil {
			log.Println("Could not count documents for rank:", err)
			// just say theyre number 1 lol
			rank = 0
		}

		res = append(res, PlayerranksGetResponse{
			SongID:       id,
			Rank:         int(rank + 1),
			IsPercentile: 0,
		})
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
