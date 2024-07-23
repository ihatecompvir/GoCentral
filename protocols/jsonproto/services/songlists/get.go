package songlists

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

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

type GetSonglistsResponse struct {
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

	setlistCursor, err := setlistCollection.Find(nil, bson.D{})

	if err != nil {
		log.Printf("Error getting songlists: %s", err)
	}

	res := []GetSonglistsResponse{}

	for setlistCursor.Next(context.TODO()) {
		var setlist GetSonglistsResponse
		var setlistToCopy models.Setlist

		setlistCursor.Decode(&setlistToCopy)

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

		res = append(res, setlist)
	}

	if len(res) == 0 {
		return marshaler.MarshalResponse(service.Path(), []GetSonglistsResponse{{}})
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
