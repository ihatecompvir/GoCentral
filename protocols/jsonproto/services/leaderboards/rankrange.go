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

type RankRangeGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	SongID      int    `json:"song_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
	RoleID      int    `json:"role_id"`
	LBType      int    `json:"lb_type"`
	StartRank   int    `json:"start_rank"`
	EndRank     int    `json:"end_rank"`
}

type RankRangeGetResponse struct {
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

type RankRangeGetService struct {
}

func (service RankRangeGetService) Path() string {
	return "leaderboards/rankrange/get"
}

func (service RankRangeGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req RankRangeGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID000))

	if !validPIDres {
		log.Println("Client is attempting to get leaderboards without a valid server-assigned PID, rejecting call")
		return "", err
	}

	// fetch friends list for IsFriend marking and friends leaderboard filtering
	friendsMap, _ := db.GetFriendsForPID(context.Background(), database, req.PID000)

	scoresCollection := database.Collection("scores")

	startRank := int64(req.StartRank - 1)
	numRows := int64(req.EndRank - req.StartRank + 1)

	var scores []models.Score
	var aggregatedScores []struct {
		PID        int `bson:"_id"`
		TotalScore int `bson:"totalScore"`
	}
	isAggregated := req.LBType == LBTypeTotalScore || req.LBType == LBTypeRB3Only

	if isAggregated {
		matchStage := bson.D{}

		// Exclude battle and setlist scores from total score calculations
		matchStage = append(matchStage, bson.E{Key: "battle_id", Value: bson.D{{Key: "$not", Value: bson.D{{Key: "$gt", Value: 0}}}}})
		matchStage = append(matchStage, bson.E{Key: "setlist_id", Value: bson.D{{Key: "$not", Value: bson.D{{Key: "$gt", Value: 0}}}}})

		// For RB3 Only, filter to song_id 1001-1106 (I think this is the full range)
		if req.LBType == LBTypeRB3Only {
			matchStage = append(matchStage, bson.E{Key: "song_id", Value: bson.D{{Key: "$gte", Value: 1001}, {Key: "$lte", Value: 1106}}})
		}

		matchStage = append(matchStage, bson.E{Key: "role_id", Value: req.RoleID})

		// Build aggregation pipeline
		// i genuinely hate mongo syntax
		pipeline := mongo.Pipeline{}
		if len(matchStage) > 0 {
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchStage}})
		}
		pipeline = append(pipeline,
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$pid"},
				{Key: "totalScore", Value: bson.D{{Key: "$sum", Value: "$score"}}},
			}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: "totalScore", Value: -1}}}},
			bson.D{{Key: "$skip", Value: startRank}},
			bson.D{{Key: "$limit", Value: numRows}},
		)

		cursor, err := scoresCollection.Aggregate(context.TODO(), pipeline)
		if err != nil {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
		defer cursor.Close(context.TODO())

		if err = cursor.All(context.Background(), &aggregatedScores); err != nil {
			log.Println("Failed to decode aggregated scores:", err)
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
	} else {
		filter := bson.M{"song_id": req.SongID, "role_id": req.RoleID}

		cursor, err := scoresCollection.Find(context.TODO(), filter, &options.FindOptions{
			Skip:  &startRank,
			Limit: &numRows,
			Sort:  bson.M{"score": -1},
		})

		if err != nil {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}

		if err = cursor.All(context.Background(), &scores); err != nil {
			log.Println("Failed to decode all scores:", err)
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
	}

	// collect all the player and band PIDs we need to fetch
	playerPIDs := make([]int, 0)
	if isAggregated {
		for _, score := range aggregatedScores {
			playerPIDs = append(playerPIDs, score.PID)
		}
	} else {
		for _, score := range scores {
			playerPIDs = append(playerPIDs, score.OwnerPID)
		}
	}

	// grab console-prefixed usernames for players and band names for the bands
	playerNames, _ := db.GetConsolePrefixedUsernamesByPIDs(context.Background(), database, playerPIDs)
	nonPrefixedPlayerNames, _ := db.GetUsernamesByPIDs(context.Background(), database, playerPIDs)
	bandNames, _ := db.GetBandNamesByOwnerPIDs(context.Background(), database, playerPIDs)

	var res []RankRangeGetResponse
	var startIdx int = req.StartRank

	if isAggregated {
		// Build response for aggregated leaderboards
		isBandScore := req.RoleID == 10

		for _, score := range aggregatedScores {
			var name string

			if isBandScore {
				name = bandNames[score.PID]
			} else {
				name = playerNames[score.PID]
			}

			if name == "" {
				if isBandScore {
					playerName := nonPrefixedPlayerNames[score.PID]
					if playerName != "" {
						name = playerName + "'s Band"
					} else {
						name = "Unnamed Band"
					}
				} else {
					name = "Unnamed Player"
				}
			}

			isFriend := 0
			if friendsMap[score.PID] {
				isFriend = 1
			}

			res = append(res, RankRangeGetResponse{
				PID:          score.PID,
				Name:         name,
				DiffID:       0, // Aggregated - no specific difficulty
				Rank:         startIdx,
				Score:        score.TotalScore,
				IsPercentile: 0,
				InstMask:     0, // Aggregated - no specific instrument
				NotesPct:     0, // Aggregated - no specific notes percent
				IsFriend:     isFriend,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        startIdx,
			})

			startIdx++
		}
	} else {
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
						name = playerName + "'s Band" // "Player's Band" if the band name is not set but the player is known
					} else {
						name = "Unnamed Band"
					}
				} else {
					name = "Unnamed Player"
				}
			}

			isFriend := 0
			if friendsMap[score.OwnerPID] {
				isFriend = 1
			}

			res = append(res, RankRangeGetResponse{
				PID:          score.OwnerPID,
				Name:         name,
				DiffID:       score.DiffID,
				Rank:         startIdx,
				Score:        score.Score,
				IsPercentile: 0,
				InstMask:     score.InstrumentMask,
				NotesPct:     score.NotesPercent,
				IsFriend:     isFriend,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        startIdx,
			})

			startIdx++
		}
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
