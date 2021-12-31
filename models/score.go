package models

type Score struct {
	SongID   int `bson:"song_id"`
	OwnerPID int `bson:"owner_pid"`
	RoleID   int `bson:"role_id"`
	Score    int `bson:"score"`
	CScore   int `bson:"cscore"`
	CCScore  int `bson:"ccscore"`
	Percent  int `bson:"percent"`
	DiffID   int `bson:"diff_id"`
	Slot     int `bson:"slot"`
}
