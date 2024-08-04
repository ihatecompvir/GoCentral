package music_library

import (
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type SortAndFiltersRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	Filters     string `json:"filters"`
	Sort        string `json:"sort"`
	Mode        string `json:"mode"`
}

type SortAndFiltersResponse struct {
	Success int `json:"success"`
}

type SortAndFiltersService struct {
}

func (service SortAndFiltersService) Path() string {
	return "music_library/sort_and_filters"
}

func (service SortAndFiltersService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	// no practical reason to log which filters and sorting options players are using. this is only here to reduce errors in the server log
	return marshaler.MarshalResponse(service.Path(), SortAndFiltersResponse{1})
}
