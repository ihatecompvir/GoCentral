package leaderboard

import (
	"context"
	"log"
	db "rb3server/database"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PlayerGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	SongID      int    `json:"song_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
	RoleID      int    `json:"role_id"`
	LBType      int    `json:"lb_type"`
	LBMode      int    `json:"lb_mode"`
	NumRows     int    `json:"num_rows"`
}

type PlayerGetResponse struct {
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

type PlayerGetService struct {
}

func (service PlayerGetService) Path() string {
	return "leaderboards/player/get"
}

func (service PlayerGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req PlayerGetRequest

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
	err = scoresCollection.FindOne(context.TODO(), bson.M{"song_id": req.SongID, "role_id": req.RoleID, "pid": req.PID000}).Decode(&playerScore)
	if err != nil && err != mongo.ErrNoDocuments {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	if err == mongo.ErrNoDocuments {
		err = scoresCollection.FindOne(context.TODO(), bson.M{"song_id": req.SongID, "role_id": req.RoleID}, &options.FindOneOptions{
			Sort: bson.M{"score": -1},
		}).Decode(&playerScore)
		if err != nil && err != mongo.ErrNoDocuments {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
	}

	playerScoreIdx, err := scoresCollection.CountDocuments(context.TODO(), bson.M{"song_id": req.SongID, "role_id": req.RoleID, "score": bson.M{"$gt": playerScore.Score}})
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	startRank := playerScoreIdx - (playerScoreIdx % 19)
	endRank := startRank + 19

	cursor, err := scoresCollection.Find(context.TODO(), bson.M{"song_id": req.SongID, "role_id": req.RoleID}, &options.FindOptions{
		Skip:  &startRank,
		Limit: &endRank,
		Sort:  bson.M{"score": -1},
	})

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	var res []PlayerGetResponse

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
			res = append(res, PlayerGetResponse{
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
			res = append(res, PlayerGetResponse{
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
