package ticker

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	db "rb3server/database"
)

type TickerInfoRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	RoleID      int    `json:"role_id"` // current instrument?
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
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for getting ticker info")
		return "", err
	}

	bandsCollection := database.Collection("bands")
	var band models.Band
	err = bandsCollection.FindOne(nil, bson.M{"pid": req.PID}).Decode(&band)

	setlistsCollection := database.Collection("setlists")

	// count the number of setlists with a type of 1000, 1001, or 1002
	battleCount, err := setlistsCollection.CountDocuments(nil, bson.M{"type": bson.M{"$in": []int{1000, 1001, 1002}}})
	if err != nil {
		return "", err
	}

	scoresCollection := database.Collection("scores")

	// Aggregation to get total scores for each player across all instruments
	// mongo actually the GOAT for this
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{{"_id", "$pid"}, {"totalScore", bson.D{{"$sum", "$score"}}}}}},
		{{"$sort", bson.D{{"totalScore", -1}}}},
	}

	cursor, err := scoresCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return "", err
	}
	defer cursor.Close(context.TODO())

	var results []struct {
		ID         int `bson:"_id"`
		TotalScore int `bson:"totalScore"`
	}
	if err := cursor.All(context.TODO(), &results); err != nil {
		return "", err
	}

	// Calculate overall rank
	totalScoreRank := 0
	for i, result := range results {
		if result.ID == req.PID {
			totalScoreRank = i + 1
			break
		}
	}
	if totalScoreRank == 0 {
		totalScoreRank = len(results) + 1
	}

	// Aggregation to get total scores for the specified instrument
	rolePipeline := mongo.Pipeline{
		{{"$match", bson.D{{"role_id", req.RoleID}}}},
		{{"$group", bson.D{{"_id", "$pid"}, {"totalScore", bson.D{{"$sum", "$score"}}}}}},
		{{"$sort", bson.D{{"totalScore", -1}}}},
	}

	roleCursor, err := scoresCollection.Aggregate(context.TODO(), rolePipeline)
	if err != nil {
		return "", err
	}
	defer roleCursor.Close(context.TODO())

	var roleResults []struct {
		ID         int `bson:"_id"`
		TotalScore int `bson:"totalScore"`
	}
	if err := roleCursor.All(context.TODO(), &roleResults); err != nil {
		return "", err
	}

	roleRank := 0
	for i, result := range roleResults {
		if result.ID == req.PID {
			roleRank = i + 1
			break
		}
	}
	if roleRank == 0 {
		roleRank = len(roleResults) + 1
	}

	// Spoof account linking status, 12345 pid
	res := []TickerInfoResponse{{
		req.PID,
		db.GetCoolFact(),
		int(battleCount),
		req.RoleID,
		roleRank,
		1,
		0,
		band.BandID,
		totalScoreRank,
		1,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
