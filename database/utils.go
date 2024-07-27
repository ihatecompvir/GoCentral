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

func GetConsolePrefixedUsernameForPID(pid int) string {
	var user models.User

	usersCollection := GocentralDatabase.Collection("users")

	_ = usersCollection.FindOne(nil, bson.M{"pid": pid}).Decode(&user)

	if user.Username != "" {
		switch user.ConsoleType {
		case 0:
			return user.Username + " [360]"
		case 1:
			return user.Username + " [PS3]"
		case 2:
			return user.Username + " [Wii]"
		case 3:
			return user.Username + " [RPCS3]"
		default:
			return user.Username
		}
	} else {
		return "Unnamed Player"
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
		username := GetUsernameForPID(pid)
		if username != "" {
			return username + "'s Band"
		} else {
			return "Unnamed Band"
		}
	}
}
