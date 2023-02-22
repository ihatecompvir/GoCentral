package leaderboard

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type MaxrankGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	SongID      int    `json:"song_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	RoleID      int    `json:"role_id"`
	PID000      int    `json:"pid000"`
	LBType      int    `json:"lb_type"`
}

type MaxrankGetResponse struct {
	MaxRank int `json:"max_rank"`
}

type MaxrankGetService struct {
}

func (service MaxrankGetService) Path() string {
	return "leaderboards/maxrank/get"
}

func (service MaxrankGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req MaxrankGetRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for maxrank")
		return "", err
	}

	// Spoof account linking status, 12345 pid
	res := []MaxrankGetResponse{{
		1,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
