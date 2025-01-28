package character

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"
	"strings"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CharacterNameCheckRequest struct {
	Name        string `json:"name"`
	Region      string `json:"region"`
	Flags       int    `json:"flags"`
	PID         int    `json:"pid"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
}

type CharacterNameCheckResponse struct {
	RetCode int `json:"ret_code"`
}

type CharacterNameCheckService struct {
}

func (service CharacterNameCheckService) Path() string {
	return "entities/character/update"
}

func (service CharacterNameCheckService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req CharacterNameCheckRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to namecheck for a character without a valid server-assigned PID, rejecting call")
		return "", err
	}

	// do a profanity check before updating the band
	var config models.Config
	configCollection := database.Collection("config")
	err = configCollection.FindOne(context.TODO(), bson.M{}).Decode(&config)
	if err != nil {
		log.Printf("Could not get config %v\n", err)
	}

	// check if the band name contains anything in the profanity list
	// NOTE: exercise caution with the profanity list. Putting "ass" on the list would mean that a name like "Band Assistant" is not allowed.
	// use your best judgment, it's up to you to define your own profanity list as a server host, GoCentral does not and will not ship with one
	for _, profanity := range config.ProfanityList {
		if profanity != "" && req.Name != "" && len(req.Name) >= len(profanity) {
			lowerName := strings.ToLower(req.Name)
			lowerProfanity := strings.ToLower(profanity)

			if lowerName == lowerProfanity {
				return marshaler.MarshalResponse(service.Path(), []CharacterNameCheckResponse{{2}})
			}

			if strings.Contains(lowerName, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []CharacterNameCheckResponse{{2}})
			}
		}
	}

	return marshaler.MarshalResponse(service.Path(), []CharacterNameCheckResponse{{1}})
}
