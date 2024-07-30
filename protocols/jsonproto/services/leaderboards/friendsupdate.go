package leaderboard

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

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for player rank range leaderboards")
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	log.Println("Updating friends list for player ", req.PID)

	// Lookup the user by their PID
	var user models.User
	err = database.Collection("users").FindOne(context.Background(), bson.M{"pid": req.PID}).Decode(&user)
	if err != nil {
		log.Println("Failed to find user with PID ", req.PID, ": ", err)
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	// loop through every name and guid and update the friends list
	for _, name := range req.Names {
		var pid int = 0
		pid = db.GetPIDForUsername(name)

		if pid == 0 {
			continue
		}

		// add the pid to the "friends" int array in the user's document if it is not already there
		filter := bson.M{"pid": req.PID}
		update := bson.M{
			"$addToSet": bson.M{"friends": pid},
		}

		_, err := database.Collection("users").UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Println("Failed to update friends list for player ", req.PID, " with friend PID ", pid, ": ", err)
			continue
		}
	}

	res := []FriendsUpdateResponse{{0}}

	return marshaler.MarshalResponse(service.Path(), res)
}
