package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Character struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	CharData    []byte             `json:"char_data" bson:"char_data"`
	Name        string             `json:"name" bson:"name"`
	OwnerPID    int                `json:"owner_pid" bson:"owner_pid"`
	GUID        string             `json:"guid" bson:"guid"`
	CharacterID int                `json:"character_id" bson:"character_id"`
}
