package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID               primitive.ObjectID `json:"_id" bson:"_id"`
	Username         string             `json:"username" bson:"username"`
	PID              uint32             `json:"pid" bson:"pid"`
	StationURL       string             `json:"station_url" bson:"station_url"`
	IntStationURL    string             `json:"int_station_url" bson:"int_station_url"`
	ConsoleType      int                `json:"console_type" bson:"console_type"`
	GUID             string             `json:"guid" bson:"guid"`
	LinkCode         string             `json:"link_code" bson:"link_code"`
	Friends          []int              `json:"friends" bson:"friends"`
	Groups           []string           `json:"groups" bson:"groups"`
	CrossplayEnabled bool               `json:"crossplay_enabled" bson:"crossplay_enabled"`

	// machine stuff
	CreatedByMachineID int `json:"created_by_machine_id" bson:"created_by_machine_id"`
}
