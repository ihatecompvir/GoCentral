package songlists

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"
	"time"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	db "rb3server/database"
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

	res, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID000))

	if !res {
		log.Println("Client is attempting to get songlists without a valid server-assigned PID, rejecting call")
		return "", err
	}

	setlistCollection := database.Collection("setlists")

	setlistCursor, err := setlistCollection.Find(context.TODO(), bson.D{{"shared", "t"}})

	if err != nil {
		log.Printf("Error getting songlists: %s", err)
	}

	jsonStrings := []string{}

	for setlistCursor.Next(context.TODO()) {
		var setlistToCopy models.Setlist

		setlistCursor.Decode(&setlistToCopy)

		// normal setlist
		if setlistToCopy.Type == 1 || setlistToCopy.Type == 2 || setlistToCopy.Type == 0 {

			// always show "Harmonix Recommends" aka server-provided setlists which are intended to be global for all players
			if setlistToCopy.Type != 2 {
				// make sure we only get setlists created by our friends
				isFriendCreated, err := db.IsPIDAFriendOfPID(req.PID000, setlistToCopy.PID)

				if err != nil {
					continue
				}

				if !isFriendCreated {
					continue
				}
			}

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

			// always show server-provided battles
			if setlistToCopy.Type != 1002 {
				// make sure we only get battles created by our friends
				isFriendCreated, err := db.IsPIDAFriendOfPID(req.PID000, setlistToCopy.PID)

				if err != nil {
					continue
				}

				if !isFriendCreated {
					continue
				}
			}

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
