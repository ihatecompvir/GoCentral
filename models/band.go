package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Band struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id"`
	Art      []byte             `json:"art" bson:"art"`
	Name     string             `json:"name" bson:"name"`
	OwnerPID int                `json:"owner_pid" bson:"owner_pid"`
	BandID   int                `json:"band_id" bson:"band_id"`
}
