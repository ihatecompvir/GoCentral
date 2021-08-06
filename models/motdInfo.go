package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type MOTDInfo struct {
	ID  primitive.ObjectID `json:"_id" bson:"_id"`
	DTA string             `json:"dta" bson:"dta"`
}
