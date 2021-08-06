package scores

import (
	"rb3server/protocols/jsonproto/marshaler"

	"go.mongodb.org/mongo-driver/mongo"
)

type ScoreRecordRequest struct {
	Region           string `json:"region"`
	Locale           string `json:"locale"`
	SystemMS         int    `json:"system_ms"`
	SongID           int    `json:"song_id"`
	MachineID        string `json:"machine_id"`
	SessionGUID      string `json:"session_guid"`
	PID000           int    `json:"pid000"`
	BoiID            int    `json:"boi_id"`
	BandMask         int    `json:"band_mask"`
	ProvideInstaRank int    `json:"provide_insta_rank"`
	RoleID           int    `json:"role_id000"`
	Score            int    `json:"score000"`
	Stars            int    `json:"stars000"`
	Slot             int    `json:"slot000"`
	DiffID           int    `json:"diff_id000"`
	CScore           int    `json:"c_score000"`
	CCScore          int    `json:"cc_score000"`
	Percent          int    `json:"percent000"`
}

type ScoreRecordResponse struct {
	ID           int    `json:"id"`
	IsBOI        int    `json:"is_boi"`
	InstaRank    int    `json:"insta_rank"`
	IsPercentile int    `json:"is_percentile"`
	Part1        string `json:"part_1"`
	Part2        string `json:"part_2"`
	Slot         int    `json:"slot"`
}

type ScoreRecordService struct {
}

func (service ScoreRecordService) Path() string {
	return "scores/record"
}

func (service ScoreRecordService) Handle(data string, database *mongo.Database) (string, error) {
	var req ScoreRecordRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	// send back something fake - the game is just looking for an ack here
	// this is presumably where we would take this data and enter it into a DB for leaderboards
	res := []ScoreRecordResponse{{
		req.SongID,
		0,
		1,
		0,
		"a",
		"a",
		req.Slot,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
