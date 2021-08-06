package battles

import (
	"rb3server/protocols/jsonproto/marshaler"

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

func (service BattleCreateService) Handle(data string, database *mongo.Database) (string, error) {
	var req BattleCreateRequest

	res := []BattleCreateResponse{{0, 12345}}

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	return marshaler.MarshalResponse(service.Path(), res)
}
