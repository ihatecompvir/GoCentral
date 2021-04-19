package setlistcreation

import (
	"rb3server/protocols/jsonproto/marshaler"
)

type SetlistCreationRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
}

type SetlistCreationResponse struct {
	PID     int `json:"pid"`
	Creator int `json:"creator"`
}

type SetlistCreationService struct {
}

func (service SetlistCreationService) Path() string {
	return "misc/get_accounts_setlist_creation_status"
}

func (service SetlistCreationService) Handle(data string) (string, error) {
	var req SetlistCreationRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	res := []SetlistCreationResponse{{
		12345,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
