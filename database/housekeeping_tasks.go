package database

import (
	"context"
	"log"
	"rb3server/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func CleanupDuplicateScores() {
	scoresCollection := GocentralDatabase.Collection("scores")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{"$sort", bson.D{{"_id", -1}}}},
		{{"$group", bson.D{
			{"_id", bson.D{
				{"pid", "$pid"}, {"role_id", "$role_id"}, {"song_id", "$song_id"},
				{"boi", "$boi"}, {"diff_id", "$diff_id"}, {"instrument_mask", "$instrument_mask"},
				{"notespct", "$notespct"}, {"score", "$score"}, {"stars", "$stars"},
			}},
			{"ids", bson.D{{"$push", "$_id"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		{{"$match", bson.D{{"count", bson.D{{"$gt", 1}}}}}},
		{{"$project", bson.D{
			{"_id", 0},
			// Take all but the first (newest) id as duplicates to delete.
			{"dups", bson.D{{"$slice",
				bson.A{"$ids", 1, bson.D{{"$subtract", bson.A{bson.D{{"$size", "$ids"}}, 1}}}},
			}}},
		}}},
	}

	cursor, err := scoresCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Could not aggregate duplicate scores:", err)
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Println("Could not decode aggregation results:", err)
		return
	}

	deletedCount := 0
	for _, result := range results {
		dupsAny, ok := result["dups"]
		if !ok || dupsAny == nil {
			continue
		}
		dups, ok := dupsAny.(bson.A)
		if !ok || len(dups) == 0 {
			continue
		}

		res, err := scoresCollection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": dups}})
		if err != nil {
			log.Println("Could not delete duplicate scores:", err)
			continue
		}
		deletedCount += int(res.DeletedCount)
	}

	if deletedCount != 0 {
		log.Printf("Deleted %d duplicate scores.\n", deletedCount)
	}
}

func PruneOldSessions() {

	gatherings := GocentralDatabase.Collection("gatherings")

	// find any gatherings which haven't had their "updated" field updated in the last hour and delete them
	// technically speaking, someone playing a song longer than one hour could have their gathering deleted, but this is such an extreme and unlikely edge case that it's not worth worrying about
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deletedCount := 0

	// safe deletion function
	deleteInvalidScores := func(filter bson.M) {
		result, err := scoresCollection.DeleteMany(ctx, filter)
		if err != nil {
			log.Println("Could not delete invalid scores: ", err)
			return
		}
		if result != nil {
			deletedCount += int(result.DeletedCount)
		}
	}

	// Delete scores based on various conditions
	deleteInvalidScores(bson.M{"song_id": 0})                   // Invalid song ID
	deleteInvalidScores(bson.M{"role_id": bson.M{"$gt": 10}})   // Role ID greater than 10
	deleteInvalidScores(bson.M{"score": bson.M{"$lte": 0}})     // Score less than or equal to 0
	deleteInvalidScores(bson.M{"stars": bson.M{"$gt": 6}})      // Stars greater than 6
	deleteInvalidScores(bson.M{"diff_id": bson.M{"$gt": 4}})    // Difficulty ID greater than 4
	deleteInvalidScores(bson.M{"notespct": bson.M{"$gt": 100}}) // Percentage greater than 100

	if deletedCount != 0 {
		log.Printf("Deleted %d invalid scores.\n", deletedCount)
	}
}

func DeleteExpiredBattles() {
	setlistsCollection := GocentralDatabase.Collection("setlists")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := setlistsCollection.Find(ctx, bson.M{"type": bson.M{"$in": []int{1000, 1001, 1002}}})
	if err != nil {
		log.Println("Could not get setlists for deletion: ", err)
		return
	}
	defer cursor.Close(ctx)

	deletedCount := 0

	for cursor.Next(ctx) {
		var setlist models.Setlist
		cursor.Decode(&setlist)

		isExpired, expiryTime := GetBattleExpiryInfo(setlist.SetlistID)

		if isExpired {
			// allow players 3 days to view the leaderboards of the setlist before it is nuked
			// the game itself should prevent recording scores at this time, but we should add a check for this in the battle score record too
			expiredTime := expiryTime.Add(3 * 24 * time.Hour)

			if time.Now().After(expiredTime) {
				_, err := setlistsCollection.DeleteOne(ctx, bson.M{"setlist_id": setlist.SetlistID})
				if err != nil {
					log.Println("Could not delete expired battle: ", err)
				} else {
					deletedCount++
				}

				// delete all scores associated with this setlist
				scoresCollection := GocentralDatabase.Collection("scores")
				_, err = scoresCollection.DeleteMany(ctx, bson.M{"setlist_id": setlist.SetlistID})

				if err != nil {
					log.Println("Could not delete scores associated with expired battle: ", err)
				}
			}
		}
	}

	if deletedCount != 0 {
		log.Printf("Deleted %d expired battles.\n", deletedCount)
	}

}

func CleanupBannedUserScores() {
	config, err := GetCachedConfig(context.Background())
	if err != nil {
		log.Println("Could not get config for banned user score cleanup:", err)
		return
	}

	scoresCollection := GocentralDatabase.Collection("scores")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	deletedCount := 0

	for _, bannedPlayer := range config.BannedPlayers {
		// Only delete scores for permanently banned users
		if bannedPlayer.ExpiresAt.IsZero() {
			pid := GetPIDForUsername(bannedPlayer.Username)
			if pid == 0 {
				continue
			}

			// Delete all scores for this user
			res, err := scoresCollection.DeleteMany(ctx, bson.M{"pid": pid})
			if err != nil {
				log.Printf("Error deleting scores for banned user %s (PID %d): %v\n", bannedPlayer.Username, pid, err)
			} else if res.DeletedCount > 0 {
				log.Printf("Deleted %d scores for permanently banned user %s (PID %d)\n", res.DeletedCount, bannedPlayer.Username, pid)
				deletedCount += int(res.DeletedCount)
			}
		}
	}

	if deletedCount > 0 {
		log.Printf("CleanupBannedUserScores: Removed a total of %d scores from banned users.\n", deletedCount)
	}
}

func CleanupBannedUserAccomplishments() {
	config, err := GetCachedConfig(context.Background())
	if err != nil {
		log.Println("Could not get config for banned user accomplishment cleanup:", err)
		return
	}

	accomplishmentsCollection := GocentralDatabase.Collection("accomplishments")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Collect PIDs of permanently banned users
	var bannedPIDs []int
	for _, bannedPlayer := range config.BannedPlayers {
		if bannedPlayer.ExpiresAt.IsZero() {
			pid := GetPIDForUsername(bannedPlayer.Username)
			if pid != 0 {
				bannedPIDs = append(bannedPIDs, pid)
			}
		}
	}

	if len(bannedPIDs) == 0 {
		return
	}

	// Get the accomplishments document
	var accomplishments models.Accomplishments
	err = accomplishmentsCollection.FindOne(ctx, bson.M{}).Decode(&accomplishments)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			log.Println("Could not get accomplishments for banned user cleanup:", err)
		}
		return
	}

	// Helper to check if a PID is banned
	isBanned := func(pid int) bool {
		for _, bannedPID := range bannedPIDs {
			if pid == bannedPID {
				return true
			}
		}
		return false
	}

	// Helper to filter out banned PIDs from a slice
	filterEntries := func(entries []models.AccomplishmentScoreEntry) []models.AccomplishmentScoreEntry {
		filtered := make([]models.AccomplishmentScoreEntry, 0, len(entries))
		for _, entry := range entries {
			if !isBanned(entry.PID) {
				filtered = append(filtered, entry)
			}
		}
		return filtered
	}

	// Filter all accomplishment leaderboards
	accomplishments.LBGoalValueCampaignMetascore = filterEntries(accomplishments.LBGoalValueCampaignMetascore)
	accomplishments.LBGoalValueAccTourgoldlocal1 = filterEntries(accomplishments.LBGoalValueAccTourgoldlocal1)
	accomplishments.LBGoalValueAccTourgoldlocal2 = filterEntries(accomplishments.LBGoalValueAccTourgoldlocal2)
	accomplishments.LBGoalValueAccTourgoldregional1 = filterEntries(accomplishments.LBGoalValueAccTourgoldregional1)
	accomplishments.LBGoalValueAccTourgoldregional2 = filterEntries(accomplishments.LBGoalValueAccTourgoldregional2)
	accomplishments.LBGoalValueAccTourgoldcontinental1 = filterEntries(accomplishments.LBGoalValueAccTourgoldcontinental1)
	accomplishments.LBGoalValueAccTourgoldcontinental2 = filterEntries(accomplishments.LBGoalValueAccTourgoldcontinental2)
	accomplishments.LBGoalValueAccTourgoldcontinental3 = filterEntries(accomplishments.LBGoalValueAccTourgoldcontinental3)
	accomplishments.LBGoalValueAccTourgoldglobal1 = filterEntries(accomplishments.LBGoalValueAccTourgoldglobal1)
	accomplishments.LBGoalValueAccTourgoldglobal2 = filterEntries(accomplishments.LBGoalValueAccTourgoldglobal2)
	accomplishments.LBGoalValueAccTourgoldglobal3 = filterEntries(accomplishments.LBGoalValueAccTourgoldglobal3)
	accomplishments.LBGoalValueAccOverdrivemaintain3 = filterEntries(accomplishments.LBGoalValueAccOverdrivemaintain3)
	accomplishments.LBGoalValueAccOverdrivecareer = filterEntries(accomplishments.LBGoalValueAccOverdrivecareer)
	accomplishments.LBGoalValueAccCareersaves = filterEntries(accomplishments.LBGoalValueAccCareersaves)
	accomplishments.LBGoalValueAccMillionpoints = filterEntries(accomplishments.LBGoalValueAccMillionpoints)
	accomplishments.LBGoalValueAccBassstreaklarge = filterEntries(accomplishments.LBGoalValueAccBassstreaklarge)
	accomplishments.LBGoalValueAccHopothreehundredbass = filterEntries(accomplishments.LBGoalValueAccHopothreehundredbass)
	accomplishments.LBGoalValueAccDrumfill170 = filterEntries(accomplishments.LBGoalValueAccDrumfill170)
	accomplishments.LBGoalValueAccDrumstreaklong = filterEntries(accomplishments.LBGoalValueAccDrumstreaklong)
	accomplishments.LBGoalValueAccDeployguitarfour = filterEntries(accomplishments.LBGoalValueAccDeployguitarfour)
	accomplishments.LBGoalValueAccGuitarstreaklarge = filterEntries(accomplishments.LBGoalValueAccGuitarstreaklarge)
	accomplishments.LBGoalValueAccHopoonethousand = filterEntries(accomplishments.LBGoalValueAccHopoonethousand)
	accomplishments.LBGoalValueAccDoubleawesomealot = filterEntries(accomplishments.LBGoalValueAccDoubleawesomealot)
	accomplishments.LBGoalValueAccTripleawesomealot = filterEntries(accomplishments.LBGoalValueAccTripleawesomealot)
	accomplishments.LBGoalValueAccKeystreaklong = filterEntries(accomplishments.LBGoalValueAccKeystreaklong)
	accomplishments.LBGoalValueAccProbassstreakepic = filterEntries(accomplishments.LBGoalValueAccProbassstreakepic)
	accomplishments.LBGoalValueAccProdrumroll3 = filterEntries(accomplishments.LBGoalValueAccProdrumroll3)
	accomplishments.LBGoalValueAccProdrumstreaklong = filterEntries(accomplishments.LBGoalValueAccProdrumstreaklong)
	accomplishments.LBGoalValueAccProguitarstreakepic = filterEntries(accomplishments.LBGoalValueAccProguitarstreakepic)
	accomplishments.LBGoalValueAccProkeystreaklong = filterEntries(accomplishments.LBGoalValueAccProkeystreaklong)
	accomplishments.LBGoalValueAccDeployvocals = filterEntries(accomplishments.LBGoalValueAccDeployvocals)
	accomplishments.LBGoalValueAccDeployvocalsonehundred = filterEntries(accomplishments.LBGoalValueAccDeployvocalsonehundred)

	// Update the document
	update := bson.M{
		"$set": bson.M{
			"lb_goal_value_campaign_metascore":         accomplishments.LBGoalValueCampaignMetascore,
			"lb_goal_value_acc_tourgoldlocal1":         accomplishments.LBGoalValueAccTourgoldlocal1,
			"lb_goal_value_acc_tourgoldlocal2":         accomplishments.LBGoalValueAccTourgoldlocal2,
			"lb_goal_value_acc_tourgoldregional1":      accomplishments.LBGoalValueAccTourgoldregional1,
			"lb_goal_value_acc_tourgoldregional2":      accomplishments.LBGoalValueAccTourgoldregional2,
			"lb_goal_value_acc_tourgoldcontinental1":   accomplishments.LBGoalValueAccTourgoldcontinental1,
			"lb_goal_value_acc_tourgoldcontinental2":   accomplishments.LBGoalValueAccTourgoldcontinental2,
			"lb_goal_value_acc_tourgoldcontinental3":   accomplishments.LBGoalValueAccTourgoldcontinental3,
			"lb_goal_value_acc_tourgoldglobal1":        accomplishments.LBGoalValueAccTourgoldglobal1,
			"lb_goal_value_acc_tourgoldglobal2":        accomplishments.LBGoalValueAccTourgoldglobal2,
			"lb_goal_value_acc_tourgoldglobal3":        accomplishments.LBGoalValueAccTourgoldglobal3,
			"lb_goal_value_acc_overdrivemaintain3":     accomplishments.LBGoalValueAccOverdrivemaintain3,
			"lb_goal_value_acc_overdrivecareer":        accomplishments.LBGoalValueAccOverdrivecareer,
			"lb_goal_value_acc_careersaves":            accomplishments.LBGoalValueAccCareersaves,
			"lb_goal_value_acc_millionpoints":          accomplishments.LBGoalValueAccMillionpoints,
			"lb_goal_value_acc_bassstreaklarge":        accomplishments.LBGoalValueAccBassstreaklarge,
			"lb_goal_value_acc_hopothreehundredbass":   accomplishments.LBGoalValueAccHopothreehundredbass,
			"lb_goal_value_acc_drumfill170":            accomplishments.LBGoalValueAccDrumfill170,
			"lb_goal_value_acc_drumstreaklong":         accomplishments.LBGoalValueAccDrumstreaklong,
			"lb_goal_value_acc_deployguitarfour":       accomplishments.LBGoalValueAccDeployguitarfour,
			"lb_goal_value_acc_guitarstreaklarge":      accomplishments.LBGoalValueAccGuitarstreaklarge,
			"lb_goal_value_acc_hopoonethousand":        accomplishments.LBGoalValueAccHopoonethousand,
			"lb_goal_value_acc_doubleawesomealot":      accomplishments.LBGoalValueAccDoubleawesomealot,
			"lb_goal_value_acc_tripleawesomealot":      accomplishments.LBGoalValueAccTripleawesomealot,
			"lb_goal_value_acc_keystreaklong":          accomplishments.LBGoalValueAccKeystreaklong,
			"lb_goal_value_acc_probassstreakepic":      accomplishments.LBGoalValueAccProbassstreakepic,
			"lb_goal_value_acc_prodrumroll3":           accomplishments.LBGoalValueAccProdrumroll3,
			"lb_goal_value_acc_prodrumstreaklong":      accomplishments.LBGoalValueAccProdrumstreaklong,
			"lb_goal_value_acc_proguitarstreakepic":    accomplishments.LBGoalValueAccProguitarstreakepic,
			"lb_goal_value_acc_prokeystreaklong":       accomplishments.LBGoalValueAccProkeystreaklong,
			"lb_goal_value_acc_deployvocals":           accomplishments.LBGoalValueAccDeployvocals,
			"lb_goal_value_acc_deployvocalsonehundred": accomplishments.LBGoalValueAccDeployvocalsonehundred,
		},
	}

	_, err = accomplishmentsCollection.UpdateOne(ctx, bson.M{}, update)
	if err != nil {
		log.Println("Could not update accomplishments after banned user cleanup:", err)
	}
}

func CleanupInvalidUsers() {
	usersCollection := GocentralDatabase.Collection("users")
	scoresCollection := GocentralDatabase.Collection("scores")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find all users with empty usernames
	cursor, err := usersCollection.Find(ctx, bson.M{"username": ""})
	if err != nil {
		log.Println("Could not find invalid users:", err)
		return
	}
	defer cursor.Close(ctx)

	var invalidUsers []models.User
	if err = cursor.All(ctx, &invalidUsers); err != nil {
		log.Println("Could not decode invalid users:", err)
		return
	}

	deletedUserCount := 0
	deletedScoreCount := 0

	for _, user := range invalidUsers {
		// Delete all scores for this user
		scoresResult, err := scoresCollection.DeleteMany(ctx, bson.M{"pid": user.PID})
		if err != nil {
			log.Printf("Could not delete scores for invalid user PID %d: %v\n", user.PID, err)
			continue
		}
		deletedScoreCount += int(scoresResult.DeletedCount)

		// Delete the user
		userResult, err := usersCollection.DeleteOne(ctx, bson.M{"pid": user.PID})
		if err != nil {
			log.Printf("Could not delete invalid user PID %d: %v\n", user.PID, err)
			continue
		}
		if userResult.DeletedCount > 0 {
			deletedUserCount++
		}
	}

	if deletedUserCount > 0 || deletedScoreCount > 0 {
		log.Printf("Deleted %d invalid users and %d associated scores.\n", deletedUserCount, deletedScoreCount)
	}
}

