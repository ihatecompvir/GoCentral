package config

import (
	"rb3server/protocols/jsonproto/marshaler"
)

type ConfigRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
}

type ConfigResponse struct {
	OutDta  string `json:"out_dta"`
	Version string `json:"version"`
}

type ConfigService struct {
}

func (service ConfigService) Path() string {
	return "config/get"
}

func (service ConfigService) Handle(data string) (string, error) {
	var req ConfigRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	res := []ConfigResponse{{
		"{do {main_hub_panel set_motd \"Hello World\"} {main_hub_panel set_dlcmotd \"Example Text\"} }",
		"3",
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
