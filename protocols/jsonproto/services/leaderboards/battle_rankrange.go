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

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID000))

	if !validPIDres {
		log.Println("Client is attempting to get leaderboards without a valid server-assigned PID, rejecting call")
		return "", err
	}

	scoresCollection := database.Collection("scores")

	startRank := int64(req.StartRank - 1)
	endRank := int64((req.EndRank - req.StartRank) - 1)

	cursor, err := scoresCollection.Find(context.TODO(), bson.M{"battle_id": req.BattleID}, &options.FindOptions{
		Skip:  &startRank,
		Limit: &endRank,
		Sort:  bson.M{"score": -1},
	})

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	// just grab all the relevant scores into a single slice
	var scores []models.Score
	if err = cursor.All(context.Background(), &scores); err != nil {
		log.Println("Failed to decode all scores:", err)
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	// collect all the player and band PIDs we need to fetch
	playerPIDs := make([]int, 0)
	bandPIDs := make([]int, 0)
	for _, score := range scores {
		playerPIDs = append(playerPIDs, score.OwnerPID)
		if score.RoleID == 10 { // this indicates a band score
			bandPIDs = append(bandPIDs, score.OwnerPID)
		}
	}

	// grab console-prefixed usernames for players and band names for the bands
	playerNames, _ := db.GetConsolePrefixedUsernamesByPIDs(context.Background(), database, playerPIDs)
	nonPrefixedPlayerNames, _ := db.GetUsernamesByPIDs(context.Background(), database, playerPIDs)
	bandNames, _ := db.GetBandNamesByBandIDs(context.Background(), database, bandPIDs)

	var res []BattleRankRangeGetResponse
	var startIdx int = req.StartRank

	for _, score := range scores {
		var name string
		isBandScore := score.RoleID == 10

		// get the band or player name
		// since we prefetched the names this is a quick map lookup
		if isBandScore {
			name = bandNames[score.OwnerPID]
		} else {
			name = playerNames[score.OwnerPID]
		}

		// use fallback names if something could not be fetched or wasn't in the db
		if name == "" {
			if isBandScore {
				playerName := nonPrefixedPlayerNames[score.OwnerPID]
				if playerName != "" {
					name = playerName + "'s Band"
				} else {
					name = "Unnamed Band"
				}
			} else {
				name = "Unnamed Player"
			}
		}

		res = append(res, BattleRankRangeGetResponse{
			PID:          score.OwnerPID,
			Name:         name,
			DiffID:       score.DiffID,
			Rank:         startIdx,
			Score:        score.Score,
			IsPercentile: 0,
			InstMask:     score.InstrumentMask,
			NotesPct:     score.NotesPercent,
			IsFriend:     0,
			UnnamedBand:  0,
			PGUID:        "",
			ORank:        startIdx,
		})

		startIdx++
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
