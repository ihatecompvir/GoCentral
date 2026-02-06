package leaderboard

import (
	"context"
	"log"
	db "rb3server/database"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"
	"sort"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccRankRangeGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	AccID       string `json:"acc_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
	StartRank   int    `json:"start_rank"`
	EndRank     int    `json:"end_rank"`
	LBType      int    `json:"lb_type"`
}

type AccRankRangeGetResponse struct {
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

type AccRankRangeGetService struct {
}

func (service AccRankRangeGetService) Path() string {
	return "leaderboards/acc_rankrange/get"
}

func (service AccRankRangeGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccRankRangeGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID000))

	if !validPIDres {
		log.Println("Client is attempting to get leaderboards without a valid server-assigned PID, rejecting call")
		return "", err
	}

	// fetch friends list for IsFriend marking
	friendsMap, _ := db.GetFriendsForPID(context.Background(), database, req.PID000)

	accomplishmentsCollection := database.Collection("accomplishments")

	var accomplishments models.Accomplishments
	err = accomplishmentsCollection.FindOne(context.TODO(), bson.M{}).Decode(&accomplishments)

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	accSlice := getAccomplishmentField(req.AccID, accomplishments)

	// sort acc scores by score
	sort.Slice(accSlice, func(i, j int) bool {
		return accSlice[i].Score > accSlice[j].Score
	})

	// calculate what the actual range will be
	start := req.StartRank - 1
	if start < 0 {
		start = 0
	}
	end := req.EndRank
	if end > len(accSlice) {
		end = len(accSlice)
	}
	if start > end {
		start = end
	}
	visibleScores := accSlice[start:end]

	// collect all the player PIDs we need to fetch
	playerPIDs := make([]int, 0, len(visibleScores))
	for _, score := range visibleScores {
		playerPIDs = append(playerPIDs, score.PID)
	}

	// grab console-prefixed usernames for all players at once
	playerNames, _ := db.GetConsolePrefixedUsernamesByPIDs(context.Background(), database, playerPIDs)

	res := []AccRankRangeGetResponse{}

	for i, score := range visibleScores {
		// get the player name from the map
		// since we prefetched the names this is a quick map lookup
		name := playerNames[score.PID]

		// use fallback name if something could not be fetched or wasn't in the db
		if name == "" {
			name = "Unnamed Player"
		}

		rank := start + i + 1

		isFriend := 0
		if friendsMap[score.PID] {
			isFriend = 1
		}

		res = append(res, AccRankRangeGetResponse{
			PID:          score.PID,
			Name:         name,
			DiffID:       0,
			Rank:         rank,
			Score:        score.Score,
			IsPercentile: 0,
			InstMask:     0,
			NotesPct:     0,
			IsFriend:     isFriend,
			UnnamedBand:  0,
			PGUID:        "",
			ORank:        rank,
		})
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
