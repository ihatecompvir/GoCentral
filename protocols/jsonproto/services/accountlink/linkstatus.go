package accountlink

import (
	"log"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type AccountLinkRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type AccountLinkResponse struct {
	PID    int `json:"pid"`
	Linked int `json:"linked"`
}

type AccountLinkService struct {
}

func (service AccountLinkService) Path() string {
	return "misc/get_accounts_web_linked_status"
}

func (service AccountLinkService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccountLinkRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	validPIDres, err := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to check link status without a valid server-assigned PID, rejecting call")
		return "", err
	}

	res := []AccountLinkResponse{{
		req.PID,
		1,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
