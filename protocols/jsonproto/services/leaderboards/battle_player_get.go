package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	db "rb3server/database"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BattlePlayerGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	BattleID    int    `json:"battle_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
	LBMode      int    `json:"lb_mode"`
	NumRows     int    `json:"num_rows"`
}

type BattlePlayerGetResponse struct {
	PID          int    `json:"pid"`
	Name         string `json:"name"`
	DiffID       int    `json:"diff_id"`
	Rank         int    `json:"rank"`
	Score        int    `json:"score"`
	IsPercentile int    `json:"is_percentile"`
	InstMask     int    `json:"inst_mask"`
	NotesPct     int    `json:"notes_pct"`
	IsFriend     int    `json:"is_friend"`
	UnnamedBand  int    `json:"unnamed_band"`
	PGUID        string `json:"pguid"`
	ORank        int    `json:"orank"`
}

type BattlePlayerGetService struct {
}

func (service BattlePlayerGetService) Path() string {
	return "leaderboards/battle_player/get"
}

func (service BattlePlayerGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req BattlePlayerGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID000))

	if !validPIDres {
		log.Println("Client is attempting to get leaderboards without a valid server-assigned PID, rejecting call")
		return "", err
	}

	scoresCollection := database.Collection("scores")

	var playerScore models.Score
	err = scoresCollection.FindOne(context.TODO(), bson.M{"battle_id": req.BattleID, "pid": req.PID000}).Decode(&playerScore)
	if err != nil && err != mongo.ErrNoDocuments {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	if err == mongo.ErrNoDocuments {
		err = scoresCollection.FindOne(context.TODO(), bson.M{"battle_id": req.BattleID}, &options.FindOneOptions{
			Sort: bson.M{"score": -1},
		}).Decode(&playerScore)
		if err != nil && err != mongo.ErrNoDocuments {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
	}

	playerScoreIdx, err := scoresCollection.CountDocuments(context.TODO(), bson.M{"battle_id": req.BattleID, "score": bson.M{"$gt": playerScore.Score}})
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	startRank := playerScoreIdx - (playerScoreIdx % 19)
	endRank := startRank + 19

	cursor, err := scoresCollection.Find(context.TODO(), bson.M{"battle_id": req.BattleID}, &options.FindOptions{
		Skip:  &startRank,
		Limit: &endRank,
		Sort:  bson.M{"score": -1},
	})

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	var res []BattlePlayerGetResponse

	var idx int64 = startRank + 1

	for cursor.Next(context.Background()) {
		var score models.Score
		err := cursor.Decode(&score)

		if err != nil {
			log.Println("Failed to decode score:", err)
			continue
		}

		isBandScore := score.RoleID == 10

		if isBandScore {
			res = append(res, BattlePlayerGetResponse{
				PID:          score.OwnerPID,
				Name:         db.GetBandNameForBandID(score.OwnerPID),
				DiffID:       score.DiffID,
				Rank:         int(idx),
				Score:        score.Score,
				IsPercentile: 0,
				InstMask:     score.InstrumentMask,
				NotesPct:     score.NotesPercent,
				IsFriend:     0,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        int(idx),
			})
		} else {
			res = append(res, BattlePlayerGetResponse{
				PID:          score.OwnerPID,
				Name:         db.GetConsolePrefixedUsernameForPID(score.OwnerPID),
				DiffID:       score.DiffID,
				Rank:         int(idx),
				Score:        score.Score,
				IsPercentile: 0,
				InstMask:     score.InstrumentMask,
				NotesPct:     score.NotesPercent,
				IsFriend:     0,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        int(idx),
			})
		}

		idx++
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
