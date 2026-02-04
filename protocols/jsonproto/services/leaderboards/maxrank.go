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

// lb type enum
// seems the game uses these with a song ID of 0 for things like the total score/ rock band 3 score leaderboards
const (
	LBTypeNormal     = 0
	LBTypeTotalScore = 1
	LBTypeRB3Only    = 2
)

type MaxrankGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	SongID      int    `json:"song_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	RoleID      int    `json:"role_id"`
	PID000      int    `json:"pid000"`
	LBType      int    `json:"lb_type"`
}

type MaxrankGetResponse struct {
	MaxRank int `json:"max_rank"`
}

type MaxrankGetService struct {
}

func (service MaxrankGetService) Path() string {
	return "leaderboards/maxrank/get"
}

func (service MaxrankGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req MaxrankGetRequest
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

	var numScores int64

	switch req.LBType {
	case LBTypeNormal:
		// Normal behavior - count documents matching song_id and role_id
		filter := bson.M{"song_id": req.SongID, "role_id": req.RoleID}
		numScores, err = scoresCollection.CountDocuments(context.TODO(), filter)
		if err != nil {
			return marshaler.MarshalResponse(service.Path(), []MaxrankGetResponse{{0}})
		}

	case LBTypeTotalScore, LBTypeRB3Only:
		// Aggregated leaderboards - count unique users with scores
		matchStage := bson.D{}

		// Exclude battle and setlist scores from total score calculations
		matchStage = append(matchStage, bson.E{Key: "battle_id", Value: 0})
		matchStage = append(matchStage, bson.E{Key: "setlist_id", Value: 0})

		// For RB3 Only, filter to song_id 1001-1106 (I think this is the full range)
		if req.LBType == LBTypeRB3Only {
			matchStage = append(matchStage, bson.E{Key: "song_id", Value: bson.D{{Key: "$gte", Value: 1001}, {Key: "$lte", Value: 1106}}})
		}

		// Filter by role_id if specified
		if req.RoleID > 0 {
			matchStage = append(matchStage, bson.E{Key: "role_id", Value: req.RoleID})
		}

		// Build pipeline
		pipeline := mongo.Pipeline{}
		if len(matchStage) > 0 {
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchStage}})
		}
		pipeline = append(pipeline,
			bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$pid"}}}},
			bson.D{{Key: "$count", Value: "total"}},
		)

		cursor, err := scoresCollection.Aggregate(context.TODO(), pipeline)
		if err != nil {
			return marshaler.MarshalResponse(service.Path(), []MaxrankGetResponse{{0}})
		}
		defer cursor.Close(context.TODO())

		var results []struct {
			Total int64 `bson:"total"`
		}
		if err := cursor.All(context.TODO(), &results); err != nil || len(results) == 0 {
			numScores = 0
		} else {
			numScores = results[0].Total
		}

	default:
		// Unknown LBType, return 0
		return marshaler.MarshalResponse(service.Path(), []MaxrankGetResponse{{0}})
	}

	res := []MaxrankGetResponse{{
		int(numScores),
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
