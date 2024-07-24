package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"sort"

	db "rb3server/database"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for battle leaderboards")
		return "", err
	}

	setlistsCollection := database.Collection("setlists")

	// FindOne the setlist from the DB with the associated battle ID
	var setlist models.Setlist
	err = setlistsCollection.FindOne(context.TODO(), bson.M{"setlist_id": req.BattleID}).Decode(&setlist)

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	res := []BattleRankRangeGetResponse{}

	// sort battle scores by score
	sort.Slice(setlist.BattleScores, func(i, j int) bool {
		return setlist.BattleScores[i].Score > setlist.BattleScores[j].Score
	})

	// get the scores in the range, and append them to the response
	for i := req.StartRank - 1; i < req.EndRank-1; i++ {
		if i >= len(setlist.BattleScores) {
			break
		}

		score := setlist.BattleScores[i]
		res = append(res, BattleRankRangeGetResponse{
			PID:          score.PID,
			Name:         db.GetUsernameForPID(score.PID),
			DiffID:       0,
			Rank:         i + 1,
			Score:        score.Score,
			IsPercentile: 0,
			InstMask:     0,
			NotesPct:     0,
			IsFriend:     0,
			UnnamedBand:  0,
			PGUID:        "",
			ORank:        i + 1,
		})
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
