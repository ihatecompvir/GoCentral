package restapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
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
	IsFriend     int    `json:"is_friend"`
	UnnamedBand  int    `json:"unnamed_band"`
	PGUID        string `json:"pguid"`
	ORank        int    `json:"orank"`
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

	var leaderboard []LeaderboardEntry
	rank := (page-1)*pageSize + 1

	for cursor.Next(context.TODO()) {
		var score models.Score
		if err := cursor.Decode(&score); err != nil {
			log.Println("Error decoding score:", err)
			continue
		}

		isBandScore := score.RoleID == 10

		var entry LeaderboardEntry

		if isBandScore {
			entry = LeaderboardEntry{
				PID:          score.OwnerPID,
				Name:         database.GetBandNameForBandID(score.OwnerPID),
				DiffID:       score.DiffID,
				Rank:         rank,
				Score:        score.Score,
				IsPercentile: 0,
				InstMask:     score.InstrumentMask,
				NotesPct:     score.NotesPercent,
				IsFriend:     0,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        rank,
			}
		} else {
			entry = LeaderboardEntry{
				PID:          score.OwnerPID,
				Name:         database.GetConsolePrefixedUsernameForPID(score.OwnerPID),
				DiffID:       score.DiffID,
				Rank:         rank,
				Score:        score.Score,
				IsPercentile: 0,
				InstMask:     score.InstrumentMask,
				NotesPct:     score.NotesPercent,
				IsFriend:     0,
				UnnamedBand:  0,
				PGUID:        "",
				ORank:        rank,
			}
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
