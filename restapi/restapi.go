package restapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	database "rb3server/database"
	"rb3server/models"
)

var (
	motdPattern    = regexp.MustCompile(`set_motd\s+"([^"]+)"`)
	dlcmotdPattern = regexp.MustCompile(`set_dlcmotd\s+"([^"]+)"`)
)

type Stats struct {
	Scores                     int64   `json:"scores"`
	Machines                   int64   `json:"machines"`
	Setlists                   int64   `json:"setlists"`
	Characters                 int64   `json:"characters"`
	Bands                      int64   `json:"bands"`
	ActiveGatherings           int64   `json:"active_gatherings"`
	ActiveGatheringsPS3        int64   `json:"active_gatherings_ps3"`
	ActiveGatheringsWii        int64   `json:"active_gatherings_wii"`
	MostPopularSongIDs         []int   `json:"most_popular_song_ids"`
	MostPopularSongScoreCounts []int64 `json:"most_popular_song_score_counts"`
}

type LeaderboardEntry struct {
	PID          int    `json:"pid"`
	Name         string `json:"name"`
	DiffID       int    `json:"diff_id"`
	Rank         int    `json:"rank"`
	Score        int    `json:"score"`
	IsPercentile int    `json:"is_percentile"`
	InstMask     int    `json:"inst_mask"`
	NotesPct     int    `json:"notes_pct"`
	UnnamedBand  int    `json:"unnamed_band"`
	PGUID        string `json:"pguid"`
	ORank        int    `json:"orank"`
	Stars        int    `json:"stars"`
}

type BattleLeaderboardEntry struct {
	PID   int    `json:"pid"`
	Name  string `json:"name"`
	Score int    `json:"score"`
	Rank  int    `json:"rank"`
	ORank int    `json:"orank"`
}

type GlobalBattleInfo struct {
	BattleID    int    `json:"battle_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartsAt    int64  `json:"starts_at"`
	ExpiresAt   int64  `json:"expires_at"`
	Instrument  int    `json:"instrument"`
	SongIDs     []int  `json:"song_ids"`
}

type CreateBattleRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SongIDs     []int     `json:"song_ids"`
	ExpiresAt   time.Time `json:"expires_at"`
	Instrument  int       `json:"instrument"`
	Flags       int       `json:"flags"`
}

type DeleteBattleRequest struct {
	BattleID int `json:"battle_id"`
}

type BanPlayerRequest struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
	Duration string `json:"duration"` // e.g., "24h", "7d", "permanent"
}

type UnbanPlayerRequest struct {
	Username string `json:"username"`
}

type DeletePlayerScoresRequest struct {
	Username string `json:"username"`
}

func AddStandardHeaders(writer http.ResponseWriter) {
	headers := map[string]string{
		"Server":                      "GoCentral",
		"X-Clacks-Overhead":           "GNU maxton",
		"Access-Control-Allow-Origin": "*",
	}

	for key, value := range headers {
		writer.Header().Set(key, value)
	}
}

// Checks for a valid API token in the Authorization header.
func AdminTokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// must be in the pretty standard bearer auth token format
		splitToken := strings.Split(authHeader, "Bearer ")
		if len(splitToken) != 2 {
			sendError(w, http.StatusInternalServerError, "Could not verify authorization")
			return
		}
		token := splitToken[1]

		config, err := database.GetCachedConfig(r.Context())
		if err != nil {
			log.Printf("ERROR: could not get config for auth: %v", err)
			sendError(w, http.StatusInternalServerError, "Could not verify authorization")
			return
		}

		if config.AdminAPIToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(config.AdminAPIToken)) != 1 {
			sendError(w, http.StatusInternalServerError, "Could not verify authorization")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// JSON helper methods to not repeat the same code in every handler

// Sends a JSON response with the given status code and payload.
func sendJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("ERROR: could not write json response: %v", err)
	}
}

// Sends an error response with the given status code and message.
func sendError(w http.ResponseWriter, statusCode int, message string) {
	sendJSON(w, statusCode, map[string]string{"error": message})
}

// Handles the health check endpoint to verify if the database is reachable.
// If somehow the DB has gone down, this will return a 503 Service Unavailable status so clients know that the service is not operational.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := database.GocentralDatabase.Client().Ping(ctx, nil); err != nil {
		sendError(w, http.StatusServiceUnavailable, "database is not available")
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func StatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var stats Stats
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	recordError := func(err error) {
		if err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}
	}

	collections := map[string]*mongo.Collection{
		"scores":     database.GocentralDatabase.Collection("scores"),
		"machines":   database.GocentralDatabase.Collection("machines"),
		"setlists":   database.GocentralDatabase.Collection("setlists"),
		"characters": database.GocentralDatabase.Collection("characters"),
		"bands":      database.GocentralDatabase.Collection("bands"),
	}

	// count up the documents in each collection to get how many X there are
	for name, coll := range collections {
		wg.Add(1)
		go func(name string, coll *mongo.Collection) {
			defer wg.Done()
			count, err := coll.CountDocuments(ctx, bson.M{})
			if err != nil {
				recordError(err)
				return
			}
			mu.Lock()
			switch name {
			case "scores":
				stats.Scores = count
			case "machines":
				stats.Machines = count
			case "setlists":
				stats.Setlists = count
			case "characters":
				stats.Characters = count
			case "bands":
				stats.Bands = count
			}
			mu.Unlock()
		}(name, coll)
	}

	// do the work on the DB to check for active gatherings rather than how we did it before
	// active = updated wiuthin last 5 minutes
	gatherings := database.GocentralDatabase.Collection("gatherings")
	unixTimeFiveminutesAgo := time.Now().Unix() - 5*60
	filter := bson.M{"last_updated": bson.M{"$gt": unixTimeFiveminutesAgo}}

	wg.Add(1)
	go func() {
		defer wg.Done()
		count, err := gatherings.CountDocuments(ctx, filter)
		recordError(err)
		stats.ActiveGatherings = count
	}()

	// get three most popular songs
	// why three? idk lol
	// TODO: allow this to be configurable for up to 10 or so, so we can show more popular songs
	wg.Add(1)
	go func() {
		defer wg.Done()
		scores := database.GocentralDatabase.Collection("scores")
		pipeline := mongo.Pipeline{
			{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$song_id"}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
			{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
			{{Key: "$limit", Value: 3}},
		}
		cursor, err := scores.Aggregate(ctx, pipeline)
		if err != nil {
			recordError(err)
			return
		}
		defer cursor.Close(ctx)

		var mostScoredSongs []struct {
			ID    int   `bson:"_id"`
			Count int64 `bson:"count"`
		}
		if err := cursor.All(ctx, &mostScoredSongs); err != nil {
			recordError(err)
			return
		}

		mu.Lock()
		for _, song := range mostScoredSongs {
			stats.MostPopularSongIDs = append(stats.MostPopularSongIDs, song.ID)
			stats.MostPopularSongScoreCounts = append(stats.MostPopularSongScoreCounts, song.Count)
		}
		mu.Unlock()
	}()

	wg.Wait()

	if firstErr != nil {
		log.Printf("ERROR: could not fetch stats: %v", firstErr)
		sendError(w, http.StatusInternalServerError, "Failed to retrieve statistics")
		return
	}

	sendJSON(w, http.StatusOK, stats)
}

func SongListHandler(w http.ResponseWriter, r *http.Request) {
	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Aggregate pipeline to group by song_id to get unique song ids
	pipeline := mongo.Pipeline{{{"$group", bson.D{{"_id", "$song_id"}}}}}

	cursor, err := scoresCollection.Aggregate(r.Context(), pipeline)
	if err != nil {
		log.Printf("ERROR: failed to aggregate songs: %v", err)
		sendError(w, http.StatusInternalServerError, "Could not retrieve song list")
		return
	}
	defer cursor.Close(r.Context())

	var songs []int
	for cursor.Next(r.Context()) {
		var result struct {
			ID int `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("ERROR: failed to decode song id: %v", err)
			sendError(w, http.StatusInternalServerError, "Could not process song list")
			return
		}
		songs = append(songs, result.ID)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("ERROR: cursor error in song list: %v", err)
		sendError(w, http.StatusInternalServerError, "Could not read song list from database")
		return
	}

	sendJSON(w, http.StatusOK, map[string][]int{"songs": songs})
}

func MotdHandler(w http.ResponseWriter, r *http.Request) {
	var motdInfo models.MOTDInfo
	motdCollection := database.GocentralDatabase.Collection("motd")

	err := motdCollection.FindOne(r.Context(), bson.D{}).Decode(&motdInfo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			sendError(w, http.StatusNotFound, "MOTD not found")
			return
		}
		log.Printf("ERROR: failed to find motd: %v", err)
		sendError(w, http.StatusInternalServerError, "Could not retrieve MOTD")
		return
	}

	// try to pull MOTD and DLC MOTD from the DTA field using some regex
	// this won't return the full DTA so if there is any extra shit this wont get it
	motdMatches := motdPattern.FindStringSubmatch(motdInfo.DTA)
	dlcmotdMatches := dlcmotdPattern.FindStringSubmatch(motdInfo.DTA)

	motd := ""
	if len(motdMatches) > 1 {
		motd = motdMatches[1]
	}

	dlcmotd := ""
	if len(dlcmotdMatches) > 1 {
		dlcmotd = dlcmotdMatches[1]
	}

	response := map[string]string{
		"motd":    motd,
		"dlcmotd": dlcmotd,
	}

	sendJSON(w, http.StatusOK, response)
}

// Returns the list of Harmonix battles and any associated data
func BattleListHandler(w http.ResponseWriter, r *http.Request) {
	setlistsCollection := database.GocentralDatabase.Collection("setlists")
	ctx := r.Context()

	// filter for Harmonix battles
	filter := bson.M{"type": 1002}
	cursor, err := setlistsCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("ERROR: could not query global battles: %v", err)
		sendError(w, http.StatusInternalServerError, "Could not retrieve global battles")
		return
	}
	defer cursor.Close(ctx)

	var battles []GlobalBattleInfo
	for cursor.Next(ctx) {
		var setlist models.Setlist
		if err := cursor.Decode(&setlist); err != nil {
			log.Printf("ERROR: failed to decode battle setlist: %v", err)
			continue
		}

		_, expiresAt := database.GetBattleExpiryInfo(setlist.SetlistID)

		startsAt := time.Unix(setlist.Created, 0).UTC()

		battleInfo := GlobalBattleInfo{
			BattleID:    setlist.SetlistID,
			Title:       setlist.Title,
			Description: setlist.Desc,
			StartsAt:    startsAt.Unix(),  // give unix time for start rather than some kind of string
			ExpiresAt:   expiresAt.Unix(), // give unix time for expiry rather than some kind of string
			Instrument:  setlist.Instrument,
			SongIDs:     setlist.SongIDs,
		}
		battles = append(battles, battleInfo)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("ERROR: cursor error in global battles: %v", err)
		sendError(w, http.StatusInternalServerError, "Could not read global battles from database")
		return
	}

	sendJSON(w, http.StatusOK, map[string][]GlobalBattleInfo{"battles": battles})
}

func LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	AddStandardHeaders(w)

	songIDStr := r.URL.Query().Get("song_id")
	roleIDStr := r.URL.Query().Get("role_id")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	if songIDStr == "" || roleIDStr == "" {
		sendError(w, http.StatusBadRequest, "role_id and song_id are required")
		return
	}

	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid song_id")
		return
	}

	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid role_id")
		return
	}

	page := 1
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			sendError(w, http.StatusBadRequest, "Invalid page number")
			return
		}
	}

	pageSize := 20 // Default page size
	if pageSizeStr != "" {
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 { // limit to 100 to avoid large result set
			sendError(w, http.StatusBadRequest, "Invalid page_size")
			return
		}
	}

	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)
	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Find scores for the song and role ID, sorted by score descending
	findOptions := options.Find().SetSort(bson.M{"score": -1}).SetSkip(skip).SetLimit(limit)
	cursor, err := scoresCollection.Find(context.TODO(), bson.M{"song_id": songID, "role_id": roleID}, findOptions)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to query leaderboard data")
		return
	}
	defer cursor.Close(context.TODO())

	var scores []models.Score
	var bandPIDs []int
	var userPIDs []int

	for cursor.Next(context.TODO()) {
		var score models.Score
		if err := cursor.Decode(&score); err != nil {
			log.Println("Error decoding score:", err)
			continue
		}
		scores = append(scores, score)

		isBandScore := score.RoleID == 10
		if isBandScore {
			bandPIDs = append(bandPIDs, score.OwnerPID)
		} else {
			userPIDs = append(userPIDs, score.OwnerPID)
		}
	}

	// fetch all names at onnce in a single shot
	ctx := context.TODO()
	bandNameMap, err := database.GetBandNamesByOwnerPIDs(ctx, database.GocentralDatabase, bandPIDs)
	if err != nil {
		log.Println("Error fetching band names:", err)
		bandNameMap = make(map[int]string)
	}

	userNameMap, err := database.GetConsolePrefixedUsernamesByPIDs(ctx, database.GocentralDatabase, userPIDs)
	if err != nil {
		log.Println("Error fetching usernames:", err)
		userNameMap = make(map[int]string)
	}

	// use cached stuff
	// TODO: this is a bit messy
	var leaderboard []LeaderboardEntry
	rank := (page-1)*pageSize + 1

	for _, score := range scores {
		isBandScore := score.RoleID == 10
		var entryName string

		if isBandScore {
			if name, ok := bandNameMap[score.OwnerPID]; ok {
				entryName = name
			} else {
				entryName = "Unnamed Band"
			}
		} else {
			if name, ok := userNameMap[score.OwnerPID]; ok {
				entryName = name
			} else {
				entryName = "Unnamed Player"
			}
		}

		entry := LeaderboardEntry{
			PID:          score.OwnerPID,
			Name:         entryName,
			DiffID:       score.DiffID,
			Rank:         rank,
			Score:        score.Score,
			IsPercentile: 0,
			InstMask:     score.InstrumentMask,
			NotesPct:     score.NotesPercent,
			UnnamedBand:  0,
			PGUID:        "",
			ORank:        rank,
			Stars:        score.Stars,
		}
		leaderboard = append(leaderboard, entry)
		rank++
	}

	if err := cursor.Err(); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to query leaderboard data")
		return
	}
	sendJSON(w, http.StatusOK, map[string][]LeaderboardEntry{"leaderboard": leaderboard})
}

func BattleLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	AddStandardHeaders(w)

	battleIDStr := r.URL.Query().Get("battle_id")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	if battleIDStr == "" {
		sendError(w, http.StatusBadRequest, "battle_id is required")
		return
	}

	battleID, err := strconv.Atoi(battleIDStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid battle_id")
		return
	}

	page := 1
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			sendError(w, http.StatusBadRequest, "Invalid page number")
			return
		}
	}

	pageSize := 20
	if pageSizeStr != "" {
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 {
			sendError(w, http.StatusBadRequest, "Invalid page_size")
			return
		}
	}

	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)
	scoresCollection := database.GocentralDatabase.Collection("scores")

	findOptions := options.Find().SetSort(bson.M{"score": -1}).SetSkip(skip).SetLimit(limit)
	cursor, err := scoresCollection.Find(context.TODO(), bson.M{"battle_id": battleID}, findOptions)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to query battle leaderboard data")
		return
	}
	defer cursor.Close(context.TODO())

	var scores []models.Score
	var bandPIDs []int
	var userPIDs []int

	for cursor.Next(context.TODO()) {
		var score models.Score
		if err := cursor.Decode(&score); err != nil {
			log.Println("Error decoding score:", err)
			continue
		}
		scores = append(scores, score)

		isBandScore := score.RoleID == 10
		if isBandScore {
			bandPIDs = append(bandPIDs, score.OwnerPID)
		} else {
			userPIDs = append(userPIDs, score.OwnerPID)
		}
	}

	// get all names in a single shot
	ctx := context.TODO()
	bandNameMap, err := database.GetBandNamesByOwnerPIDs(ctx, database.GocentralDatabase, bandPIDs)
	if err != nil {
		log.Println("Error fetching band names:", err)
		bandNameMap = make(map[int]string)
	}

	userNameMap, err := database.GetConsolePrefixedUsernamesByPIDs(ctx, database.GocentralDatabase, userPIDs)
	if err != nil {
		log.Println("Error fetching usernames:", err)
		userNameMap = make(map[int]string)
	}

	var leaderboard []BattleLeaderboardEntry
	rank := (page-1)*pageSize + 1

	for _, score := range scores {
		isBandScore := score.RoleID == 10
		var entryName string

		if isBandScore {
			if name, ok := bandNameMap[score.OwnerPID]; ok {
				entryName = name
			} else {
				entryName = "Unnamed Band"
			}
		} else {
			if name, ok := userNameMap[score.OwnerPID]; ok {
				entryName = name
			} else {
				entryName = "Unnamed Player"
			}
		}

		entry := BattleLeaderboardEntry{
			PID:   score.OwnerPID,
			Name:  entryName,
			Rank:  rank,
			Score: score.Score,
			ORank: rank,
		}
		leaderboard = append(leaderboard, entry)
		rank++
	}

	if err := cursor.Err(); err != nil {
		sendError(w, http.StatusInternalServerError, "Cursor error while fetching battle leaderboard data")
		return
	}
	sendJSON(w, http.StatusOK, map[string][]BattleLeaderboardEntry{"leaderboard": leaderboard})
}

// Handles the creation of a new Harmonix battle.
// This endpoint will create a global Harmonix battle that is shown to all users.
// It requires a valid admin API token in the Authorization header.
func CreateBattleHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateBattleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Input Validation
	if req.Title == "" || len(req.SongIDs) == 0 {
		sendError(w, http.StatusBadRequest, "Title and at least one song_id are required")
		return
	}
	if req.ExpiresAt.Before(time.Now()) {
		sendError(w, http.StatusBadRequest, "expires_at must be in the future")
		return
	}

	ctx := r.Context()

	// race condition prevention
	newBattleID, err := database.GetNextSetlistID(ctx)
	if err != nil {
		log.Printf("ERROR: Could not get next setlist ID: %v", err)
		sendError(w, http.StatusInternalServerError, "Could not generate battle ID")
		return
	}

	// calculate the expiry duration, in seconds
	durationSeconds := int(time.Until(req.ExpiresAt).Seconds())
	if durationSeconds <= 0 {
		sendError(w, http.StatusBadRequest, "expires_at must be in the future")
		return
	}

	setlistCollection := database.GocentralDatabase.Collection("setlists")

	newBattle := models.Setlist{
		SetlistID:    newBattleID,
		PID:          0,
		Title:        req.Title,
		Desc:         req.Description,
		Type:         1002,
		Owner:        "Harmonix",
		Shared:       "t",
		SongIDs:      req.SongIDs,
		SongNames:    make([]string, len(req.SongIDs)),
		TimeEndVal:   durationSeconds,
		TimeEndUnits: "seconds",
		Flags:        req.Flags,
		Instrument:   req.Instrument,
		Created:      time.Now().Unix(),
	}

	_, err = setlistCollection.InsertOne(ctx, newBattle)
	if err != nil {
		log.Printf("Could not insert new battle: %v", err)
		sendError(w, http.StatusInternalServerError, "Failed to create battle")
		return
	}

	log.Printf("Successfully created global battle #%d titled '%s'", newBattleID, req.Title)
	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"success":   true,
		"battle_id": newBattleID,
	})
}

// Deletes a global battle and any associated scores.
// Requires a valid admin API token in the Authorization header.
func DeleteBattleHandler(w http.ResponseWriter, r *http.Request) {
	var req DeleteBattleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.BattleID == 0 {
		sendError(w, http.StatusBadRequest, "battle_id is required")
		return
	}

	ctx := r.Context()
	setlists := database.GocentralDatabase.Collection("setlists")
	scores := database.GocentralDatabase.Collection("scores")

	res, err := setlists.DeleteOne(ctx, bson.M{"setlist_id": req.BattleID, "type": 1002})
	if err != nil {
		log.Printf("ERROR: failed to delete battle %d: %v", req.BattleID, err)
		sendError(w, http.StatusInternalServerError, "Failed to delete battle")
		return
	}
	if res.DeletedCount == 0 {
		sendError(w, http.StatusNotFound, "Battle not found")
		return
	}

	scoreRes, err := scores.DeleteMany(ctx, bson.M{"battle_id": req.BattleID})
	if err != nil {
		log.Printf("WARN: battle %d deleted, but failed to delete scores: %v", req.BattleID, err)
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"success":        true,
			"battle_id":      req.BattleID,
			"scores_deleted": 0,
			"warning":        "Battle deleted, but score cleanup failed",
		})
		return
	}

	log.Printf("Deleted battle #%d (scores cleaned: %d)", req.BattleID, scoreRes.DeletedCount)
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"battle_id":      req.BattleID,
		"scores_deleted": scoreRes.DeletedCount,
	})
}

// Handles banning a player. This will add a new ban record to the config's banned_players array.
func BanPlayerHandler(w http.ResponseWriter, r *http.Request) {
	var req BanPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Username == "" {
		sendError(w, http.StatusBadRequest, "Username is required")
		return
	}

	var expiresAt time.Time
	// zero time means permanent
	if req.Duration != "permanent" && req.Duration != "" {
		duration, err := time.ParseDuration(req.Duration)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid duration format. Use '1h', '24h', '7d', etc., or 'permanent'.")
			return
		}
		expiresAt = time.Now().Add(duration)
	}

	newBan := models.BannedPlayer{
		Username:  strings.ToLower(req.Username),
		Reason:    req.Reason,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	configCollection := database.GocentralDatabase.Collection("config")
	filter := bson.M{}

	update := bson.M{"$push": bson.M{"banned_players": newBan}}

	if _, err := configCollection.UpdateOne(r.Context(), filter, update); err != nil {
		log.Printf("ERROR: could not add ban for %s: %v", req.Username, err)
		sendError(w, http.StatusInternalServerError, "Failed to update ban list")
		return
	}

	log.Printf("Added new ban record for player: %s. Reason: %s. Duration: %s", req.Username, req.Reason, req.Duration)
	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "New ban record for player " + req.Username + " has been created.",
	})
}

// Handles "unbanning" a player by expiring their most recent ban, preserving the record.
func UnbanPlayerHandler(w http.ResponseWriter, r *http.Request) {
	var req UnbanPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Username == "" {
		sendError(w, http.StatusBadRequest, "Username is required")
		return
	}

	ctx := r.Context()
	configCollection := database.GocentralDatabase.Collection("config")

	var config models.Config
	if err := configCollection.FindOne(ctx, bson.M{}).Decode(&config); err != nil {
		sendError(w, http.StatusInternalServerError, "Could not retrieve config to process unban")
		return
	}

	// find the most recent active ban for this user (case-insensitive)
	latestBanIndex := -1
	var latestBanTime time.Time
	for i, ban := range config.BannedPlayers {
		if strings.EqualFold(ban.Username, req.Username) {
			// check if it is still active
			if ban.ExpiresAt.IsZero() || time.Now().Before(ban.ExpiresAt) {
				// verify if it's the latest one
				if latestBanIndex == -1 || ban.CreatedAt.After(latestBanTime) {
					latestBanIndex = i
					latestBanTime = ban.CreatedAt
				}
			}
		}
	}

	if latestBanIndex == -1 {
		sendError(w, http.StatusNotFound, "No active ban found for player "+req.Username)
		return
	}

	// expire the ban by setting its expiration to the current time
	config.BannedPlayers[latestBanIndex].ExpiresAt = time.Now()

	// update the entire document
	filter := bson.M{"_id": config.ID}
	update := bson.M{"$set": bson.M{"banned_players": config.BannedPlayers}}
	result, err := configCollection.UpdateOne(ctx, filter, update)
	if err != nil || result.ModifiedCount == 0 {
		log.Printf("ERROR: could not expire ban for player %s: %v", req.Username, err)
		sendError(w, http.StatusInternalServerError, "Failed to update ban list")
		return
	}

	log.Printf("Unbanned player: %s by expiring their latest ban.", req.Username)
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Player " + req.Username + "'s most recent ban has been expired.",
	})
}

// Handles deleting all scores for a specific player.
func DeletePlayerScoresHandler(w http.ResponseWriter, r *http.Request) {
	var req DeletePlayerScoresRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Username == "" {
		sendError(w, http.StatusBadRequest, "Username is required")
		return
	}

	pid := database.GetPIDForUsername(req.Username)
	if pid == 0 {
		sendError(w, http.StatusNotFound, "User not found")
		return
	}

	scoresCollection := database.GocentralDatabase.Collection("scores")
	res, err := scoresCollection.DeleteMany(r.Context(), bson.M{"pid": pid})

	if err != nil {
		log.Printf("ERROR: could not delete scores for user %s: %v", req.Username, err)
		sendError(w, http.StatusInternalServerError, "Failed to delete user scores")
		return
	}

	log.Printf("Deleted %d scores for user %s (PID %d)", res.DeletedCount, req.Username, pid)
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"scores_deleted": res.DeletedCount,
	})
}

// Lists all currently active bans.
func ListBannedPlayersHandler(w http.ResponseWriter, r *http.Request) {
	var config models.Config
	configCollection := database.GocentralDatabase.Collection("config")

	err := configCollection.FindOne(r.Context(), bson.D{}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			sendJSON(w, http.StatusOK, map[string][]models.BannedPlayer{"banned_players": {}})
			return
		}
		log.Printf("ERROR: could not get config for listing banned players: %v", err)
		sendError(w, http.StatusInternalServerError, "Could not retrieve ban list")
		return
	}

	activeBans := []models.BannedPlayer{}
	if config.BannedPlayers != nil {
		for _, ban := range config.BannedPlayers {
			if ban.ExpiresAt.IsZero() || time.Now().Before(ban.ExpiresAt) {
				activeBans = append(activeBans, ban)
			}
		}
	}

	sendJSON(w, http.StatusOK, map[string][]models.BannedPlayer{"banned_players": activeBans})
}
