package battles

import (
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type GetBattlesRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
}

type GetBattlesResponse struct {
	ID          int      `json:"id"`
	PID         int      `json:"pid"`
	Title       string   `json:"title"`
	Desc        string   `json:"desc"`
	Type        int      `json:"type"`
	Owner       string   `json:"owner"`
	OwnerGUID   string   `json:"owner_guid"`
	GUID        string   `json:"guid"`
	ArtURL      string   `json:"art_url"`
	SongID000   []int    `json:"s_idXXX"`
	SongName000 []string `json:"s_nameXXX"`
}

type GetBattlesService struct {
}

func (service GetBattlesService) Path() string {
	return "battles/closed/get"
}

func (service GetBattlesService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req GetBattlesRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting getting battles")
		return "", err
	}

	setlistCollection := database.Collection("battles")

	setlistCursor, err := setlistCollection.Find(nil, bson.D{})

	if err != nil {
		log.Printf("Error getting setlist for battle: %s", err)
	}

	res := []GetBattlesResponse{}

	for setlistCursor.Next(nil) {
		var setlist GetBattlesResponse
		var setlistToCopy models.Setlist

		setlistCursor.Decode(&setlistToCopy)

		copier.Copy(&setlist, &setlistToCopy)

		res = append(res, setlist)
	}

	if len(res) == 0 {
		return marshaler.MarshalResponse(service.Path(), []GetBattlesResponse{{}})
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
