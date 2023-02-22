package accomplishment

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccomplishmentRecordRequest struct {
	Name        string `json:"name"`
	Region      string `json:"region"`
	Flags       int    `json:"flags"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	GUID        string `json:"guid"`
	CharData    string `json:"char_data"`
}

type AccomplishmentRecordResponse struct {
	Test int `json:"test"`
}

type AccomplishmentRecordService struct {
}

func (service AccomplishmentRecordService) Path() string {
	return "accomplishment/record"
}

func (service AccomplishmentRecordService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccomplishmentRecordRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for recording accomplishment")
		return "", err
	}

	// Spoof account linking status, 12345 pid
	res := []AccomplishmentRecordResponse{{
		1,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
