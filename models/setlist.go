package models

// represents a battle score entry in the database
type BattleScoreEntry struct {
	PID   int `bson:"pid"`
	Score int `bson:"score"`
}

// represents a Setlist in the database
// this is used for both setlists and battles, battles are really just setlists with a type of 1000/1001/1002
type Setlist struct {
	Created   int64    `bson:"created"` // unix timestamp of when the setlist was created
	SetlistID int      `bson:"setlist_id"`
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
