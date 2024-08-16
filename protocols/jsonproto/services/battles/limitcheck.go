package battles

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	db "rb3server/database"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type LimitCheckRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type LimitCheckResponse struct {
	Success int `json:"success"`
}

type LimitCheckService struct {
}

func (service LimitCheckService) Path() string {
	return "battles/limit/check"
}

func (service LimitCheckService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req LimitCheckRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting battle limit check")
		return "", err
	}

	var config models.Config
	configCollection := database.Collection("config")
	err = configCollection.FindOne(context.TODO(), bson.M{}).Decode(&config)
	if err != nil {
		log.Printf("Could not get config %v\n", err)
	}

	users := database.Collection("users")
	var user models.User
	err = users.FindOne(context.TODO(), bson.M{"pid": req.PID}).Decode(&user)

	if err != nil {
		log.Printf("Could not find user with PID %d, could not check limit", req.PID)
		return marshaler.MarshalResponse(service.Path(), []LimitCheckResponse{{0x16}})
	}

	if db.IsPIDInGroup(req.PID, "battle_admin") {
		// if the user is a battle administrator, they can create as many battles as they want
		// so do not check the limit
		return marshaler.MarshalResponse(service.Path(), []LimitCheckResponse{{0}})
	}

	// find how many battles this user has created
	// type must be either 1000, 1001, or 1002 so we don't catch normal setlists
	setlistsCollection := database.Collection("setlists")
	count, err := setlistsCollection.CountDocuments(context.TODO(), bson.M{"pid": req.PID, "type": bson.M{"$in": []int{1000, 1001, 1002}}})
	if err != nil {
		log.Printf("Could not count setlists for user %d, could not check battle limit", req.PID)
		return marshaler.MarshalResponse(service.Path(), []LimitCheckResponse{{0x16}})
	}

	if int(count) >= config.BattleLimit {
		return marshaler.MarshalResponse(service.Path(), []LimitCheckResponse{{0x16}})
	}

	res := []LimitCheckResponse{{0}}

	return marshaler.MarshalResponse(service.Path(), res)
}
