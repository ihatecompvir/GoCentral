package setlists

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"strings"
	"time"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SetlistUpdateRequest struct {
	Type        int    `json:"type"`
	Name        string `json:"name"`
	Region      string `json:"region"`
	Description string `json:"description"`
	Flags       int    `json:"flags"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	Shared      string `json:"shared"`
	ListGUID    string `json:"list_guid"`
	SongIDs     []int  `json:"song_idXXX"`
}

type SetlistUpdateResponse struct {
	RetCode int `json:"ret_code"`
}

type SetlistUpdateService struct {
}

func (service SetlistUpdateService) Path() string {
	return "setlists/update"
}

func (service SetlistUpdateService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req SetlistUpdateRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting setlist update")
		return "", err
	}

	// Do a profanity check before updating the setlist
	var config models.Config
	configCollection := database.Collection("config")
	err = configCollection.FindOne(context.TODO(), bson.M{}).Decode(&config)
	if err != nil {
		log.Printf("Could not get config %v\n", err)
	}

	// Check if the setlist name or description contain anything in the profanity list
	for _, profanity := range config.ProfanityList {
		if profanity != "" && req.Name != "" && len(req.Name) >= len(profanity) {
			lowerName := strings.ToLower(req.Name)
			lowerProfanity := strings.ToLower(profanity)

			if lowerName == lowerProfanity || strings.Contains(lowerName, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0xF}})
			}
		}

		if profanity != "" && req.Description != "" && len(req.Description) >= len(profanity) {
			lowerDesc := strings.ToLower(req.Description)
			lowerProfanity := strings.ToLower(profanity)

			if lowerDesc == lowerProfanity || strings.Contains(lowerDesc, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0x10}})
			}
		}
	}

	log.Println("Attempting to update setlist with list_guid %s", req.ListGUID)

	_, err = configCollection.UpdateOne(
		context.TODO(),
		bson.M{},
		bson.D{
			{"$set", bson.D{{"last_setlist_id", config.LastSetlistID + 1}}},
		},
	)

	if err != nil {
		log.Println("Could not update config in database while updating character: ", err)
	}

	config.LastSetlistID += 1

	users := database.Collection("users")
	var user models.User
	err = users.FindOne(context.TODO(), bson.M{"pid": req.PID}).Decode(&user)
	if err != nil {
		log.Printf("Could not find user with PID %d, defaulting to \"Player\": %v", req.PID, err)
		user.Username = "Player"
	}

	// Write setlist to database
	setlistCollection := database.Collection("setlists")

	var setlist models.Setlist
	err = setlistCollection.FindOne(context.TODO(), bson.M{"guid": req.ListGUID}).Decode(&setlist)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Printf("Error finding setlist: %s", err)
		return "", err
	}

	setlist.ArtURL = ""
	setlist.Desc = req.Description
	setlist.Title = req.Name
	setlist.Type = 0
	setlist.Owner = user.Username
	setlist.OwnerGUID = user.GUID
	setlist.SetlistID = config.LastSetlistID
	setlist.PID = req.PID
	setlist.SongIDs = req.SongIDs

	if setlist.Created == 0 {
		setlist.Created = time.Now().Unix()
	}

	// Create song names that are just empty strings for now
	for i := 0; i < len(req.SongIDs); i++ {
		setlist.SongNames = append(setlist.SongNames, "")
	}

	filter := bson.M{"guid": req.ListGUID}
	update := bson.M{
		"$set": bson.M{
			"art_url":    setlist.ArtURL,
			"desc":       setlist.Desc,
			"title":      setlist.Title,
			"type":       setlist.Type,
			"owner":      setlist.Owner,
			"owner_guid": setlist.OwnerGUID,
			"setlist_id": setlist.SetlistID,
			"pid":        setlist.PID,
			"s_ids":      setlist.SongIDs,
			"s_names":    setlist.SongNames,
			"guid":       req.ListGUID,
			"shared":     req.Shared,
			"created":    setlist.Created,
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err = setlistCollection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Printf("Error upserting setlist: %s", err)
	}

	res := []SetlistUpdateResponse{{
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
