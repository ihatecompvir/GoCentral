package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

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

func (service PlayerGetService) Handle(data string, database *mongo.Database) (string, error) {
	var req PlayerGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	scores := database.Collection("scores")

	filter := bson.M{"song_id": req.SongID, "role_id": req.RoleID}
	cur, err := scores.Find(context.TODO(), filter, options.Find().SetSort(bson.D{{"score", -1}}))
	if err != nil {
		log.Fatal(err)
	}

	res := []PlayerGetResponse{}

	// for rank
	curIndex := 1

	for cur.Next(nil) && curIndex != 16 {
		username := "Player"

		// create a value into which the single document can be decoded
		var score models.Score
		err := cur.Decode(&score)
		if err != nil {
			log.Fatal(err)
		}

		if score.BOI == 1 && req.RoleID != 10 {

			users := database.Collection("users")
			var user models.User
			err = users.FindOne(nil, bson.M{"pid": score.OwnerPID}).Decode(&user)

			if err == nil {
				username = user.Username
			}

			res = append(res, PlayerGetResponse{
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
				"",
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

			res = append(res, PlayerGetResponse{
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
				"",
				curIndex,
			})
		}
		curIndex += 1
	}

	if len(res) == 0 {
		return marshaler.MarshalResponse(service.Path(), []PlayerGetResponse{{}})
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
