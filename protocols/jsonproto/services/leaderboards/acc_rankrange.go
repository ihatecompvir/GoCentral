package leaderboard

import (
	"context"
	"log"
	db "rb3server/database"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
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

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for acc leaderboards")
		return "", err
	}

	accomplishmentsCollection := database.Collection("accomplishments")

	// FindOne the accomplishment scores
	var accomplishments models.Accomplishments
	err = accomplishmentsCollection.FindOne(context.TODO(), bson.M{}).Decode(&accomplishments)

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	res := []AccRankRangeGetResponse{}

	accSlice := getAccomplishmentField(req.AccID, accomplishments)

	// sort acc scores by score
	sort.Slice(accSlice, func(i, j int) bool {
		return accSlice[i].Score > accSlice[j].Score
	})

	// get the scores in the range, and append them to the response
	for i := req.StartRank - 1; i < req.EndRank-1; i++ {
		if i >= len(accSlice) {
			break
		}

		score := accSlice[i]
		res = append(res, AccRankRangeGetResponse{
			PID:          score.PID,
			Name:         db.GetConsolePrefixedUsernameForPID(score.PID),
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
