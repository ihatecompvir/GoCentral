package models

type Setlist struct {
	ID          string `bson:"_id"`
	SetlistID   int    `bson:"setlistid"`
	PID         int    `bson:"pid"`
	Title       string `bson:"title"`
	Desc        string `bson:"desc"`
	Type        int    `bson:"type"`
	Owner       string `bson:"owner"`
	OwnerGUID   string `bson:"owner_guid"`
	GUID        string `bson:"guid"`
	ArtURL      string `bson:"art_url"`
	SongID000   int    `bson:"s_id000"`
	SongName000 string `bson:"s_name000"`
	SongID001   int    `bson:"s_id001"`
	SongName001 string `bson:"s_name001"`
	SongID002   int    `bson:"s_id002"`
	SongName002 string `bson:"s_name002"`
	SongID003   int    `bson:"s_id003"`
	SongName003 string `bson:"s_name003"`
	SongID004   int    `bson:"s_id004"`
	SongName004 string `bson:"s_name004"`
	SongID005   int    `bson:"s_id005"`
	SongName005 string `bson:"s_name005"`
	SongID006   int    `bson:"s_id006"`
	SongName006 string `bson:"s_name006"`
}
