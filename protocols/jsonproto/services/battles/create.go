package battles

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
)

type BattleCreateRequest struct {
	Type         int    `json:"type"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	Description  string `json:"description"`
	Flags        int    `json:"flags"`
	Instrument   int    `json:"instrument"`
	SystemMS     int    `json:"system_ms"`
	MachineID    string `json:"machine_id"`
	SessionGUID  string `json:"session_guid"`
	PID          int    `json:"pid"`
	TimeEndVal   int    `json:"time_end_val"`
	TimeEndUnits string `json:"time_end_units"`
	SongIDs      []int  `json:"song_idXXX"`
}

type BattleCreateResponse struct {
	Success  int `json:"success"`
	BattleID int `json:"battle_id"`
}

type BattleCreateService struct {
}

func (service BattleCreateService) Path() string {
	return "battles/create"
}

func (service BattleCreateService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req BattleCreateRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting battle creation")
		return "", err
	}

	// do a profanity check before updating the setlist
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
				return marshaler.MarshalResponse(service.Path(), []BattleCreateResponse{{0xF, -1}})
			}

			if strings.Contains(lowerName, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []BattleCreateResponse{{0xF, -1}})
			}
		}

		// check description too
		if profanity != "" && req.Description != "" && len(req.Description) >= len(profanity) {
			lowerName := strings.ToLower(req.Description)
			lowerProfanity := strings.ToLower(profanity)

			if lowerName == lowerProfanity {
				return marshaler.MarshalResponse(service.Path(), []BattleCreateResponse{{0x10, -1}})
			}

			if strings.Contains(lowerName, lowerProfanity) {
				return marshaler.MarshalResponse(service.Path(), []BattleCreateResponse{{0x10, -1}})
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
		log.Println("Could not update config in database while creating battle: ", err)
	}

	config.LastSetlistID += 1

	users := database.Collection("users")
	var user models.User
	err = users.FindOne(context.TODO(), bson.M{"pid": req.PID}).Decode(&user)

	if err != nil {
		log.Printf("Could not find user with PID %d, defaulting to \"Player\": %v", req.PID, err)
		user.Username = "Player"
	}

	// write setlist to database
	setlistCollection := database.Collection("setlists")

	var setlist models.Setlist
	setlist.ArtURL = ""
	setlist.Desc = req.Description
	setlist.Title = req.Name
	setlist.Type = 1000
	setlist.Owner = user.Username
	setlist.OwnerGUID = user.GUID
	setlist.SetlistID = config.LastSetlistID
	setlist.PID = req.PID
	setlist.SongIDs = req.SongIDs

	// battle-specific fields
	setlist.TimeEndVal = req.TimeEndVal
	setlist.TimeEndUnits = req.TimeEndUnits
	setlist.Flags = req.Flags
	setlist.Instrument = req.Instrument

	setlist.Created = time.Now().Unix()

	// create song names that are just empty strings for now
	// TODO: create a song ID DB so we can store the proper names
	// perhaps there is some way we can automatically create this, but I don't think the game ever sends song names
	setlist.SongNames = make([]string, len(req.SongIDs))

	update := bson.D{
		{Key: "art_url", Value: setlist.ArtURL},
		{Key: "desc", Value: setlist.Desc},
		{Key: "title", Value: setlist.Title},
		{Key: "type", Value: setlist.Type},
		{Key: "owner", Value: setlist.Owner},
		{Key: "owner_guid", Value: setlist.OwnerGUID},
		{Key: "setlist_id", Value: setlist.SetlistID},
		{Key: "pid", Value: setlist.PID},
		{Key: "s_ids", Value: setlist.SongIDs},
		{Key: "s_names", Value: setlist.SongNames},
		{Key: "shared", Value: "t"},
		{Key: "time_end_val", Value: setlist.TimeEndVal},
		{Key: "time_end_units", Value: setlist.TimeEndUnits},
		{Key: "flags", Value: setlist.Flags},
		{Key: "instrument", Value: setlist.Instrument},
		{Key: "created", Value: setlist.Created},
	}

	_, err = setlistCollection.InsertOne(context.TODO(), update)
	if err != nil {
		log.Printf("Error inserting battle to DB: %s", err)
	}

	res := []BattleCreateResponse{{
		0,
		config.LastSetlistID,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
