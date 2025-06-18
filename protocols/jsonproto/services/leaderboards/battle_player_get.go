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
	limit := int64(19) // The limit is always the page size, not the end rank.

	cursor, err := scoresCollection.Find(context.TODO(), bson.M{"battle_id": req.BattleID}, &options.FindOptions{
		Skip:  &startRank,
		Limit: &limit,
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
		if score.RoleID == 10 { // this indicates a band score
			bandPIDs = append(bandPIDs, score.OwnerPID)
		} else {
			playerPIDs = append(playerPIDs, score.OwnerPID)
		}
	}

	// grab console-prefixed usernames for players and band names for the bands
	playerNames, _ := db.GetConsolePrefixedUsernamesByPIDs(context.Background(), database, playerPIDs)
	bandNames, _ := db.GetBandNamesByBandIDs(context.Background(), database, bandPIDs)

	var res []BattlePlayerGetResponse
	var idx int64 = startRank + 1

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
				playerName := playerNames[score.OwnerPID]
				if playerName != "" {
					name = playerName + "'s Band"
				} else {
					name = "Unnamed Band"
				}
			} else {
				name = "Unnamed Player"
			}
		}

		res = append(res, BattlePlayerGetResponse{
			PID:          score.OwnerPID,
			Name:         name,
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

		idx++
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
