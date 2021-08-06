package config

import (
	"fmt"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
}

type ConfigResponse struct {
	OutDta  string `json:"out_dta"`
	Version string `json:"version"`
}

type ConfigService struct {
}

func (service ConfigService) Path() string {
	return "config/get"
}

func (service ConfigService) Handle(data string, database *mongo.Database) (string, error) {
	var req ConfigRequest

	var motdInfo models.MOTDInfo

	motdCollection := database.Collection("motd")

	result := motdCollection.FindOne(nil, bson.D{}).Decode(&motdInfo)

	fmt.Printf("Found %v documents\n", result)

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	res := []ConfigResponse{{
		motdInfo.DTA,
		"3",
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
