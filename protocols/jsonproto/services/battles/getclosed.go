package battles

import (
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
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

type GetBattlesClosedService struct {
}

func (service GetBattlesClosedService) Path() string {
	return "battles/closed/get"
}

func (service GetBattlesClosedService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	// just give an empty response so all battles will appear as open
	return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
}
