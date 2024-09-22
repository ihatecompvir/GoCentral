package database

import (
	"context"
	"math/rand"
	"rb3server/models"
	"regexp"
	"strconv"
	"time"

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

// returns the pid for a given username
func GetPIDForUsername(username string) int {
	var user models.User

	usersCollection := GocentralDatabase.Collection("users")

	res := usersCollection.FindOne(nil, bson.M{"username": username})

	if res.Err() != nil {
		return 0
	}

	err := res.Decode(&user)

	if err != nil {
		return 0
	}

	return int(user.PID)
}

func GetCrossplayStatusForPID(pid int) bool {
	var user models.User

	usersCollection := GocentralDatabase.Collection("users")

	_ = usersCollection.FindOne(nil, bson.M{"pid": pid}).Decode(&user)

	return user.CrossplayEnabled
}

func GetCrossplayStatusForUsername(username string) bool {
	var user models.User

	usersCollection := GocentralDatabase.Collection("users")

	_ = usersCollection.FindOne(nil, bson.M{"username": username}).Decode(&user)

	return user.CrossplayEnabled
}

func UpdateCrossplayStatusForPID(pid int, status bool) error {
	usersCollection := GocentralDatabase.Collection("users")

	_, err := usersCollection.UpdateOne(nil, bson.M{"pid": pid}, bson.M{"$set": bson.M{"crossplay_enabled": status}})
	return err
}

func UpdateCrossplayStatusForUsername(username string, status bool) error {
	usersCollection := GocentralDatabase.Collection("users")

	_, err := usersCollection.UpdateOne(nil, bson.M{"username": username}, bson.M{"$set": bson.M{"crossplay_enabled": status}})
	return err
}

// gets the username of the user with a console specific prefix
// e.g. "Player [360]"
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

// gets a random fact about the DB
func GetCoolFact() string {
	// generate a random number between 0-3
	var num int = rand.Intn(4)

	// check if local date is March 31
	if time.Now().Month() == time.March && time.Now().Day() == 31 {
		return "GoCentral's first commit was made on March 31, 2021. Happy birthday GoCentral!"
	}

	pluralize := func(count int64, singular string, plural string) string {
		if count == 1 {
			return singular
		}
		return plural
	}

	switch num {
	case 0:
		scoresCollection := GocentralDatabase.Collection("scores")

		// aggregate all scores and get the cumulative number of stars
		cursor, err := scoresCollection.Aggregate(nil, bson.A{
			bson.M{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$stars"}}},
		})

		if err != nil {
			return "Players on this server have earned an unknown number of stars because something broke trying to calculate it!"
		}

		var result []bson.M
		if err = cursor.All(nil, &result); err != nil {
			return "Players on this server have earned an unknown number of stars because something broke trying to calculate it!"
		}

		stars := result[0]["total"].(int32)
		return "Players on this server have earned a cumulative " + strconv.Itoa(int(stars)) + " " + pluralize(int64(stars), "star", "stars") + "!"
	case 1:
		scoresCollection := GocentralDatabase.Collection("scores")

		// get number of scores
		count, err := scoresCollection.CountDocuments(nil, bson.M{})
		if err != nil {
			return "There are an unknown number of scores on this server because something broke trying to calculate it!"
		}

		return "There " + pluralize(count, "is", "are") + " " + strconv.FormatInt(count, 10) + " " + pluralize(count, "score", "scores") + " on this server!"
	case 2:
		charactersCollection := GocentralDatabase.Collection("characters")

		// get number of characters
		count, err := charactersCollection.CountDocuments(nil, bson.M{})
		if err != nil {
			return "There are an unknown number of characters on this server because something broke trying to calculate it!"
		}

		return "Players on this server have created " + strconv.FormatInt(count, 10) + " " + pluralize(count, "character", "characters") + "!"
	case 3:
		bandsCollection := GocentralDatabase.Collection("bands")

		// get number of bands
		count, err := bandsCollection.CountDocuments(nil, bson.M{})
		if err != nil {
			return "There are an unknown number of bands on this server because something broke trying to calculate it!"
		}

		return "Players on this server have named " + strconv.FormatInt(count, 10) + " " + pluralize(count, "band", "bands") + "!"
	}

	// this should never happen
	return "Rock Band 3 is a game released by Harmonix in 2010. It is the third main game in the Rock Band series."
}

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// generates a 10 digit alphanumeric link code
func GenerateLinkCode(length int) string {
	linkCode := make([]byte, length)
	for i := range linkCode {
		linkCode[i] = charset[rand.Intn(len(charset))]
	}
	return string(linkCode)
}

// checks if a particular PID is a friend of another
func IsPIDAFriendOfPID(pid int, friendPID int) (bool, error) {
	usersCollection := GocentralDatabase.Collection("users")

	// check if the friendPID is in the friend list of the user
	count, err := usersCollection.CountDocuments(context.TODO(), bson.M{"pid": pid, "friends": friendPID})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// gets the expiry info about a battle, namely whether it is currently expired and when it will/did expire
// does NOT include the grace period time that the housekeeping task uses
func GetBattleExpiryInfo(battleID int) (bool, time.Time) {
	setlistsCollection := GocentralDatabase.Collection("setlists")

	var battle models.Setlist

	_ = setlistsCollection.FindOne(context.TODO(), bson.M{"setlist_id": battleID}).Decode(&battle)

	createdTime := time.Unix(battle.Created, 0)

	var expiredTime time.Time

	switch battle.TimeEndUnits {
	case "seconds":
		expiredTime = createdTime.Add(time.Second * time.Duration(battle.TimeEndVal))
	case "minutes":
		expiredTime = createdTime.Add(time.Minute * time.Duration(battle.TimeEndVal))
	case "hours":
		expiredTime = createdTime.Add(time.Hour * time.Duration(battle.TimeEndVal))
	case "days":
		expiredTime = createdTime.Add(time.Hour * 24 * time.Duration(battle.TimeEndVal))
	}

	if time.Now().After(expiredTime) {
		return true, expiredTime
	}

	return false, expiredTime
}

// TODO: might rework or revisit this system when the web app and REST API become more fleshed out
// whether or not a PID has a certain role
func IsPIDInGroup(pid int, groupID string) bool {
	usersCollection := GocentralDatabase.Collection("users")

	// sanity checks
	if pid == 0 || groupID == "" {
		return false
	}

	// find user by pid
	var user models.User
	_ = usersCollection.FindOne(context.TODO(), bson.M{"pid": pid}).Decode(&user)

	// failed to find user
	if user.Username == "" {
		return false
	}

	// check if user is in the group
	for _, group := range user.Groups {
		if group == groupID {
			return true
		}
	}

	return false
}

// checks if a PID is a master user
func IsPIDAMasterUser(pid int) bool {
	usersCollection := GocentralDatabase.Collection("users")

	var user models.User
	err := usersCollection.FindOne(context.TODO(), bson.M{"pid": pid}).Decode(&user)

	if err != nil || user.Username == "" {
		return false
	}

	masterUserPattern := `^Master User \(\d+\)$`
	matched, err := regexp.MatchString(masterUserPattern, user.Username)

	if err != nil || !matched {
		return false
	}

	return true
}

// checks if a username is a master user
func IsUsernameAMasterUser(username string) bool {
	masterUserPattern := `^Master User \(\d+\)$`

	matched, err := regexp.MatchString(masterUserPattern, username)

	return err == nil && matched
}

// gets the Wii friend code from the Master User username, looks up the machine associated with it, and returns its machine ID
func GetMachineIDFromUsername(username string) int {
	machinesCollection := GocentralDatabase.Collection("machines")

	// Define the pattern to capture the digits inside the parentheses
	masterUserPattern := `^Master User \((\d+)\)$`

	// Compile the regex
	re := regexp.MustCompile(masterUserPattern)

	// Find the first match and extract the captured group (the digits)
	matches := re.FindStringSubmatch(username)

	// Check if we have a match
	if len(matches) == 2 {
		var machine models.Machine
		machinesCollection.FindOne(context.TODO(), bson.M{"wii_friend_code": matches[1]}).Decode(&machine)

		if machine.MachineID != 0 {
			return machine.MachineID
		} else {
			return 0
		}
	} else {
		return 0
	}

}
