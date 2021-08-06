package models

type Score struct {
	PID          int    `bson:"pid"`
	Name         string `bson:"name"`
	DiffID       int    `bson:"diff_id"`
	Rank         int    `bson:"rank"`
	Score        int    `bson:"score"`
	IsPercentile int    `bson:"is_percentile"`
	InstMask     int    `bson:"inst_mask"`
	NotesPct     int    `bson:"notes_pct"`
	IsFriend     int    `bson:"is_friend"`
	UnnamedBand  int    `bson:"unnamed_band"`
	PGUID        string `bson:"pguid"`
	ORank        int    `bson:"orank"`
}
