package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	db "rb3server/database"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BattleRankRangeGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
	BattleID    int    `json:"battle_id"`
	StartRank   int    `json:"start_rank"`
	EndRank     int    `json:"end_rank"`
}

type BattleRankRangeGetResponse struct {
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

type BattleRankRangeGetService struct {
}

func (service BattleRankRangeGetService) Path() string {
	return "leaderboards/battle_rankrange/get"
}

func (service BattleRankRangeGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req BattleRankRangeGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for battle rank range leaderboards")
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	scoresCollection := database.Collection("scores")

	startRank := int64(req.StartRank - 1)
	endRank := int64((req.EndRank - req.StartRank) - 1)

	// get cursor of scores filtered by song_id and role_id, with starting and ending ranks as indices (like scores[start_rank:end_rank]), sorted by biggest to smallest
	// so if start_rank is 1 and end_rank is 10, we get the top 10 scores
	cursor, err := scoresCollection.Find(context.TODO(), bson.M{"battle_id": req.BattleID}, &options.FindOptions{
		Skip:  &startRank,
		Limit: &endRank,
		Sort:  bson.M{"score": -1},
	})

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	var res []BattleRankRangeGetResponse

	var idx int = req.StartRank

	// iterate through the cursor and append each score to the response
	for cursor.Next(context.Background()) {
		var score models.Score
		err := cursor.Decode(&score)

		if err != nil {
			log.Println("Failed to decode score:", err)
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}

		isBandScore := score.RoleID == 10

		if isBandScore {
			res = append(res, BattleRankRangeGetResponse{
				PID:          score.OwnerPID,
				Name:         db.GetBandNameForBandID(score.OwnerPID),
				DiffID:       score.DiffID,
				Rank:         idx,
				Score:        score.Score,
				IsPercentile: 0,
				InstMask:     score.InstrumentMask,
				NotesPct:     score.NotesPercent,
				IsFriend:     0,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        idx,
			})
		} else {
			res = append(res, BattleRankRangeGetResponse{
				PID:          score.OwnerPID,
				Name:         db.GetConsolePrefixedUsernameForPID(score.OwnerPID),
				DiffID:       score.DiffID,
				Rank:         idx,
				Score:        score.Score,
				IsPercentile: 0,
				InstMask:     score.InstrumentMask,
				NotesPct:     score.NotesPercent,
				IsFriend:     0,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        idx,
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
