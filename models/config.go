package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Config struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	LastPID int                `json:"last_pid" bson:"last_pid"`
}
