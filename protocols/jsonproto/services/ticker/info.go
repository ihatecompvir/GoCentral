package ticker

import (
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TickerInfoRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	RoleID      int    `json:"role_id"` // TODO (I was doing something but I forgot what it was.)
}

type TickerInfoResponse struct {
	PID              int    `json:"pid"`
	MOTD             string `json:"motd"`
	BattleCount      int    `json:"battle_count"`
	RoleID           int    `json:"role_id"`
	RoleRank         int    `json:"role_rank"`
	RoleIsGlobal     int    `json:"role_is_global"`
	RoleIsPercentile int    `json:"role_is_percentile"`
	BandID           int    `json:"band_id"`
	BandRank         int    `json:"band_rank"`
	BankIsGlobal     int    `json:"band_is_global"`
	BandIsPercentile int    `json:"band_is_percentile"`
}

type TickerInfoService struct {
}

func (service TickerInfoService) Path() string {
	return "ticker/info/get"
}

func (service TickerInfoService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req TickerInfoRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		users := database.Collection("users")
		var user models.User
		err = users.FindOne(nil, bson.M{"pid": req.PID}).Decode(&user)
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for getting ticker info")
		log.Println("Database PID : ", user.PID)
		client.SetPlayerID(user.PID)
		log.Println("Client PID : ", client.PlayerID())
	}

	bandsCollection := database.Collection("bands")
	var band models.Band
	err = bandsCollection.FindOne(nil, bson.M{"pid": req.PID}).Decode(&band)

	// Spoof account linking status, 12345 pid
	res := []TickerInfoResponse{{
		req.PID,
		"",
		0,
		3,
		0,
		0,
		0,
		band.BandID,
		0,
		0,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
