package songlists

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"time"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type GetSonglistsRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
}

type GetSonglistResponse struct {
	SetlistID int      `json:"id"`
	PID       int      `json:"pid"`
	Title     string   `json:"title"`
	Desc      string   `json:"desc"`
	Type      int      `json:"type"`
	Owner     string   `json:"owner"`
	OwnerGUID string   `json:"owner_guid"`
	GUID      string   `json:"guid"`
	ArtURL    string   `json:"art_url"`
	SongIDs   []int    `json:"s_idXXX"`
	SongNames []string `json:"s_nameXXX"`
}

type GetBattleSonglistResponse struct {
	SetlistID   int      `json:"id"`
	PID         int      `json:"pid"`
	Title       string   `json:"title"`
	Desc        string   `json:"desc"`
	Type        int      `json:"type"`
	Owner       string   `json:"owner"`
	OwnerGUID   string   `json:"owner_guid"`
	GUID        string   `json:"guid"`
	ArtURL      string   `json:"art_url"`
	SecondsLeft int      `json:"seconds_left"`
	ValidInstr  int      `json:"valid_instr"`
	SongIDs     []int    `json:"s_idXXX"`
	SongNames   []string `json:"s_nameXXX"`
}

type GetSonglistsService struct {
}

func (service GetSonglistsService) Path() string {
	return "songlists/get"
}

func (service GetSonglistsService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req GetSonglistsRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for songlists")
		return "", err
	}

	setlistCollection := database.Collection("setlists")

	setlistCursor, err := setlistCollection.Find(context.TODO(), bson.D{{"shared", "t"}, {"pid", bson.D{{"$ne", req.PID000}}}})

	if err != nil {
		log.Printf("Error getting songlists: %s", err)
	}

	jsonStrings := []string{}

	for setlistCursor.Next(context.TODO()) {
		var setlistToCopy models.Setlist

		setlistCursor.Decode(&setlistToCopy)

		// normal setlist
		if setlistToCopy.Type == 1 || setlistToCopy.Type == 2 || setlistToCopy.Type == 0 {
			var setlist GetSonglistResponse
			setlist.ArtURL = setlistToCopy.ArtURL
			setlist.Desc = setlistToCopy.Desc
			setlist.GUID = setlistToCopy.GUID
			setlist.Owner = setlistToCopy.Owner
			setlist.OwnerGUID = setlistToCopy.OwnerGUID
			setlist.PID = setlistToCopy.PID
			setlist.SetlistID = setlistToCopy.SetlistID
			setlist.Title = setlistToCopy.Title
			setlist.Type = setlistToCopy.Type
			setlist.SongIDs = append(setlist.SongIDs, setlistToCopy.SongIDs...)
			setlist.SongNames = append(setlist.SongNames, setlistToCopy.SongNames...)

			resString, _ := marshaler.MarshalResponse(service.Path(), []GetSonglistResponse{setlist})

			jsonStrings = append(jsonStrings, resString)
		}

		// battle setlist
		if setlistToCopy.Type == 1000 || setlistToCopy.Type == 1001 || setlistToCopy.Type == 1002 {
			var battle GetBattleSonglistResponse
			battle.ArtURL = setlistToCopy.ArtURL
			battle.Desc = setlistToCopy.Desc
			battle.GUID = setlistToCopy.GUID
			battle.Owner = setlistToCopy.Owner
			battle.OwnerGUID = setlistToCopy.OwnerGUID
			battle.PID = setlistToCopy.PID
			battle.SetlistID = setlistToCopy.SetlistID
			battle.Title = setlistToCopy.Title
			battle.Type = setlistToCopy.Type
			battle.SongIDs = append(battle.SongIDs, setlistToCopy.SongIDs...)
			battle.SongNames = append(battle.SongNames, setlistToCopy.SongNames...)

			switch setlistToCopy.TimeEndUnits {
			case "seconds":
				battle.SecondsLeft = int(setlistToCopy.Created + int64(setlistToCopy.TimeEndVal) - (time.Now().Unix()))
			case "minutes":
				battle.SecondsLeft = int(setlistToCopy.Created + int64(setlistToCopy.TimeEndVal*60) - (time.Now().Unix()))
			case "hours":
				battle.SecondsLeft = int(setlistToCopy.Created + int64(setlistToCopy.TimeEndVal*3600) - (time.Now().Unix()))
			case "days":
				battle.SecondsLeft = int(setlistToCopy.Created + int64(setlistToCopy.TimeEndVal*86400) - (time.Now().Unix()))
			case "weeks":
				battle.SecondsLeft = int(setlistToCopy.Created + int64(setlistToCopy.TimeEndVal*604800) - (time.Now().Unix()))
			default:
				battle.SecondsLeft = 60 * 60 // default to 1 hour if there is nothing, but this should ideally never happen
			}

			battle.ValidInstr = setlistToCopy.Instrument

			resString, _ := marshaler.MarshalResponse(service.Path(), []GetBattleSonglistResponse{battle})

			jsonStrings = append(jsonStrings, resString)
		}
	}

	resString, _ := marshaler.CombineJSONMethods(jsonStrings)
	return resString, nil
}
