package misc

import (
	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type OptionDataRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type OptionDataService struct {
}

func (service OptionDataService) Path() string {
	return "misc/option_data"
}

func (service OptionDataService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	// no practical reason to store players options on the server side since they won't even sync if they go to a different console, so this is only here to reduce errors in the server log
	return "", nil
}
