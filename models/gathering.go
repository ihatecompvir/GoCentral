package models

import "go.mongodb.org/mongo-driver/bson/primitive"

/// This is different from the model inside of the serialization folder and refers to the MongoDB representation of a gathering
type Gathering struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	GatheringID int                `json:"gathering_id" bson:"gathering_id"`
	Creator     string             `json:"creator" bson:"creator"`
	Contents    []byte             `json:"contents" bson:"contents"`
	State       uint32             `json:"state" bson:"state"`
	LastUpdated int64              `json:"last_updated" bson:"last_updated"`
}
