package character

import (
	"context"
	"encoding/hex"
	"log"
	db "rb3server/database"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CharacterUpdateRequest struct {
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

type CharacterUpdateResponse struct {
	RetCode int `json:"ret_code"`
}

type CharacterUpdateService struct {
}

func (service CharacterUpdateService) Path() string {
	return "entities/character/update"
}

func (service CharacterUpdateService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req CharacterUpdateRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to update a character without a valid server-assigned PID, rejecting call")
		return "", err
	}

	characterBytes, err := hex.DecodeString(req.CharData)
	if err != nil {
		log.Printf("Could not update character %s with GUID %s for PID %v: %s\n", req.Name, req.GUID, req.PID, err)
		return marshaler.MarshalResponse(service.Path(), []CharacterUpdateResponse{{0}})
	}

	characters := database.Collection("characters")
	var character models.Character
	err = characters.FindOne(nil, bson.M{"guid": req.GUID}).Decode(&character)

	if err != nil {

		newCharacterID, err := db.GetNextCharacterID(context.Background())
		if err != nil {
			log.Println("Could not get next character ID: ", err)
			return marshaler.MarshalResponse(service.Path(), []CharacterUpdateResponse{{0}})
		}

		_, err = characters.InsertOne(nil, bson.D{
			{Key: "guid", Value: req.GUID},
			{Key: "char_data", Value: characterBytes},
			{Key: "name", Value: req.Name},
			{Key: "owner_pid", Value: req.PID},
			{Key: "character_id", Value: newCharacterID},
		})
		if err != nil {
			log.Printf("Could not update character %s with GUID %s for PID %v: %s\n", req.Name, req.GUID, req.PID, err)
		}
		return marshaler.MarshalResponse(service.Path(), []CharacterUpdateResponse{{1}})
	}

	_, err = characters.UpdateOne(nil, bson.M{"guid": req.GUID}, bson.M{"$set": bson.M{
		"char_data": characterBytes,
		"name":      req.Name,
	}})

	if err != nil {
		log.Printf("Could not update character %s with GUID %s for PID %v: %s\n", req.Name, req.GUID, req.PID, err)
		return marshaler.MarshalResponse(service.Path(), []CharacterUpdateResponse{{0}})
	}

	return marshaler.MarshalResponse(service.Path(), []CharacterUpdateResponse{{1}})
}
