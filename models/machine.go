package models

type Machine struct {
	ConsoleType   int    `json:"console_type" bson:"console_type"`
	MachineID     int    `json:"machine_id" bson:"machine_id"`
	Status        string `json:"status" bson:"status"`
	WiiFriendCode string `json:"wii_friend_code" bson:"wii_friend_code"`
	StationURL    string `json:"station_url" bson:"station_url"`
	IntStationURL string `json:"int_station_url" bson:"int_station_url"`
}
