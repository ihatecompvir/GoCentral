package database

import (
	"rb3server/models"

	"go.mongodb.org/mongo-driver/bson"
)

// Convenience functions involving the DB
// This is to reduce the boilerplate code everywhere for common functions like PID-->Username resolution

// returns the username for a given PID
func GetUsernameForPID(pid int) string {
	var user models.User

	usersCollection := GocentralDatabase.Collection("users")

	_ = usersCollection.FindOne(nil, bson.M{"pid": pid}).Decode(&user)

	if user.Username != "" {
		return user.Username
	} else {
		return "Player"
	}
}

// returns the name of the band for a given band_id
func GetBandNameForBandID(pid int) string {
	var band models.Band

	bandsCollection := GocentralDatabase.Collection("bands")

	_ = bandsCollection.FindOne(nil, bson.M{"owner_pid": pid}).Decode(&band)

	if band.Name != "" {
		return band.Name
	} else {
		return "Band"
	}
}
