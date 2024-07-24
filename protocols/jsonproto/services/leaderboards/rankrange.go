package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

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
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for leaderboards")
		return "", err
	}

	scores := database.Collection("scores")

	options := options.Find().SetSort(bson.D{{"score", -1}}).SetSkip(int64(req.StartRank)).SetLimit(int64(req.EndRank - req.StartRank))
	filter := bson.M{"song_id": req.SongID, "role_id": req.RoleID}
	cur, err := scores.Find(context.TODO(), filter, options)
	if err != nil {
		return "", err
	}

	res := []RankRangeGetResponse{}

	curIndex := req.StartRank

	for cur.Next(nil) {
		username := "Player"

		// create a value into which the single document can be decoded
		var score models.Score
		err := cur.Decode(&score)
		if err != nil {
			log.Printf("Error decoding score: %v", err)
			return marshaler.MarshalResponse(service.Path(), []RankRangeGetResponse{{}})
		}

		if score.BOI == 1 && req.RoleID != 10 {

			users := database.Collection("users")
			var user models.User
			err = users.FindOne(nil, bson.M{"pid": score.OwnerPID}).Decode(&user)

			if err == nil {
				username = user.Username
			}

			res = append(res, RankRangeGetResponse{
				score.OwnerPID,
				username,
				score.DiffID,
				curIndex,
				score.Score,
				0,
				score.InstrumentMask,
				score.NotesPercent,
				0,
				0,
				"N/A",
				curIndex,
			})

		} else {
			bands := database.Collection("bands")
			var band models.Band
			var bandName = "Band"
			err = bands.FindOne(nil, bson.M{"owner_pid": score.OwnerPID}).Decode(&band)

			if err == nil {
				bandName = band.Name
			}

			res = append(res, RankRangeGetResponse{
				score.OwnerPID,
				bandName,
				score.DiffID,
				curIndex,
				score.Score,
				0,
				score.InstrumentMask,
				score.NotesPercent,
				0,
				0,
				"N/A",
				curIndex,
			})
		}
		curIndex += 1
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
