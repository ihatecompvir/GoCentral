package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func CleanupDuplicateScores() {
	scoresCollection := GocentralDatabase.Collection("scores")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", bson.D{
				{"pid", "$pid"},
				{"role_id", "$role_id"},
				{"song_id", "$song_id"},
				{"boi", "$boi"},
				{"diff_id", "$diff_id"},
				{"instrument_mask", "$instrument_mask"},
				{"notespct", "$notespct"},
				{"score", "$score"},
				{"stars", "$stars"},
			}},
			{"count", bson.D{{"$sum", 1}}},
			{"docs", bson.D{{"$push", "$$ROOT"}}},
		}}},
		{{"$match", bson.D{{"count", bson.D{{"$gt", 1}}}}}},
	}

	cursor, err := scoresCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Could not aggregate duplicate scores: ", err)
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Println("Could not decode aggregation results: ", err)
		return
	}

	deletedCount := 0

	for _, result := range results {
		docs := result["docs"].(bson.A)
		for i := 1; i < len(docs); i++ { // skip the first document to keep one
			doc := docs[i].(bson.M)
			_, err := scoresCollection.DeleteOne(ctx, bson.M{"_id": doc["_id"]})
			if err != nil {
				log.Println("Could not delete duplicate score: ", err)
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount != 0 {
		log.Printf("Deleted %d duplicate scores.\n", deletedCount)
	}
}

func PruneOldSessions() {

	gatherings := GocentralDatabase.Collection("gatherings")

	// find any gatherings which haven't had their "updated" field updated in the last hour and delete them
	// technically speaking, someone playing a song longer than one hour could have their gathering deleted, but this is such an extreme and unlikely edge case that it's not worth worrying about
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Calculate the Unix time for 1 hour ago
	cutoff := time.Now().Add(-1 * time.Hour).Unix()

	_, err := gatherings.DeleteMany(ctx, bson.M{"last_updated": bson.M{"$lt": cutoff}})
	if err != nil {
		log.Println("Could not delete old gatherings: ", err)
	}
}

func CleanupInvalidScores() {
	scoresCollection := GocentralDatabase.Collection("scores")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deletedCount := 0

	// Delete any scores where the song ID is 0 (invalid)
	// some customs might have song ID 0, but the scores on such a leaderboard are not going to be right anyway since it will be *every* custom with an invalid song ID
	result, err := scoresCollection.DeleteMany(ctx, bson.M{"song_id": 0})

	deletedCount += int(result.DeletedCount)

	if err != nil {
		log.Println("Could not delete invalid scores: ", err)
	}

	// Delete any scores where the role ID is greater than 10
	result, err = scoresCollection.DeleteMany(ctx, bson.M{"role_id": bson.M{"$gt": 10}})
	if err != nil {
		log.Println("Could not delete invalid scores: ", err)
	}

	deletedCount += int(result.DeletedCount)

	// delete any scores where the score is less than or equal to 0
	result, err = scoresCollection.DeleteMany(ctx, bson.M{"score": bson.M{"$lte": 0}})
	if err != nil {
		log.Println("Could not delete invalid scores: ", err)
	}

	deletedCount += int(result.DeletedCount)

	// delete any scores where the stars are greater than 6
	result, err = scoresCollection.DeleteMany(ctx, bson.M{"stars": bson.M{"$gt": 6}})
	if err != nil {
		log.Println("Could not delete invalid scores: ", err)
	}

	deletedCount += int(result.DeletedCount)

	// delete any scores where the diff ID is greater than 4
	// DIFFICULTIES ARE 1-INDEXED, DON'T CHANGE THIS
	result, err = scoresCollection.DeleteMany(ctx, bson.M{"diff_id": bson.M{"$gt": 4}})
	if err != nil {
		log.Println("Could not delete invalid scores: ", err)
	}

	deletedCount += int(result.DeletedCount)

	// delete any scores where the percentage is greater than 100
	result, err = scoresCollection.DeleteMany(ctx, bson.M{"notespct": bson.M{"$gt": 100}})
	if err != nil {
		log.Println("Could not delete invalid scores: ", err)
	}

	deletedCount += int(result.DeletedCount)

	if deletedCount != 0 {
		log.Printf("Deleted %d invalid scores.\n", deletedCount)
	}

}
