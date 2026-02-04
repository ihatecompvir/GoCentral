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

	// fetch friends list for IsFriend marking and friends leaderboard filtering
	friendsMap, _ := db.GetFriendsForPID(context.Background(), database, req.PID000)

	scoresCollection := database.Collection("scores")

	isAggregated := req.LBType == LBTypeTotalScore || req.LBType == LBTypeRB3Only

	if isAggregated {

		var pidFilter []int

		if req.LBMode == 1 {
			for pid := range friendsMap {
				pidFilter = append(pidFilter, pid)
			}
		} else if req.LBMode >= 2 && req.LBMode <= 5 {
			consoleTypeMap := map[int]int{2: 1, 3: 3, 4: 2, 5: 0}
			consolePIDs, _ := db.GetPIDsByConsoleType(context.Background(), database, consoleTypeMap[req.LBMode])
			for pid := range consolePIDs {
				pidFilter = append(pidFilter, pid)
			}
		}

		matchStage := bson.D{}

		// Exclude battle and setlist scores from total score calculations
		matchStage = append(matchStage, bson.E{Key: "battle_id", Value: bson.D{{Key: "$not", Value: bson.D{{Key: "$gt", Value: 0}}}}})
		matchStage = append(matchStage, bson.E{Key: "setlist_id", Value: bson.D{{Key: "$not", Value: bson.D{{Key: "$gt", Value: 0}}}}})

		// For RB3 Only, filter to song_id 1001-1106 (I think this is the full range)
		if req.LBType == LBTypeRB3Only {
			matchStage = append(matchStage, bson.E{Key: "song_id", Value: bson.D{{Key: "$gte", Value: 1001}, {Key: "$lte", Value: 1106}}})
		}

		if req.RoleID > 0 {
			matchStage = append(matchStage, bson.E{Key: "role_id", Value: req.RoleID})
		}

		if len(pidFilter) > 0 {
			matchStage = append(matchStage, bson.E{Key: "pid", Value: bson.D{{Key: "$in", Value: pidFilter}}})
		}

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
		)

		cursor, err := scoresCollection.Aggregate(context.TODO(), pipeline)
		if err != nil {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}

		var allAggregatedScores []struct {
			PID        int `bson:"_id"`
			TotalScore int `bson:"totalScore"`
		}
		if err = cursor.All(context.TODO(), &allAggregatedScores); err != nil {
			cursor.Close(context.TODO())
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
		cursor.Close(context.TODO())

		playerRank := -1
		for i, score := range allAggregatedScores {
			if score.PID == req.PID000 {
				playerRank = i
				break
			}
		}

		if playerRank == -1 {
			playerRank = 0
		}

		// calc page start
		startRank := int64(playerRank - (playerRank % 19))
		limit := int64(19)

		// Extract the page of results
		endIdx := int(startRank + limit)
		if endIdx > len(allAggregatedScores) {
			endIdx = len(allAggregatedScores)
		}
		startIdx := int(startRank)
		if startIdx > len(allAggregatedScores) {
			startIdx = len(allAggregatedScores)
		}
		pageScores := allAggregatedScores[startIdx:endIdx]

		// Collect PIDs for name lookup
		playerPIDs := make([]int, 0)
		for _, score := range pageScores {
			playerPIDs = append(playerPIDs, score.PID)
		}

		// Fetch names
		playerNames, _ := db.GetConsolePrefixedUsernamesByPIDs(context.Background(), database, playerPIDs)
		nonPrefixedPlayerNames, _ := db.GetUsernamesByPIDs(context.Background(), database, playerPIDs)
		bandNames, _ := db.GetBandNamesByOwnerPIDs(context.Background(), database, playerPIDs)

		// Build response
		var res []PlayerGetResponse
		idx := int(startRank) + 1
		isBandScore := req.RoleID == 10

		for _, score := range pageScores {
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

			res = append(res, PlayerGetResponse{
				PID:          score.PID,
				Name:         name,
				DiffID:       0,
				Rank:         idx,
				Score:        score.TotalScore,
				IsPercentile: 0,
				InstMask:     0,
				NotesPct:     0,
				IsFriend:     isFriend,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        idx,
			})
			idx++
		}

		if len(res) == 0 {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
		return marshaler.MarshalResponse(service.Path(), res)
	}

	// build the base query filter
	baseFilter := bson.M{"song_id": req.SongID, "role_id": req.RoleID}

	// for friends leaderboard, filter to only friends' scores
	if req.LBMode == 1 {
		friendPIDs := make([]int, 0, len(friendsMap))
		for pid := range friendsMap {
			friendPIDs = append(friendPIDs, pid)
		}
		baseFilter["pid"] = bson.M{"$in": friendPIDs}
	}

	// for console-specific leaderboards, filter to only scores from users with that console type
	// mode 2 = PS3 (console_type 1), mode 3 = RPCS3 (console_type 3), mode 4 = Wii (console_type 2), mode 5 = Xbox (console_type 0)
	if req.LBMode >= 2 && req.LBMode <= 5 {
		consoleTypeMap := map[int]int{2: 1, 3: 3, 4: 2, 5: 0} // LBMode -> consoleType
		consolePIDs, _ := db.GetPIDsByConsoleType(context.Background(), database, consoleTypeMap[req.LBMode])
		pidList := make([]int, 0, len(consolePIDs))
		for pid := range consolePIDs {
			pidList = append(pidList, pid)
		}
		baseFilter["pid"] = bson.M{"$in": pidList}
	}

	var playerScore models.Score
	playerFilter := bson.M{"song_id": req.SongID, "role_id": req.RoleID, "pid": req.PID000}
	err = scoresCollection.FindOne(context.TODO(), playerFilter).Decode(&playerScore)
	if err != nil && err != mongo.ErrNoDocuments {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	if err == mongo.ErrNoDocuments {
		err = scoresCollection.FindOne(context.TODO(), baseFilter, &options.FindOneOptions{
			Sort: bson.M{"score": -1},
		}).Decode(&playerScore)
		if err != nil && err != mongo.ErrNoDocuments {
			return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
		}
	}

	// count scores higher than player's score within the filtered set
	countFilter := bson.M{"song_id": req.SongID, "role_id": req.RoleID, "score": bson.M{"$gt": playerScore.Score}}
	if req.LBMode == 1 {
		friendPIDs := make([]int, 0, len(friendsMap))
		for pid := range friendsMap {
			friendPIDs = append(friendPIDs, pid)
		}
		countFilter["pid"] = bson.M{"$in": friendPIDs}
	}
	// apply the same console type filter to count query
	if req.LBMode >= 2 && req.LBMode <= 5 {
		consoleTypeMap := map[int]int{2: 1, 3: 3, 4: 2, 5: 0}
		consolePIDs, _ := db.GetPIDsByConsoleType(context.Background(), database, consoleTypeMap[req.LBMode])
		pidList := make([]int, 0, len(consolePIDs))
		for pid := range consolePIDs {
			pidList = append(pidList, pid)
		}
		countFilter["pid"] = bson.M{"$in": pidList}
	}
	playerScoreIdx, err := scoresCollection.CountDocuments(context.TODO(), countFilter)
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	startRank := playerScoreIdx - (playerScoreIdx % 19)
	limit := int64(19) // The limit is always the page size, not the end rank.

	cursor, err := scoresCollection.Find(context.TODO(), baseFilter, &options.FindOptions{
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
	for _, score := range scores {
		playerPIDs = append(playerPIDs, score.OwnerPID)
	}

	// grab console-prefixed usernames for players and band names for the bands
	playerNames, _ := db.GetConsolePrefixedUsernamesByPIDs(context.Background(), database, playerPIDs)
	nonPrefixedPlayerNames, _ := db.GetUsernamesByPIDs(context.Background(), database, playerPIDs)
	bandNames, _ := db.GetBandNamesByOwnerPIDs(context.Background(), database, playerPIDs)

	var res []PlayerGetResponse

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

		isFriend := 0
		if friendsMap[score.OwnerPID] {
			isFriend = 1
		}

		res = append(res, PlayerGetResponse{
			PID:          score.OwnerPID,
			Name:         name,
			DiffID:       score.DiffID,
			Rank:         int(idx),
			Score:        score.Score,
			IsPercentile: 0,
			InstMask:     score.InstrumentMask,
			NotesPct:     score.NotesPercent,
			IsFriend:     isFriend,
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
