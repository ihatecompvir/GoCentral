package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type FriendsUpdateRequest struct {
	Region      string   `json:"region"`
	SystemMS    int      `json:"system_ms"`
	MachineID   string   `json:"machine_id"`
	SessionGUID string   `json:"session_guid"`
	PID         int      `json:"pid"`
	Names       []string `json:"nameXXX"`
	GUIDs       []string `json:"guidXXX"`
}

type FriendsUpdateResponse struct {
	Success int `json:"success"`
}

type FriendsUpdateService struct {
}

func (service FriendsUpdateService) Path() string {
	return "leaderboards/friends/update"
}

func (service FriendsUpdateService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req FriendsUpdateRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to update friends without a valid server-assigned PID, rejecting call")
		return "", err
	}

	log.Println("Updating friends list for player ", req.PID)

	// Lookup the user by their PID
	var user models.User
	err = database.Collection("users").FindOne(context.Background(), bson.M{"pid": req.PID}).Decode(&user)
	if err != nil {
		log.Println("Failed to find user with PID ", req.PID, ": ", err)
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	// lookup all usernames in a single shot
	usersCollection := database.Collection("users")

	cursor, err := usersCollection.Find(context.Background(), bson.M{"username": bson.M{"$in": req.Names}})
	if err != nil {
		log.Println("Failed to lookup friend PIDs:", err)
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}
	defer cursor.Close(context.Background())

	// collate valid pids
	var friendPIDs []int
	for cursor.Next(context.Background()) {
		var friendUser models.User
		if err := cursor.Decode(&friendUser); err != nil {
			log.Println("Failed to decode friend user:", err)
			continue
		}
		friendPIDs = append(friendPIDs, int(friendUser.PID))
	}

	// update all PIDs in a single query
	if len(friendPIDs) > 0 {
		filter := bson.M{"pid": req.PID}
		update := bson.M{
			"$addToSet": bson.M{"friends": bson.M{"$each": friendPIDs}},
		}

		_, err := usersCollection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Println("Failed to update friends list for player ", req.PID, ": ", err)
		}
	}

	res := []FriendsUpdateResponse{{0}}

	return marshaler.MarshalResponse(service.Path(), res)
}
