package character

import (
	"encoding/hex"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

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
	Test int `json:"test"`
}

type CharacterUpdateService struct {
}

func (service CharacterUpdateService) Path() string {
	return "entities/character/update"
}

func (service CharacterUpdateService) Handle(data string, database *mongo.Database) (string, error) {
	var req CharacterUpdateRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
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
		_, err = characters.InsertOne(nil, bson.D{
			{Key: "guid", Value: req.GUID},
			{Key: "char_data", Value: characterBytes},
			{Key: "name", Value: req.Name},
			{Key: "owner_pid", Value: req.PID},
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
