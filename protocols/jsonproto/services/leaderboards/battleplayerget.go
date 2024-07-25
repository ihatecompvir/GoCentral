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
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for battle leaderboards")
		return "", err
	}

	if req.LBMode == 1 {
		// friends leaderboard not supported
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	setlistsCollection := database.Collection("setlists")

	// FindOne the setlist from the DB with the associated battle ID
	var setlist models.Setlist
	err = setlistsCollection.FindOne(context.TODO(), bson.M{"setlist_id": req.BattleID}).Decode(&setlist)

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	res := []BattlePlayerGetResponse{}

	// sort battle scores by score
	sort.Slice(setlist.BattleScores, func(i, j int) bool {
		return setlist.BattleScores[i].Score > setlist.BattleScores[j].Score
	})

	// find the player's score idx in the sorted list
	// if the player has no scores, just start with the first score
	playerScoreIdx := 0
	for idx, score := range setlist.BattleScores {
		if score.PID == req.PID000 {
			playerScoreIdx = idx
			break
		}
	}

	// start and end idx must be in a window size of 20 otherwise the UI will act a bit buggy
	startIdx := (playerScoreIdx / 20) * 20
	endIdx := min(len(setlist.BattleScores), startIdx+20)

	for i := startIdx; i < endIdx; i++ {
		score := setlist.BattleScores[i]
		res = append(res, BattlePlayerGetResponse{
			PID:          score.PID,
			Score:        score.Score,
			DiffID:       0,
			Name:         db.GetConsolePrefixedUsernameForPID(score.PID),
			IsPercentile: 0,
			IsFriend:     0,
			InstMask:     0,
			NotesPct:     0,
			UnnamedBand:  0,
			PGUID:        "",
			Rank:         i + 1,
			ORank:        i + 1,
		})
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
