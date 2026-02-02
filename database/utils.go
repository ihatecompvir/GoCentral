package database

import (
	"context"
	"log"
	"math/rand"
	"rb3server/models"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Returns a case-insensitive regex filter for username queries
func CaseInsensitiveUsername(username string) bson.M {
	return bson.M{"username": bson.M{
		"$regex":   "^" + regexp.QuoteMeta(username) + "$",
		"$options": "i",
	}}
}

// config cache
var (
	configCache       *models.Config
	configCacheMu     sync.RWMutex
	configCacheExpiry time.Time
	configCacheTTL    = 30 * time.Second
)

// returns the cached config, or fetches it from the live DB if needed
func GetCachedConfig(ctx context.Context) (*models.Config, error) {
	configCacheMu.RLock()
	if configCache != nil && time.Now().Before(configCacheExpiry) {
		cachedConfig := *configCache
		configCacheMu.RUnlock()
		return &cachedConfig, nil
	}
	configCacheMu.RUnlock()

	configCollection := GocentralDatabase.Collection("config")
	var config models.Config
	err := configCollection.FindOne(ctx, bson.M{}).Decode(&config)
	if err != nil {
		return nil, err
	}

	configCacheMu.Lock()
	configCache = &config
	configCacheExpiry = time.Now().Add(configCacheTTL)
	configCacheMu.Unlock()

	return &config, nil
}

// invalidates the config cache, forcing the next read to fetch from the database
// should always call this after updating the config in the live DB
func InvalidateConfigCache() {
	configCacheMu.Lock()
	configCache = nil
	configCacheMu.Unlock()
}

// atomically increments and returns the next PID
func GetNextPID(ctx context.Context) (int, error) {
	return getNextCounter(ctx, "last_pid")
}

// atomically increments and returns the next band ID
func GetNextBandID(ctx context.Context) (int, error) {
	return getNextCounter(ctx, "last_band_id")
}

// atomically increments and returns the next character ID
func GetNextCharacterID(ctx context.Context) (int, error) {
	return getNextCounter(ctx, "last_character_id")
}

// atomically increments and returns the next setlist ID
func GetNextSetlistID(ctx context.Context) (int, error) {
	return getNextCounter(ctx, "last_setlist_id")
}

// atomically increments and returns the next machine ID
func GetNextMachineID(ctx context.Context) (int, error) {
	return getNextCounter(ctx, "last_machine_id")
}

// atomically increments a counter field and returns the new value.
// this can be used for any generic counter (such as machine ID or setlist ID etc. etc. etc.)
// kind of shit but eh
func getNextCounter(ctx context.Context, field string) (int, error) {
	configCollection := GocentralDatabase.Collection("config")

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	result := configCollection.FindOneAndUpdate(
		ctx,
		bson.M{},
		bson.M{"$inc": bson.M{field: 1}},
		opts,
	)

	if result.Err() != nil {
		return 0, result.Err()
	}

	var config models.Config
	if err := result.Decode(&config); err != nil {
		return 0, err
	}

	// Invalidate the cache since the counter has changed
	InvalidateConfigCache()

	// Return the appropriate field value
	switch field {
	case "last_pid":
		return config.LastPID, nil
	case "last_band_id":
		return config.LastBandID, nil
	case "last_character_id":
		return config.LastCharacterID, nil
	case "last_setlist_id":
		return config.LastSetlistID, nil
	case "last_machine_id":
		return config.LastMachineID, nil
	default:
		return 0, nil
	}
}

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

// returns a map of usernames for a list of PIDs
// useful if you need to resolve multiple usernames at once and want to avoid multiple DB calls
func GetUsernamesByPIDs(ctx context.Context, database *mongo.Database, pids []int) (map[int]string, error) {
	if len(pids) == 0 {
		return make(map[int]string), nil
	}

	usersCollection := database.Collection("users")
	filter := bson.M{"pid": bson.M{"$in": pids}}

	// projection to only fetch pid and username and avoid getting other fields
	opts := options.Find().SetProjection(bson.M{"pid": 1, "username": 1})

	cursor, err := usersCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// create a username map to map PIDs to usernames
	usernameMap := make(map[int]string)
	for cursor.Next(ctx) {
		var user struct {
			PID      int    `bson:"pid"`
			Username string `bson:"username"`
		}
		if err := cursor.Decode(&user); err != nil {
			log.Printf("Failed to decode user for username map: %v", err)
			continue
		}
		usernameMap[user.PID] = user.Username
	}

	return usernameMap, cursor.Err()
}

// returns the pid for a given username (case-insensitive)
func GetPIDForUsername(username string) int {
	var user models.User

	usersCollection := GocentralDatabase.Collection("users")

	res := usersCollection.FindOne(nil, CaseInsensitiveUsername(username))

	if res.Err() != nil {
		return 0
	}

	err := res.Decode(&user)

	if err != nil {
		return 0
	}

	return int(user.PID)
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

// returns a map of usernames with console specific prefixes for a list of PIDs
func GetConsolePrefixedUsernamesByPIDs(ctx context.Context, database *mongo.Database, pids []int) (map[int]string, error) {
	if len(pids) == 0 {
		return make(map[int]string), nil
	}

	usersCollection := database.Collection("users")
	filter := bson.M{"pid": bson.M{"$in": pids}}

	// projection to only fetch pid, username, and console_type
	// i love mongo projections
	opts := options.Find().SetProjection(bson.M{"pid": 1, "username": 1, "console_type": 1})

	cursor, err := usersCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	usernameMap := make(map[int]string)
	for cursor.Next(ctx) {
		var user struct {
			PID         int    `bson:"pid"`
			Username    string `bson:"username"`
			ConsoleType int    `bson:"console_type"`
		}
		if err := cursor.Decode(&user); err != nil {
			log.Printf("Failed to decode user for prefixed username map: %v", err)
			continue
		}

		// do the same prefix logic as the single-user function
		var prefixedName string
		switch user.ConsoleType {
		case 0:
			prefixedName = user.Username + " [360]"
		case 1:
			prefixedName = user.Username + " [PS3]"
		case 2:
			prefixedName = user.Username + " [Wii]"
		case 3:
			prefixedName = user.Username + " [RPCS3]"
		default:
			prefixedName = user.Username
		}
		usernameMap[user.PID] = prefixedName
	}

	return usernameMap, cursor.Err()
}

// returns the name of the band for a given band_id
func GetBandNameForBandID(pid int) string {
	var band models.Band

	bandsCollection := GocentralDatabase.Collection("bands")

	_ = bandsCollection.FindOne(nil, bson.M{"band_id": pid}).Decode(&band)

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

func GetBandNameForOwnerPID(ownerPID int) string {
	var band models.Band

	bandsCollection := GocentralDatabase.Collection("bands")

	_ = bandsCollection.FindOne(nil, bson.M{"owner_pid": ownerPID}).Decode(&band)

	if band.Name != "" {
		return band.Name
	} else {
		username := GetUsernameForPID(ownerPID)
		if username != "" {
			return username + "'s Band"
		} else {
			return "Unnamed Band"
		}
	}
}

// returns a map of band names for a list of owner PIDs
func GetBandNamesByOwnerPIDs(ctx context.Context, database *mongo.Database, ownerPIDs []int) (map[int]string, error) {
	if len(ownerPIDs) == 0 {
		return make(map[int]string), nil
	}

	bandsCollection := database.Collection("bands")
	filter := bson.M{"owner_pid": bson.M{"$in": ownerPIDs}}

	// use a projection to only fetch band_id and name (that is all we need, we do not need band art and etc.)
	opts := options.Find().SetProjection(bson.M{"owner_pid": 1, "name": 1})

	cursor, err := bandsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// create a band name map to map band IDs to names
	bandNameMap := make(map[int]string)
	for cursor.Next(ctx) {
		var band struct {
			OwnerPID int    `bson:"owner_pid"`
			Name     string `bson:"name"`
		}
		if err := cursor.Decode(&band); err != nil {
			log.Printf("Failed to decode band for name map: %v", err)
			continue
		}
		bandNameMap[band.OwnerPID] = band.Name
	}

	return bandNameMap, cursor.Err()
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

		if len(result) == 0 {
			return "Players on this server have no stars because nobody's set a score!"
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

// returns a map of friend PIDs for a given PID, including the PID itself
// this is useful for batch friend lookups in leaderboards
func GetFriendsForPID(ctx context.Context, database *mongo.Database, pid int) (map[int]bool, error) {
	usersCollection := database.Collection("users")

	var user struct {
		Friends []int `bson:"friends"`
	}

	opts := options.FindOne().SetProjection(bson.M{"friends": 1})
	err := usersCollection.FindOne(ctx, bson.M{"pid": pid}, opts).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// user not found, return empty map with just self
			friendsMap := make(map[int]bool, 1)
			friendsMap[pid] = true
			return friendsMap, nil
		}
		return nil, err
	}

	friendsMap := make(map[int]bool, len(user.Friends)+1)
	friendsMap[pid] = true // include self so player appears in their own friends leaderboard
	for _, friendPID := range user.Friends {
		friendsMap[friendPID] = true
	}
	return friendsMap, nil
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
	if pid == 0 || groupID == "" {
		return false
	}

	usersCollection := GocentralDatabase.Collection("users")

	// filter to let the db do the work
	filter := bson.M{"pid": pid, "groups": groupID}
	count, err := usersCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return false
	}
	return count > 0
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

// checks if a PID is currently banned (case-insensitive username comparison)
func IsPIDBanned(pid int) bool {
	config, err := GetCachedConfig(context.Background())
	if err != nil {
		log.Printf("Error getting config for ban check: %v", err)
		return false
	}

	username := GetUsernameForPID(pid)

	for _, bannedPlayer := range config.BannedPlayers {
		if strings.EqualFold(bannedPlayer.Username, username) {
			if bannedPlayer.ExpiresAt.IsZero() || time.Now().Before(bannedPlayer.ExpiresAt) {
				return true
			}
		}
	}

	return false
}

// checks if a username is currently banned (case-insensitive comparison)
func IsUsernameBanned(username string) bool {
	config, err := GetCachedConfig(context.Background())
	if err != nil {
		log.Printf("Error getting config for ban check: %v", err)
		return false
	}
	for _, bannedPlayer := range config.BannedPlayers {
		if strings.EqualFold(bannedPlayer.Username, username) {
			if bannedPlayer.ExpiresAt.IsZero() || time.Now().Before(bannedPlayer.ExpiresAt) {
				return true
			}
		}
	}
	return false
}
