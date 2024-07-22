package songlists

import (
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"github.com/jinzhu/copier"
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
	SetlistID   int    `json:"id"`
	PID         int    `json:"pid"`
	Title       string `json:"title"`
	Desc        string `json:"desc"`
	Type        int    `json:"type"`
	Owner       string `json:"owner"`
	OwnerGUID   string `json:"owner_guid"`
	GUID        string `json:"guid"`
	ArtURL      string `json:"art_url"`
	SongID000   int    `json:"s_id000"`
	SongName000 string `json:"s_name000"`
	SongID001   int    `json:"s_id001"`
	SongName001 string `json:"s_name001"`
	SongID002   int    `json:"s_id002"`
	SongName002 string `json:"s_name002"`
	SongID003   int    `json:"s_id003"`
	SongName003 string `json:"s_name003"`
	SongID004   int    `json:"s_id004"`
	SongName004 string `json:"s_name004"`
	SongID005   int    `json:"s_id005"`
	SongName005 string `json:"s_name005"`
	SongID006   int    `json:"s_id006"`
	SongName006 string `json:"s_name006"`
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

	setlistCursor, err := setlistCollection.Find(nil, bson.D{{ "pid", bson.D{{"$ne", req.PID000}}}})

	if err != nil {
		log.Printf("Error getting songlists: %s", err)
	}

	res := []GetSonglistsResponse{}

	for setlistCursor.Next(nil) {
		var setlist GetSonglistsResponse
		var setlistToCopy models.Setlist

		setlistCursor.Decode(&setlistToCopy)

		copier.Copy(&setlist, &setlistToCopy)

		res = append(res, setlist)
	}

	if len(res) == 0 {
		return marshaler.MarshalResponse(service.Path(), []GetSonglistsResponse{{}})
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
