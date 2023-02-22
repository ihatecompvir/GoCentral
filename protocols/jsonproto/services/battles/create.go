package battles

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type BattleCreateRequest struct {
	Type         int    `json:"type"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	Description  string `json:"description"`
	Flags        int    `json:"flags"`
	Instrument   int    `json:"instrument"`
	SystemMS     int    `json:"system_ms"`
	MachineID    string `json:"machine_id"`
	SessionGUID  string `json:"session_guid"`
	PID          int    `json:"pid"`
	TimeEndVal   int    `json:"time_end_val"`
	TimeEndUnits string `json:"time_end_units"`
	SongID       int    `json:"song_id000"`
}

type BattleCreateResponse struct {
	Success  int `json:"success"`
	BattleID int `json:"battle_id"`
}

type BattleCreateService struct {
}

func (service BattleCreateService) Path() string {
	return "battles/limit/check"
}

func (service BattleCreateService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req BattleCreateRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting battle creation")
		return "", err
	}

	res := []BattleCreateResponse{{0, 12345}}

	return marshaler.MarshalResponse(service.Path(), res)
}
