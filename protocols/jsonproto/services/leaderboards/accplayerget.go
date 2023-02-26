package leaderboard

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccPlayerGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	AccID       string `json:"acc_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
	LBMode      int    `json:"lb_mode"`
	NumRows     int    `json:"num_rows"`
}

type AccPlayerGetResponse struct {
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

type AccPlayerGetService struct {
}

func (service AccPlayerGetService) Path() string {
	return "leaderboards/acc_player/get"
}

func (service AccPlayerGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccPlayerGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for accomplishment leaderboards")
		return "", err
	}
	return marshaler.MarshalResponse(service.Path(), []AccPlayerGetResponse{{
		69,
		"Test",
		4,
		1,
		69,
		0,
		2,
		100,
		0,
		0,
		"",
		1,
	}})
}
