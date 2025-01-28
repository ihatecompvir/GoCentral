package setlists

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"
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

	validPIDres, _ := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to update setlist without a valid server-assigned PID, rejecting call")
		return "", nil
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

	// look for the mongo no documents error to determine if we are inserting a new setlist
	// maybe there is a better way to do this?
	isNewSetlist := err == mongo.ErrNoDocuments

	// If it's an existing setlist, perform access control checks
	if !isNewSetlist {
		// Check if the setlist we are attempting to update is not a battle, as battles cannot be updated after-the-fact
		if setlist.Type == 1000 || setlist.Type == 1001 || setlist.Type == 1002 {
			log.Printf("Player with PID %d attempted to update a battle setlist, rejecting", req.PID)
			return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0x16}})
		}

		// Access controls so that only the owner of the setlist can update it
		if setlist.Owner != client.Username || setlist.PID != int(client.PlayerID()) {
			log.Printf("Player with PID %d attempted to update a setlist that does not belong to them, rejecting", req.PID)
			return marshaler.MarshalResponse(service.Path(), []SetlistUpdateResponse{{0x16}})
		}
	}

	if isNewSetlist {
		// Only update last_setlist_id if we are inserting a new setlist
		config.LastSetlistID += 1

		_, err = configCollection.UpdateOne(
			context.TODO(),
			bson.M{},
			bson.D{
				{"$set", bson.D{{"last_setlist_id", config.LastSetlistID}}},
			},
		)

		if err != nil {
			log.Println("Could not update config in database while updating character: ", err)
		}

		setlist.SetlistID = config.LastSetlistID
	}

	setlist.ArtURL = ""
	setlist.Desc = req.Description
	setlist.Title = req.Name
	setlist.Type = 0
	setlist.Owner = user.Username
	setlist.OwnerGUID = user.GUID
	setlist.PID = req.PID
	setlist.SongIDs = req.SongIDs

	if setlist.Created == 0 {
		setlist.Created = time.Now().Unix()
	}

	setlist.SongNames = make([]string, len(req.SongIDs))

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
