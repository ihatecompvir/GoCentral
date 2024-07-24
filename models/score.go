package models

type Score struct {
	SongID         int `bson:"song_id"`
	OwnerPID       int `bson:"pid"`
	RoleID         int `bson:"role_id"`
	Score          int `bson:"score"`
	NotesPercent   int `bson:"notespct"`
	Stars          int `bson:"stars"`
	DiffID         int `bson:"diff_id"`
	BOI            int `bson:"boi"`
	InstrumentMask int `bson:"instrument_mask"`
}
