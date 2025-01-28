package entities

import (
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type GetLinkcodeRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type GetLinkcodeResponse struct {
	Code string `json:"code"`
}

type GetLinkcodeService struct {
}

func (service GetLinkcodeService) Path() string {
	return "entities/linkcode/get"
}

func (service GetLinkcodeService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req GetLinkcodeRequest
	err := marshaler.UnmarshalRequest(data, &req)

	res := []GetLinkcodeResponse{{}}

	if err != nil {
		log.Println("Failed to unmarshal GetLinkcodeRequest:", err)
		res = []GetLinkcodeResponse{{
			"Could not get link code, please try again later",
		}}
	}

	// make sure the client is asking for their own link code
	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to get a link code without a valid server-assigned PID, rejecting call")
		return "", err
	}

	usersCollection := database.Collection("users")
	var user models.User

	result := usersCollection.FindOne(nil, bson.M{"pid": req.PID})

	if result.Err() != nil {
		log.Println("Could not find user with PID", req.PID)
		res = []GetLinkcodeResponse{{
			"Could not get link code, please try again later",
		}}
	}

	err = result.Decode(&user)

	if err != nil {
		log.Println("Could not decode user with PID", req.PID)
		res = []GetLinkcodeResponse{{
			"Could not get link code, please try again later",
		}}
	}

	// Spoof account linking status, 12345 pid
	res = []GetLinkcodeResponse{{
		user.LinkCode,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
