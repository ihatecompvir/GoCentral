package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BannedPlayer struct {
	Username  string    `json:"username" bson:"username"`
	Reason    string    `json:"reason" bson:"reason"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type Config struct {
	ID              primitive.ObjectID `json:"_id" bson:"_id"`
	LastPID         int                `json:"last_pid" bson:"last_pid"`
	LastBandID      int                `json:"last_band_id" bson:"last_band_id"`
	LastCharacterID int                `json:"last_character_id" bson:"last_character_id"`
	LastSetlistID   int                `json:"last_setlist_id" bson:"last_setlist_id"`
	ProfanityList   []string           `json:"profanity_list" bson:"profanity_list"`
	BannedPlayers   []BannedPlayer     `json:"banned_players" bson:"banned_players"`
	BattleLimit     int                `json:"battle_limit" bson:"battle_limit"`
	LastMachineID   int                `json:"last_machine_id" bson:"last_machine_id"`
	AdminAPIToken   string             `json:"admin_api_token" bson:"admin_api_token"`
}
