package band

import (
	"encoding/hex"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

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
	Test int `json:"test"`
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

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting band update")
		return "", err
	}

	artBytes, err := hex.DecodeString(req.Art)
	if err != nil {
		log.Printf("Could not update band %s for PID %v: %s\n", req.Name, req.PID, err)
		return marshaler.MarshalResponse(service.Path(), []BandUpdateResponse{{0}})
	}

	var config models.Config
	configCollection := database.Collection("config")
	err = configCollection.FindOne(nil, bson.M{}).Decode(&config)
	if err != nil {
		log.Printf("Could not get config %v\n", err)
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
