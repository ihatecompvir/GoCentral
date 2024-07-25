package scores

import (
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var instrumentMap = map[int]int{
	0: 1,
	1: 2,
	2: 4,
	3: 8,
	4: 16,
	5: 32,
	6: 64,
	7: 128,
	8: 256,
	9: 512,
}

type ScoreRecordRequest struct {
	Region           string `json:"region"`
	Locale           string `json:"locale"`
	SystemMS         int    `json:"system_ms"`
	SongID           int    `json:"song_id"`
	MachineID        string `json:"machine_id"`
	SessionGUID      string `json:"session_guid"`
	PIDs             []int  `json:"pidXXX"`
	BoiID            int    `json:"boi_id"`
	BandMask         int    `json:"band_mask"`
	ProvideInstaRank int    `json:"provide_insta_rank"`
	RoleIDs          []int  `json:"role_idXXX"`
	Scores           []int  `json:"scoreXXX"`
	Stars            []int  `json:"starsXXX"`
	Slots            []int  `json:"slotXXX"`
	DiffIDs          []int  `json:"diff_idXXX"`
	CScores          []int  `json:"c_scoreXXX"`
	CCScores         []int  `json:"cc_scoreXXX"`
	Percents         []int  `json:"percentXXX"`
}

type ScoreRecordResponse struct {
	ID           int    `json:"id"`
	IsBOI        int    `json:"is_boi"`
	InstaRank    int    `json:"insta_rank"`
	IsPercentile int    `json:"is_percentile"`
	Part1        string `json:"part_1"`
	Part2        string `json:"part_2"`
}

type ScoreRecordService struct {
}

func (service ScoreRecordService) Path() string {
	return "scores/record"
}

func (service ScoreRecordService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req ScoreRecordRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PIDs[0] != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting setlist update")
		return "", err
	}

	scoresCollection := database.Collection("scores")

	// every PID in the request has a score and etc. so make sure all those get added to the DB
	for idx, pid := range req.PIDs {
		var Score models.Score
		Score.OwnerPID = pid
		Score.SongID = req.SongID
		Score.Stars = req.Stars[idx]
		Score.DiffID = req.DiffIDs[idx]
		Score.Score = req.Scores[idx]
		Score.InstrumentMask = req.BandMask
		Score.NotesPercent = req.Percents[idx]
		Score.RoleID = req.RoleIDs[idx]

		if req.RoleIDs[idx] == 10 {
			Score.BOI = 0
		} else {
			Score.BOI = 1
			Score.InstrumentMask = instrumentMap[req.RoleIDs[idx]] // make sure the instrument mask for individual instruments is right
		}

		// upsert the new score
		_, err = scoresCollection.UpdateOne(
			nil,
			bson.M{"song_id": req.SongID, "pid": Score.OwnerPID, "role_id": Score.RoleID},
			bson.D{
				{"$set", bson.D{
					{"song_id", Score.SongID},
					{"pid", Score.OwnerPID},
					{"role_id", Score.RoleID},
					{"score", Score.Score},
					{"notespct", Score.NotesPercent},
					{"stars", Score.Stars},
					{"diff_id", Score.DiffID},
					{"boi", Score.BOI},
					{"instrument_mask", Score.InstrumentMask},
				}},
			},
			options.Update().SetUpsert(true),
		)
	}

	res := []ScoreRecordResponse{{
		req.SongID,
		0,
		1,
		0,
		"",
		"",
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
