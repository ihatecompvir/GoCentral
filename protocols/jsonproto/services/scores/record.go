package scores

import (
	"rb3server/protocols/jsonproto/marshaler"
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
	Test int `json:"test"`
}

type ScoreRecordService struct {
}

func (service ScoreRecordService) Path() string {
	return "scores/record"
}

func (service ScoreRecordService) Handle(data string) (string, error) {
	var req ScoreRecordRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	// send back something fake - the game is just looking for an ack here
	// this is presumably where we would take this data and enter it into a DB for leaderboards
	res := []ScoreRecordResponse{{
		1,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
