package models

type Score struct {
	SongID         int `bson:"song_id"`
	OwnerPID       int `bson:"pid"`
	RoleID         int `bson:"role_id"`
	Score          int `bson:"score"`
	NotesPercent   int `bson:"notespct"`
	DiffID         int `bson:"diffid"`
	BOI            int `bson:"boi"`
	InstrumentMask int `bson:"instrument_mask"`
}
