package battles

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

type GetBattlesClosedRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
}

type GetBattlesClosedResponse struct {
	ID        int      `json:"id"`
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

type GetBattlesClosedService struct {
}

func (service GetBattlesClosedService) Path() string {
	return "battles/closed/get"
}

func (service GetBattlesClosedService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req GetBattlesClosedRequest

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
		log.Printf("Error getting closed battles: %s", err)
	}

	jsonStrings := []string{}

	for setlistCursor.Next(context.TODO()) {
		var setlistToCopy models.Setlist

		setlistCursor.Decode(&setlistToCopy)

		// battle setlist
		if setlistToCopy.Type == 1000 || setlistToCopy.Type == 1001 || setlistToCopy.Type == 1002 {
			var battle GetBattlesClosedResponse
			battle.ArtURL = setlistToCopy.ArtURL
			battle.Desc = setlistToCopy.Desc
			battle.GUID = setlistToCopy.GUID
			battle.Owner = setlistToCopy.Owner
			battle.OwnerGUID = setlistToCopy.OwnerGUID
			battle.PID = setlistToCopy.PID
			battle.Title = setlistToCopy.Title
			battle.Type = setlistToCopy.Type
			battle.SongIDs = append(battle.SongIDs, setlistToCopy.SongIDs...)
			battle.SongNames = append(battle.SongNames, setlistToCopy.SongNames...)

			// get unix time of created, along with time_end_val and time_end_units, and determine if the battle is closed
			createdTime := time.Unix(setlistToCopy.Created, 0)

			switch setlistToCopy.TimeEndUnits {
			case "seconds":
				createdTime = createdTime.Add(time.Second * time.Duration(setlistToCopy.TimeEndVal))
			case "minutes":
				createdTime = createdTime.Add(time.Minute * time.Duration(setlistToCopy.TimeEndVal))
			case "hours":
				createdTime = createdTime.Add(time.Hour * time.Duration(setlistToCopy.TimeEndVal))
			case "days":
				createdTime = createdTime.AddDate(0, 0, setlistToCopy.TimeEndVal)
			}

			// if the battle is closed, add it, otherwise skip
			if time.Now().After(createdTime) {
				resString, _ := marshaler.MarshalResponse(service.Path(), []GetBattlesClosedResponse{battle})

				jsonStrings = append(jsonStrings, resString)
			} else {
				continue
			}
		}
	}

	if len(jsonStrings) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		resString, _ := marshaler.CombineJSONMethods(jsonStrings)
		return resString, nil
	}
}
