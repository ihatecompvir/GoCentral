package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Config struct {
	ID              primitive.ObjectID `json:"_id" bson:"_id"`
	LastPID         int                `json:"last_pid" bson:"last_pid"`
	LastBandID      int                `json:"last_band_id" bson:"last_band_id"`
	LastCharacterID int                `json:"last_character_id" bson:"last_character_id"`
	LastSetlistID   int                `json:"last_setlist_id" bson:"last_setlist_id"`
	ProfanityList   []string           `json:"profanity_list" bson:"profanity_list"`
}
