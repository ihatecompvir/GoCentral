package ticker

import (
	"rb3server/protocols/jsonproto/marshaler"
)

type TickerInfoRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	RoleID      int    `json:"role_id"` // current instrument?
}

type TickerInfoResponse struct {
	PID              int    `json:"pid"`
	MOTD             string `json:"motd"`
	BattleCount      int    `json:"battle_count"`
	RoleID           int    `json:"role_id"`
	RoleRank         int    `json:"role_rank"`
	RoleIsGlobal     int    `json:"role_is_global"`
	RoleIsPercentile int    `json:"role_is_percentile"`
	BandID           int    `json:"band_id"`
	BandRank         int    `json:"band_rank"`
	BankIsGlobal     int    `json:"band_is_global"`
	BandIsPercentile int    `json:"band_is_percentile"`
}

type TickerInfoService struct {
}

func (service TickerInfoService) Path() string {
	return "ticker/info/get"
}

func (service TickerInfoService) Handle(data string) (string, error) {
	var req TickerInfoRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	// Spoof account linking status, 12345 pid
	res := []TickerInfoResponse{{
		12345,
		"Hello World 2: Electric Boogaloo",
		1,
		3,
		1,
		1,
		0,
		1,
		1,
		1,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
