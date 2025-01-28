package band

import (
	"context"
	"encoding/hex"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"
	"strings"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type BandUpdateRequest struct {
	Name        string `json:"name"`
	Region      string `json:"region"`
	Flags       int    `json:"flags"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	Art         string `json:"art"`
}

type BandUpdateResponse struct {
	RetCode int `json:"ret_code"`
}

type BandUpdateService struct {
}

func (service BandUpdateService) Path() string {
	return "entities/band/update"
}

func (service BandUpdateService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req BandUpdateRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
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
				return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{2}})
			}

			if strings.Contains(lowerName, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{2}})
			}
		}
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to update a band without a valid server-assigned PID, rejecting call")
		return "", err
	}

	artBytes, err := hex.DecodeString(req.Art)
	if err != nil {
		log.Printf("Could not update band %s for PID %v: %s\n", req.Name, req.PID, err)
		return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{1}})
	}

	bands := database.Collection("bands")
	var band models.Band
	err = bands.FindOne(nil, bson.M{"owner_pid": req.PID}).Decode(&band)

	if err != nil {

		_, err = configCollection.UpdateOne(
			nil,
			bson.M{},
			bson.D{
				{"$set", bson.D{{"last_band_id", config.LastBandID + 1}}},
			},
		)

		config.LastBandID += 1

		if err != nil {
			log.Println("Could not update config in database while updating band: ", err)
		}

		_, err = bands.InsertOne(nil, bson.D{
			{Key: "art", Value: artBytes},
			{Key: "name", Value: req.Name},
			{Key: "owner_pid", Value: req.PID},
			{Key: "band_id", Value: config.LastBandID},
		})

		if err != nil {
			log.Printf("Could not update band %s for PID %v: %s\n", req.Name, req.PID, err)
			return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{0}})
		}

		return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{1}})
	}

	_, err = bands.UpdateOne(nil, bson.M{"owner_pid": req.PID}, bson.M{"$set": bson.M{
		"art":  artBytes,
		"name": req.Name,
	}})

	if err != nil {
		log.Printf("Could not update band %s for PID %v: %s\n", req.Name, req.PID, err)
		return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{0}})
	}

	return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{1}})
}
