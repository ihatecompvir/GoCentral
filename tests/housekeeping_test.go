package tests

import (
	"context"
	"rb3server/database"
	"testing"
	"time"

	"rb3server/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Helper to insert a test score
func insertTestScore(t *testing.T, score map[string]interface{}) primitive.ObjectID {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	result, err := scoresCollection.InsertOne(ctx, score)
	if err != nil {
		t.Fatalf("Failed to insert test score: %v", err)
	}
	return result.InsertedID.(primitive.ObjectID)
}

// Helper to count scores matching a filter
func countScores(t *testing.T, filter bson.M) int64 {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	count, err := scoresCollection.CountDocuments(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to count scores: %v", err)
	}
	return count
}

// Helper to delete test scores
func deleteTestScores(t *testing.T, ids []primitive.ObjectID) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	_, err := scoresCollection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		t.Logf("Warning: failed to cleanup test scores: %v", err)
	}
}

// Tests that CleanupDuplicateScores removes duplicate score entries
func TestCleanupDuplicateScores(t *testing.T) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Create a unique song_id for this test to avoid conflicts
	testSongID := 999999

	// Insert duplicate scores (same pid, role_id, song_id, etc.)
	duplicateScore := map[string]interface{}{
		"pid":             500,
		"role_id":         1,
		"song_id":         testSongID,
		"boi":             0,
		"diff_id":         2,
		"instrument_mask": 1,
		"notespct":        95,
		"score":           50000,
		"stars":           5,
	}

	var insertedIDs []primitive.ObjectID

	// Insert 3 identical scores
	for i := 0; i < 3; i++ {
		id := insertTestScore(t, duplicateScore)
		insertedIDs = append(insertedIDs, id)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Verify we have 3 scores
	initialCount := countScores(t, bson.M{"song_id": testSongID})
	if initialCount != 3 {
		t.Fatalf("Expected 3 duplicate scores, got %d", initialCount)
	}

	// Run the cleanup
	database.CleanupDuplicateScores()

	// Verify only 1 score remains (the newest one)
	finalCount := countScores(t, bson.M{"song_id": testSongID})
	if finalCount != 1 {
		t.Errorf("Expected 1 score after cleanup, got %d", finalCount)
	}

	// Cleanup: remove the remaining test score
	_, err := scoresCollection.DeleteMany(ctx, bson.M{"song_id": testSongID})
	if err != nil {
		t.Logf("Warning: failed to cleanup test scores: %v", err)
	}

	t.Logf("CleanupDuplicateScores: reduced %d duplicates to %d", initialCount, finalCount)
}

// Tests that CleanupDuplicateScores doesn't remove non-duplicate scores
func TestCleanupDuplicateScores_NoDuplicates(t *testing.T) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	testSongIDBase := 888880

	// Insert unique scores (different song_ids)
	var insertedIDs []primitive.ObjectID
	for i := 0; i < 3; i++ {
		score := map[string]interface{}{
			"pid":             500,
			"role_id":         1,
			"song_id":         testSongIDBase + i, // Different song_id for each
			"boi":             0,
			"diff_id":         2,
			"instrument_mask": 1,
			"notespct":        95,
			"score":           50000,
			"stars":           5,
		}
		id := insertTestScore(t, score)
		insertedIDs = append(insertedIDs, id)
	}

	// Count before cleanup
	initialCount := countScores(t, bson.M{"song_id": bson.M{"$gte": testSongIDBase, "$lt": testSongIDBase + 10}})

	// Run the cleanup
	database.CleanupDuplicateScores()

	// Count after cleanup - should be the same
	finalCount := countScores(t, bson.M{"song_id": bson.M{"$gte": testSongIDBase, "$lt": testSongIDBase + 10}})

	if finalCount != initialCount {
		t.Errorf("Expected %d unique scores to remain, got %d", initialCount, finalCount)
	}

	// Cleanup
	_, err := scoresCollection.DeleteMany(ctx, bson.M{"song_id": bson.M{"$gte": testSongIDBase, "$lt": testSongIDBase + 10}})
	if err != nil {
		t.Logf("Warning: failed to cleanup test scores: %v", err)
	}

	t.Logf("CleanupDuplicateScores: correctly preserved %d unique scores", finalCount)
}

// Tests that PruneOldSessions removes stale gatherings
func TestPruneOldSessions(t *testing.T) {
	ctx := context.Background()
	gatheringsCollection := database.GocentralDatabase.Collection("gatherings")

	// Insert an old gathering (last updated 2 hours ago)
	oldGatheringID := 999999
	oldGathering := map[string]interface{}{
		"gathering_id": oldGatheringID,
		"last_updated": time.Now().Add(-2 * time.Hour).Unix(), // 2 hours ago
		"owner_pid":    500,
	}

	_, err := gatheringsCollection.InsertOne(ctx, oldGathering)
	if err != nil {
		t.Fatalf("Failed to insert old gathering: %v", err)
	}

	// Insert a recent gathering (last updated 30 minutes ago)
	recentGatheringID := 999998
	recentGathering := map[string]interface{}{
		"gathering_id": recentGatheringID,
		"last_updated": time.Now().Add(-30 * time.Minute).Unix(), // 30 minutes ago
		"owner_pid":    501,
	}

	_, err = gatheringsCollection.InsertOne(ctx, recentGathering)
	if err != nil {
		t.Fatalf("Failed to insert recent gathering: %v", err)
	}

	// Run the prune
	database.PruneOldSessions()

	// Verify old gathering was deleted
	count, err := gatheringsCollection.CountDocuments(ctx, bson.M{"gathering_id": oldGatheringID})
	if err != nil {
		t.Fatalf("Failed to count old gathering: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected old gathering to be deleted, but it still exists")
	}

	// Verify recent gathering still exists
	count, err = gatheringsCollection.CountDocuments(ctx, bson.M{"gathering_id": recentGatheringID})
	if err != nil {
		t.Fatalf("Failed to count recent gathering: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected recent gathering to still exist, but it was deleted")
	}

	// Cleanup
	gatheringsCollection.DeleteMany(ctx, bson.M{"gathering_id": bson.M{"$in": []int{oldGatheringID, recentGatheringID}}})

	t.Log("PruneOldSessions: correctly removed old sessions and preserved recent ones")
}

// Tests that CleanupInvalidScores removes scores with invalid data
func TestCleanupInvalidScores(t *testing.T) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Use unique PIDs for test isolation
	testPID := 777777

	// Insert various invalid scores
	invalidScores := []map[string]interface{}{
		{"pid": testPID, "song_id": 0, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 95},    // Invalid: song_id = 0
		{"pid": testPID, "song_id": 100, "role_id": 15, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 95}, // Invalid: role_id > 10
		{"pid": testPID, "song_id": 100, "role_id": 1, "score": 0, "stars": 5, "diff_id": 2, "notespct": 95},     // Invalid: score <= 0
		{"pid": testPID, "song_id": 100, "role_id": 1, "score": -100, "stars": 5, "diff_id": 2, "notespct": 95},  // Invalid: score < 0
		{"pid": testPID, "song_id": 100, "role_id": 1, "score": 1000, "stars": 7, "diff_id": 2, "notespct": 95},  // Invalid: stars > 6
		{"pid": testPID, "song_id": 100, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 5, "notespct": 95},  // Invalid: diff_id > 4
		{"pid": testPID, "song_id": 100, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 105}, // Invalid: notespct > 100
		{"pid": testPID, "song_id": 100, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 95},  // Valid score
	}

	var insertedIDs []primitive.ObjectID
	for _, score := range invalidScores {
		result, err := scoresCollection.InsertOne(ctx, score)
		if err != nil {
			t.Fatalf("Failed to insert test score: %v", err)
		}
		insertedIDs = append(insertedIDs, result.InsertedID.(primitive.ObjectID))
	}

	// Count before cleanup
	initialCount, _ := scoresCollection.CountDocuments(ctx, bson.M{"pid": testPID})
	t.Logf("Inserted %d test scores (7 invalid, 1 valid)", initialCount)

	// Run the cleanup
	database.CleanupInvalidScores()

	// Count after cleanup - should only have 1 valid score left
	finalCount, _ := scoresCollection.CountDocuments(ctx, bson.M{"pid": testPID})

	if finalCount != 1 {
		t.Errorf("Expected 1 valid score to remain, got %d", finalCount)
	}

	// Cleanup remaining test data
	scoresCollection.DeleteMany(ctx, bson.M{"pid": testPID})

	t.Logf("CleanupInvalidScores: removed %d invalid scores, kept %d valid", initialCount-finalCount, finalCount)
}

// Tests individual invalid score conditions
func TestCleanupInvalidScores_IndividualConditions(t *testing.T) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	testCases := []struct {
		name        string
		score       map[string]interface{}
		shouldExist bool
	}{
		{
			name:        "song_id = 0 should be deleted",
			score:       map[string]interface{}{"pid": 666661, "song_id": 0, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 95},
			shouldExist: false,
		},
		{
			name:        "role_id = 11 should be deleted",
			score:       map[string]interface{}{"pid": 666662, "song_id": 100, "role_id": 11, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 95},
			shouldExist: false,
		},
		{
			name:        "score = 0 should be deleted",
			score:       map[string]interface{}{"pid": 666663, "song_id": 100, "role_id": 1, "score": 0, "stars": 5, "diff_id": 2, "notespct": 95},
			shouldExist: false,
		},
		{
			name:        "stars = 7 should be deleted",
			score:       map[string]interface{}{"pid": 666664, "song_id": 100, "role_id": 1, "score": 1000, "stars": 7, "diff_id": 2, "notespct": 95},
			shouldExist: false,
		},
		{
			name:        "diff_id = 5 should be deleted",
			score:       map[string]interface{}{"pid": 666665, "song_id": 100, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 5, "notespct": 95},
			shouldExist: false,
		},
		{
			name:        "notespct = 101 should be deleted",
			score:       map[string]interface{}{"pid": 666666, "song_id": 100, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 101},
			shouldExist: false,
		},
		{
			name:        "valid score should remain",
			score:       map[string]interface{}{"pid": 666667, "song_id": 100, "role_id": 1, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 95},
			shouldExist: true,
		},
		{
			name:        "role_id = 10 (band) should remain",
			score:       map[string]interface{}{"pid": 666668, "song_id": 100, "role_id": 10, "score": 1000, "stars": 5, "diff_id": 2, "notespct": 95},
			shouldExist: true,
		},
		{
			name:        "stars = 6 (gold stars) should remain",
			score:       map[string]interface{}{"pid": 666669, "song_id": 100, "role_id": 1, "score": 1000, "stars": 6, "diff_id": 2, "notespct": 100},
			shouldExist: true,
		},
	}

	// Insert all test scores
	for _, tc := range testCases {
		_, err := scoresCollection.InsertOne(ctx, tc.score)
		if err != nil {
			t.Fatalf("Failed to insert score for %s: %v", tc.name, err)
		}
	}

	// Run cleanup
	database.CleanupInvalidScores()

	// Verify each case
	for _, tc := range testCases {
		pid := tc.score["pid"].(int)
		count, err := scoresCollection.CountDocuments(ctx, bson.M{"pid": pid})
		if err != nil {
			t.Errorf("Failed to check score for %s: %v", tc.name, err)
			continue
		}

		exists := count > 0
		if exists != tc.shouldExist {
			if tc.shouldExist {
				t.Errorf("%s: expected score to remain but it was deleted", tc.name)
			} else {
				t.Errorf("%s: expected score to be deleted but it remains", tc.name)
			}
		}

		// Cleanup this test score
		scoresCollection.DeleteMany(ctx, bson.M{"pid": pid})
	}
}

// Tests that DeleteExpiredBattles removes old battles
func TestDeleteExpiredBattles(t *testing.T) {
	ctx := context.Background()
	setlistsCollection := database.GocentralDatabase.Collection("setlists")
	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Create an expired battle (created 5 days ago, expired after 1 hour)
	// This should be deleted since it's past the 3-day grace period
	expiredBattleID := 888888
	expiredBattle := map[string]interface{}{
		"setlist_id":     expiredBattleID,
		"type":           1002, // Harmonix battle type
		"title":          "Expired Test Battle",
		"created":        time.Now().Add(-5 * 24 * time.Hour).Unix(), // 5 days ago
		"time_end_val":   1,
		"time_end_units": "hours",
		"owner_pid":      500,
	}

	_, err := setlistsCollection.InsertOne(ctx, expiredBattle)
	if err != nil {
		t.Fatalf("Failed to insert expired battle: %v", err)
	}

	// Insert some scores for this battle
	for i := 0; i < 3; i++ {
		score := map[string]interface{}{
			"pid":        500 + i,
			"battle_id":  expiredBattleID,
			"setlist_id": expiredBattleID,
			"score":      10000 * (i + 1),
			"song_id":    100,
			"role_id":    1,
			"stars":      5,
			"diff_id":    2,
			"notespct":   95,
		}
		_, err := scoresCollection.InsertOne(ctx, score)
		if err != nil {
			t.Fatalf("Failed to insert battle score: %v", err)
		}
	}

	// Create a recently expired battle (expired 1 day ago, still in grace period)
	recentlyExpiredBattleID := 888889
	recentlyExpiredBattle := map[string]interface{}{
		"setlist_id":     recentlyExpiredBattleID,
		"type":           1002,
		"title":          "Recently Expired Battle",
		"created":        time.Now().Add(-2 * 24 * time.Hour).Unix(), // 2 days ago
		"time_end_val":   1,
		"time_end_units": "hours",
		"owner_pid":      500,
	}

	_, err = setlistsCollection.InsertOne(ctx, recentlyExpiredBattle)
	if err != nil {
		t.Fatalf("Failed to insert recently expired battle: %v", err)
	}

	// Create an active battle (not expired)
	activeBattleID := 888890
	activeBattle := map[string]interface{}{
		"setlist_id":     activeBattleID,
		"type":           1002,
		"title":          "Active Test Battle",
		"created":        time.Now().Unix(), // Just created
		"time_end_val":   7,
		"time_end_units": "days",
		"owner_pid":      500,
	}

	_, err = setlistsCollection.InsertOne(ctx, activeBattle)
	if err != nil {
		t.Fatalf("Failed to insert active battle: %v", err)
	}

	// Run the cleanup
	database.DeleteExpiredBattles()

	// Verify expired battle was deleted
	count, _ := setlistsCollection.CountDocuments(ctx, bson.M{"setlist_id": expiredBattleID})
	if count != 0 {
		t.Errorf("Expected expired battle to be deleted, but it still exists")
	}

	// Verify expired battle's scores were deleted
	scoreCount, _ := scoresCollection.CountDocuments(ctx, bson.M{"setlist_id": expiredBattleID})
	if scoreCount != 0 {
		t.Errorf("Expected expired battle scores to be deleted, but %d remain", scoreCount)
	}

	// Verify recently expired battle still exists (in grace period)
	count, _ = setlistsCollection.CountDocuments(ctx, bson.M{"setlist_id": recentlyExpiredBattleID})
	if count != 1 {
		t.Errorf("Expected recently expired battle to still exist (grace period), but it was deleted")
	}

	// Verify active battle still exists
	count, _ = setlistsCollection.CountDocuments(ctx, bson.M{"setlist_id": activeBattleID})
	if count != 1 {
		t.Errorf("Expected active battle to still exist, but it was deleted")
	}

	// Cleanup test data
	setlistsCollection.DeleteMany(ctx, bson.M{"setlist_id": bson.M{"$in": []int{expiredBattleID, recentlyExpiredBattleID, activeBattleID}}})
	scoresCollection.DeleteMany(ctx, bson.M{"setlist_id": expiredBattleID})

	t.Log("DeleteExpiredBattles: correctly handled expired, grace-period, and active battles")
}

// Tests that DeleteExpiredBattles handles different battle types correctly
func TestDeleteExpiredBattles_BattleTypes(t *testing.T) {
	ctx := context.Background()
	setlistsCollection := database.GocentralDatabase.Collection("setlists")

	// Type 1000, 1001, 1002 are battle types that should be checked
	// Other types should be ignored
	testBattles := []struct {
		setlistID   int
		battleType  int
		shouldCheck bool
	}{
		{777770, 1000, true}, // Should be checked
		{777771, 1001, true}, // Should be checked
		{777772, 1002, true}, // Should be checked (Harmonix)
		{777773, 0, false},   // Regular setlist, should be ignored
		{777774, 100, false}, // Other type, should be ignored
	}

	for _, tb := range testBattles {
		battle := map[string]interface{}{
			"setlist_id":     tb.setlistID,
			"type":           tb.battleType,
			"title":          "Test Battle",
			"created":        time.Now().Add(-10 * 24 * time.Hour).Unix(), // 10 days ago (expired)
			"time_end_val":   1,
			"time_end_units": "hours",
			"owner_pid":      500,
		}
		_, err := setlistsCollection.InsertOne(ctx, battle)
		if err != nil {
			t.Fatalf("Failed to insert battle type %d: %v", tb.battleType, err)
		}
	}

	// Run cleanup
	database.DeleteExpiredBattles()

	// Verify battle types
	for _, tb := range testBattles {
		count, _ := setlistsCollection.CountDocuments(ctx, bson.M{"setlist_id": tb.setlistID})

		if tb.shouldCheck {
			// Battle types 1000, 1001, 1002 should be deleted (they're expired)
			if count != 0 {
				t.Errorf("Battle type %d (ID %d) should have been deleted but still exists", tb.battleType, tb.setlistID)
			}
		} else {
			// Other types should still exist
			if count != 1 {
				t.Errorf("Battle type %d (ID %d) should NOT have been deleted but was removed", tb.battleType, tb.setlistID)
			}
		}

		// Cleanup
		setlistsCollection.DeleteMany(ctx, bson.M{"setlist_id": tb.setlistID})
	}
}

func TestCleanupBannedUserScores(t *testing.T) {
	// Setup
	configCollection := database.GocentralDatabase.Collection("config")
	scoresCollection := database.GocentralDatabase.Collection("scores")

	bannedUser := "BannedUserTest"
	bannedPID := 99999

	usersCollection := database.GocentralDatabase.Collection("users")
	usersCollection.InsertOne(context.TODO(), bson.M{"pid": bannedPID, "username": bannedUser})
	defer usersCollection.DeleteOne(context.TODO(), bson.M{"pid": bannedPID})

	// Add to config ban list (Permanent ban)
	_, err := configCollection.UpdateOne(context.TODO(), bson.M{}, bson.D{
		{"$push", bson.D{
			{"banned_players", models.BannedPlayer{
				Username:  bannedUser,
				Reason:    "Test Ban",
				ExpiresAt: time.Time{}, // Zero time = permanent
				CreatedAt: time.Now(),
			}},
		}},
	})
	if err != nil {
		t.Fatalf("Failed to add ban: %v", err)
	}
	// Ensure we clean up the ban after test
	defer configCollection.UpdateOne(context.TODO(), bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": bannedUser}},
	})

	insertScoreForPID(t, bannedPID)
	insertScoreForPID(t, bannedPID)

	normalUser := "NormalUserTest"
	normalPID := 88888
	usersCollection.InsertOne(context.TODO(), bson.M{"pid": normalPID, "username": normalUser})
	defer usersCollection.DeleteOne(context.TODO(), bson.M{"pid": normalPID})
	insertScoreForPID(t, normalPID)

	// Force cache invalidation to ensure we pick up the new ban
	database.InvalidateConfigCache()

	database.CleanupBannedUserScores()

	// Check banned user scores are gone
	count, err := scoresCollection.CountDocuments(context.TODO(), bson.M{"pid": bannedPID})
	if err != nil {
		t.Fatalf("Failed to count scores: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 scores for banned user, got %d", count)
	}

	// Check normal user scores remain
	count, err = scoresCollection.CountDocuments(context.TODO(), bson.M{"pid": normalPID})
	if err != nil {
		t.Fatalf("Failed to count scores: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 score for normal user, got %d", count)
	}
}

func insertScoreForPID(t *testing.T, pid int) {
	scoresCollection := database.GocentralDatabase.Collection("scores")
	_, err := scoresCollection.InsertOne(context.TODO(), models.Score{
		OwnerPID: pid,
		Score:    12345,
		SongID:   1,
	})
	if err != nil {
		t.Fatalf("Failed to insert score: %v", err)
	}
}

// Tests that CleanupBannedUserScores works with case-insensitive username matching
func TestCleanupBannedUserScores_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")
	scoresCollection := database.GocentralDatabase.Collection("scores")
	usersCollection := database.GocentralDatabase.Collection("users")

	// Create a user with mixed case username
	testPID := 77777
	actualUsername := "CaseSensitivePlayer" // The actual username in the database

	_, err := usersCollection.InsertOne(ctx, bson.M{
		"pid":      testPID,
		"username": actualUsername,
	})
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}
	defer usersCollection.DeleteOne(ctx, bson.M{"pid": testPID})

	// Add ban with DIFFERENT case than the actual username
	bannedUsername := "CASESENSITIVEPLAYER" // All uppercase

	_, err = configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$push": bson.M{"banned_players": models.BannedPlayer{
			Username:  bannedUsername, // Different case!
			Reason:    "Test case-insensitive score cleanup",
			ExpiresAt: time.Time{}, // Permanent
			CreatedAt: time.Now(),
		}},
	})
	if err != nil {
		t.Fatalf("Failed to add ban: %v", err)
	}
	defer configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": bannedUsername}},
	})

	// Insert scores for the banned user
	insertScoreForPID(t, testPID)
	insertScoreForPID(t, testPID)
	insertScoreForPID(t, testPID)

	// Verify scores exist before cleanup
	initialCount, _ := scoresCollection.CountDocuments(ctx, bson.M{"pid": testPID})
	if initialCount != 3 {
		t.Fatalf("Expected 3 scores before cleanup, got %d", initialCount)
	}

	// Invalidate cache to pick up the new ban
	database.InvalidateConfigCache()

	// Run the cleanup
	database.CleanupBannedUserScores()

	// Verify scores were deleted even though the case didn't match
	finalCount, _ := scoresCollection.CountDocuments(ctx, bson.M{"pid": testPID})
	if finalCount != 0 {
		t.Errorf("Expected 0 scores after cleanup (case-insensitive match), got %d", finalCount)
		// Clean up any remaining scores
		scoresCollection.DeleteMany(ctx, bson.M{"pid": testPID})
	} else {
		t.Logf("Successfully deleted scores for user %q when ban list had %q", actualUsername, bannedUsername)
	}
}

// Tests various case combinations for banned user score cleanup
func TestCleanupBannedUserScores_CaseVariations(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")
	scoresCollection := database.GocentralDatabase.Collection("scores")
	usersCollection := database.GocentralDatabase.Collection("users")

	testCases := []struct {
		name           string
		actualUsername string
		bannedUsername string
		pid            int
	}{
		{"lowercase ban, uppercase user", "LOUDUSER", "louduser", 66661},
		{"uppercase ban, lowercase user", "quietuser", "QUIETUSER", 66662},
		{"mixed ban, different mixed user", "MiXeDuSeR", "mIxEdUsEr", 66663},
		{"exact match", "ExactMatch", "ExactMatch", 66664},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create user
			_, err := usersCollection.InsertOne(ctx, bson.M{
				"pid":      tc.pid,
				"username": tc.actualUsername,
			})
			if err != nil {
				t.Fatalf("Failed to insert test user: %v", err)
			}
			defer usersCollection.DeleteOne(ctx, bson.M{"pid": tc.pid})

			// Add ban
			_, err = configCollection.UpdateOne(ctx, bson.M{}, bson.M{
				"$push": bson.M{"banned_players": models.BannedPlayer{
					Username:  tc.bannedUsername,
					Reason:    "Test case variation",
					ExpiresAt: time.Time{},
					CreatedAt: time.Now(),
				}},
			})
			if err != nil {
				t.Fatalf("Failed to add ban: %v", err)
			}
			defer configCollection.UpdateOne(ctx, bson.M{}, bson.M{
				"$pull": bson.M{"banned_players": bson.M{"username": tc.bannedUsername}},
			})

			// Insert score
			insertScoreForPID(t, tc.pid)

			// Invalidate cache
			database.InvalidateConfigCache()

			// Run cleanup
			database.CleanupBannedUserScores()

			// Check score was deleted
			count, _ := scoresCollection.CountDocuments(ctx, bson.M{"pid": tc.pid})
			if count != 0 {
				t.Errorf("Expected scores to be deleted for user %q (ban: %q), but %d remain",
					tc.actualUsername, tc.bannedUsername, count)
				scoresCollection.DeleteMany(ctx, bson.M{"pid": tc.pid})
			} else {
				t.Logf("Scores deleted: user=%q, ban=%q", tc.actualUsername, tc.bannedUsername)
			}
		})
	}
}
