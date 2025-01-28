package restapi

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	database "rb3server/database"
	"rb3server/models"
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

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	AddStandardHeaders(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func StatsHandler(w http.ResponseWriter, r *http.Request) {
	// return the number of users, scores, machines, and setlists
	scores := database.GocentralDatabase.Collection("scores")
	machines := database.GocentralDatabase.Collection("machines")
	setlists := database.GocentralDatabase.Collection("setlists")
	characters := database.GocentralDatabase.Collection("characters")
	bands := database.GocentralDatabase.Collection("bands")
	gatherings := database.GocentralDatabase.Collection("gatherings")

	scoreCount, _ := scores.CountDocuments(context.Background(), bson.M{})
	machineCount, _ := machines.CountDocuments(context.Background(), bson.M{})
	setlistCount, _ := setlists.CountDocuments(context.Background(), bson.M{})
	characterCount, _ := characters.CountDocuments(context.Background(), bson.M{})
	bandCount, _ := bands.CountDocuments(context.Background(), bson.M{})

	// count the number of active gatherings
	activeGatherings := 0
	activeGatheringsPS3 := 0
	activeGatheringsWii := 0

	cursor, err := gatherings.Find(context.Background(), bson.M{})
	if err != nil {
		AddStandardHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for cursor.Next(context.Background()) {
		var gathering models.Gathering
		cursor.Decode(&gathering)

		// has the gathering been updated in the last 5 minutes?
		// if not, it's not active
		var isGatheringActive bool = false
		if gathering.LastUpdated > time.Now().Unix()-5*60 {
			isGatheringActive = true
		}

		if isGatheringActive {
			activeGatherings++
		}
	}

	// find the song with the most scores
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$song_id"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		{{Key: "$limit", Value: 3}},
	}
	cursor, err = scores.Aggregate(context.Background(), pipeline)
	if err != nil {
		AddStandardHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var mostScoredSongs []struct {
		ID    int   `bson:"_id"`
		Count int64 `bson:"count"`
	}

	err = cursor.All(context.Background(), &mostScoredSongs)
	if err != nil {
		AddStandardHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var songIDs []int
	var songCounts []int64

	for _, song := range mostScoredSongs {
		songIDs = append(songIDs, song.ID)
		songCounts = append(songCounts, song.Count)
	}

	stats := Stats{
		Scores:                     scoreCount,
		Machines:                   machineCount,
		Setlists:                   setlistCount,
		Characters:                 characterCount,
		Bands:                      bandCount,
		ActiveGatherings:           int64(activeGatherings),
		ActiveGatheringsPS3:        int64(activeGatheringsPS3),
		ActiveGatheringsWii:        int64(activeGatheringsWii),
		MostPopularSongIDs:         songIDs,
		MostPopularSongScoreCounts: songCounts,
	}

	w.Header().Set("Content-Type", "application/json")
	AddStandardHeaders(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

func SongListHandler(w http.ResponseWriter, r *http.Request) {

	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Aggregate pipeline to group by song_id to get unique song ids
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{{"_id", "$song_id"}}}},
	}

	cursor, err := scoresCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		AddStandardHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var songs []int
	for cursor.Next(context.TODO()) {
		var result struct {
			ID int `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			AddStandardHeaders(w)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		songs = append(songs, result.ID)
	}

	if err := cursor.Err(); err != nil {
		AddStandardHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	AddStandardHeaders(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string][]int{"songs": songs})
}

func MotdHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var motdInfo models.MOTDInfo

	// get MOTD from database
	motdCollection := database.GocentralDatabase.Collection("motd")

	res := motdCollection.FindOne(context.Background(), bson.D{})

	if res.Err() != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// return json string of MOTD
	err := res.Decode(&motdInfo)

	if err != nil {
		AddStandardHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// use RegEx to grab the MOTD strings
	motdPattern := regexp.MustCompile(`set_motd\s+"([^"]+)"`)
	dlcmotdPattern := regexp.MustCompile(`set_dlcmotd\s+"([^"]+)"`)

	motdMatches := motdPattern.FindStringSubmatch(motdInfo.DTA)
	dlcmotdMatches := dlcmotdPattern.FindStringSubmatch(motdInfo.DTA)

	motd := ""
	dlcmotd := ""

	if len(motdMatches) > 1 {
		motd = motdMatches[1]
	}

	if len(dlcmotdMatches) > 1 {
		dlcmotd = dlcmotdMatches[1]
	}

	response := map[string]string{
		"motd":    motd,
		"dlcmotd": dlcmotd,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		AddStandardHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	AddStandardHeaders(w)
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
