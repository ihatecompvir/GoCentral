package models

type Setlist struct {
	Created   int64    `bson:"created"` // unix timestamp of when the setlist was created
	SetlistID int      `bson:"id"`
	PID       int      `bson:"pid"`
	Title     string   `bson:"title"`
	Desc      string   `bson:"desc"`
	Type      int      `bson:"type"`
	Owner     string   `bson:"owner"`
	OwnerGUID string   `bson:"owner_guid"`
	GUID      string   `bson:"guid"`
	ArtURL    string   `bson:"art_url"`
	Shared    string   `bson:"shared"`
	SongIDs   []int    `bson:"s_ids"`
	SongNames []string `bson:"s_names"`

	// battle fields
	TimeEndVal   int    `bson:"time_end_val"`
	TimeEndUnits string `bson:"time_end_units"`
	Flags        int    `bson:"flags"`
	Instrument   int    `bson:"instrument"`
}
