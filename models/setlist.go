package models

type Setlist struct {
	SetlistID int      `bson:"id"`
	PID       int      `bson:"pid"`
	Title     string   `bson:"title"`
	Desc      string   `bson:"desc"`
	Type      int      `bson:"type"`
	Owner     string   `bson:"owner"`
	OwnerGUID string   `bson:"owner_guid"`
	GUID      string   `bson:"guid"`
	ArtURL    string   `bson:"art_url"`
	SongIDs   []int    `bson:"s_ids"`
	SongNames []string `bson:"s_names"`
}
