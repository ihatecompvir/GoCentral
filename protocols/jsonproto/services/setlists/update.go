package setlists

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"strings"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// do a profanity check before updating the band
	var config models.Config
	configCollection := database.Collection("config")
	err = configCollection.FindOne(context.TODO(), bson.M{}).Decode(&config)
	if err != nil {
		log.Printf("Could not get config %v\n", err)
	}

	// check if the setlist name or description contain anything in the profanity list
	// NOTE: exercise caution with the profanity list. Putting "ass" on the list would mean that a setlist name like "Band Assistant" is not allowed.
	// use your best judgment, it's up to you to define your own profanity list as a server host, GoCentral does not and will not ship with one
	for _, profanity := range config.ProfanityList {
		if profanity != "" && req.Name != "" && len(req.Name) >= len(profanity) {
			lowerName := strings.ToLower(req.Name)
			lowerProfanity := strings.ToLower(profanity)

			if lowerName == lowerProfanity {
				return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0xF}})
			}

			if strings.Contains(lowerName, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0xF}})
			}
		}

		// check description too
		if profanity != "" && req.Description != "" && len(req.Description) >= len(profanity) {
			lowerName := strings.ToLower(req.Description)
			lowerProfanity := strings.ToLower(profanity)

			if lowerName == lowerProfanity {
				return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0x10}})
			}

			if strings.Contains(lowerName, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0x10}})
			}
		}
	}

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

	// write setlist to database
	setlistCollection := database.Collection("setlists")

	var setlist models.Setlist
	setlist.ArtURL = ""
	setlist.Desc = req.Description
	setlist.Title = req.Name
	setlist.Type = 0
	setlist.Owner = "Test"
	setlist.OwnerGUID = "00000000-0000-0000-0000-000000000000"
	setlist.GUID = primitive.NewObjectID().Hex()
	setlist.SetlistID = config.LastSetlistID
	setlist.PID = req.PID
	setlist.SongIDs = req.SongIDs

	// create song names that are just empty strings for now
	// TODO: create a song ID DB so we can store the proper names
	// perhaps there is some way we can automatically create this
	for i := 0; i < len(req.SongIDs); i++ {
		setlist.SongNames = append(setlist.SongNames, "")
	}

	filter := bson.M{"guid": setlist.GUID}
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
