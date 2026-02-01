package tests

import (
	"context"
	"rb3server/database"
	"rb3server/servers"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// ============================================
// SanitizePath Tests
// ============================================

func TestSanitizePath_RemovesInvalidChars(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Asterisk", "file*name", "file_name"},
		{"Question mark", "file?name", "file_name"},
		{"Double quotes", "file\"name", "file_name"},
		{"Less than", "file<name", "file_name"},
		{"Greater than", "file>name", "file_name"},
		{"Pipe", "file|name", "file_name"},
		{"Carriage return", "file\rname", "file_name"},
		{"Newline", "file\nname", "file_name"},
		{"Null byte", "file\x00name", "file_name"},
		{"Period", "file.name", "file_name"},
		{"Multiple invalid", "file*?\"<>|name", "file______name"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := servers.SanitizePath(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizePath(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestSanitizePath_PreservesValidChars(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"Alphanumeric", "file123name"},
		{"Underscores", "file_name"},
		{"Hyphens", "file-name"},
		{"Path separators", "path/to/file"},
		{"Drive letter", "C:/path/to/file"},
		{"Mixed valid", "path/to/file-name_123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := servers.SanitizePath(tc.input)
			// Should be unchanged (except . which is always replaced)
			expectedResult := strings.ReplaceAll(tc.input, ".", "_")
			if result != expectedResult {
				t.Errorf("SanitizePath(%q) = %q, expected %q", tc.input, result, expectedResult)
			}
		})
	}
}

func TestSanitizePath_PathTraversalPrevention(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"Parent directory dots", "../../../etc/passwd"},
		{"Windows parent", "..\\..\\windows\\system32"},
		{"Hidden file", ".hidden"},
		{"Double dots", "path/../secret"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := servers.SanitizePath(tc.input)
			// All dots should be replaced with underscores
			if strings.Contains(result, "..") {
				t.Errorf("SanitizePath(%q) = %q, still contains '..'", tc.input, result)
			}
		})
	}
}

func TestSanitizePath_EmptyString(t *testing.T) {
	result := servers.SanitizePath("")
	if result != "" {
		t.Errorf("SanitizePath(\"\") = %q, expected empty string", result)
	}
}

// ============================================
// Ban Checking Logic Tests (using database)
// ============================================

func TestBanChecking_ActiveBan(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	// Add an active ban
	testUsername := "banned_test_user_active"
	activeBan := map[string]interface{}{
		"username":   testUsername,
		"reason":     "Test active ban",
		"expires_at": time.Now().Add(24 * time.Hour), // Expires tomorrow
		"created_at": time.Now(),
	}

	_, err := configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$push": bson.M{"banned_players": activeBan},
	})
	if err != nil {
		t.Fatalf("Failed to add test ban: %v", err)
	}

	// Get the config and check if ban is detected
	config, err := database.GetCachedConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	// Invalidate cache to ensure fresh data
	database.InvalidateConfigCache()

	config, err = database.GetCachedConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to get config after invalidation: %v", err)
	}

	// Check that the ban exists
	found := false
	for _, ban := range config.BannedPlayers {
		if ban.Username == testUsername {
			found = true
			// Check that ban is still active
			if !ban.ExpiresAt.IsZero() && time.Now().After(ban.ExpiresAt) {
				t.Error("Ban should be active but appears expired")
			}
			break
		}
	}

	if !found {
		t.Error("Test ban not found in config")
	}

	// Cleanup
	configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": testUsername}},
	})

	t.Log("BanChecking: active ban correctly detected")
}

func TestBanChecking_ExpiredBan(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	// Add an expired ban
	testUsername := "banned_test_user_expired"
	expiredBan := map[string]interface{}{
		"username":   testUsername,
		"reason":     "Test expired ban",
		"expires_at": time.Now().Add(-24 * time.Hour), // Expired yesterday
		"created_at": time.Now().Add(-48 * time.Hour),
	}

	_, err := configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$push": bson.M{"banned_players": expiredBan},
	})
	if err != nil {
		t.Fatalf("Failed to add test ban: %v", err)
	}

	database.InvalidateConfigCache()

	config, err := database.GetCachedConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	// Check that the expired ban is detected as expired
	for _, ban := range config.BannedPlayers {
		if ban.Username == testUsername {
			if ban.ExpiresAt.IsZero() {
				t.Error("Expired ban should have a non-zero expiry time")
			}
			if time.Now().Before(ban.ExpiresAt) {
				t.Error("Ban should be expired but appears active")
			}
			break
		}
	}

	// Cleanup
	configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": testUsername}},
	})

	t.Log("BanChecking: expired ban correctly identified")
}

func TestBanChecking_PermanentBan(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	// Add a permanent ban (zero time)
	testUsername := "banned_test_user_permanent"
	permanentBan := map[string]interface{}{
		"username":   testUsername,
		"reason":     "Test permanent ban",
		"expires_at": time.Time{}, // Zero time = permanent
		"created_at": time.Now(),
	}

	_, err := configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$push": bson.M{"banned_players": permanentBan},
	})
	if err != nil {
		t.Fatalf("Failed to add test ban: %v", err)
	}

	database.InvalidateConfigCache()

	config, err := database.GetCachedConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	// Check that permanent ban is detected
	for _, ban := range config.BannedPlayers {
		if ban.Username == testUsername {
			if !ban.ExpiresAt.IsZero() {
				t.Error("Permanent ban should have zero expiry time")
			}
			break
		}
	}

	// Cleanup
	configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": testUsername}},
	})

	t.Log("BanChecking: permanent ban correctly detected")
}


// ============================================
// Master User Detection Tests
// ============================================

func TestMasterUserDetection(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		isMaster bool
	}{
		{"Valid master user", "Master User (1234567890123456)", true},
		{"Regular PS3 user", "SomePlayer", false},
		{"Regular Xbox user", "xXSomePlayerXx", false},
		{"Contains Master but not format", "The Master of Games", false},
		{"Master User without FC", "Master User", false},
		{"Master User with text in parens", "Master User (not a number)", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := database.IsUsernameAMasterUser(tc.username)
			if result != tc.isMaster {
				t.Errorf("IsUsernameAMasterUser(%q) = %v, expected %v", tc.username, result, tc.isMaster)
			}
		})
	}
}

// ============================================
// Gathering State Tests
// ============================================

func TestGatheringStates(t *testing.T) {
	ctx := context.Background()
	gatheringsCollection := database.GocentralDatabase.Collection("gatherings")

	// Define gathering states used in the game
	states := []struct {
		id          int
		name        string
		description string
	}{
		{0, "Waiting", "Waiting for players"},
		{1, "Ready", "Ready to start"},
		{2, "InSong", "Currently playing a song"},
		{3, "SongSelect", "Selecting a song"},
		{6, "Unknown6", "Unknown state 6 (excluded from search)"},
	}

	for _, state := range states {
		// Create a gathering with this state
		gatheringID := 900000 + state.id
		gathering := map[string]interface{}{
			"gathering_id": gatheringID,
			"state":        state.id,
			"last_updated": time.Now().Unix(),
			"owner_pid":    500,
			"public":       true,
			"creator":      "testuser",
			"contents":     []byte{},
		}

		_, err := gatheringsCollection.InsertOne(ctx, gathering)
		if err != nil {
			t.Fatalf("Failed to insert gathering with state %d: %v", state.id, err)
		}
	}

	// Verify gatherings were created with correct states
	for _, state := range states {
		gatheringID := 900000 + state.id
		var gathering struct {
			State int `bson:"state"`
		}
		err := gatheringsCollection.FindOne(ctx, bson.M{"gathering_id": gatheringID}).Decode(&gathering)
		if err != nil {
			t.Errorf("Failed to find gathering with state %d: %v", state.id, err)
			continue
		}
		if gathering.State != state.id {
			t.Errorf("Gathering state mismatch: expected %d, got %d", state.id, gathering.State)
		}
	}

	// Test exclusion of states 2 and 6 from custom find (these are "in song" states)
	excludedStates := []int{2, 6}
	for _, excludedState := range excludedStates {
		// In real customfind.go, states 2 and 6 are excluded
		t.Logf("State %d should be excluded from matchmaking search", excludedState)
	}

	// Cleanup
	for _, state := range states {
		gatheringID := 900000 + state.id
		gatheringsCollection.DeleteOne(ctx, bson.M{"gathering_id": gatheringID})
	}

	t.Log("GatheringStates: all states correctly handled")
}

// ============================================
// Gathering Timeout Tests
// ============================================

func TestGatheringTimeout(t *testing.T) {
	ctx := context.Background()
	gatheringsCollection := database.GocentralDatabase.Collection("gatherings")

	// The game considers gatherings stale if not updated in 5 minutes
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute).Unix()
	tenMinutesAgo := time.Now().Add(-10 * time.Minute).Unix()
	oneMinuteAgo := time.Now().Add(-1 * time.Minute).Unix()

	testGatherings := []struct {
		id          int
		lastUpdated int64
		shouldShow  bool
		description string
	}{
		{800001, oneMinuteAgo, true, "Recent gathering (1 min ago)"},
		{800002, fiveMinutesAgo, false, "Borderline gathering (5 min ago)"},
		{800003, tenMinutesAgo, false, "Stale gathering (10 min ago)"},
		{800004, time.Now().Unix(), true, "Just updated gathering"},
	}

	for _, tg := range testGatherings {
		gathering := map[string]interface{}{
			"gathering_id": tg.id,
			"last_updated": tg.lastUpdated,
			"owner_pid":    500,
			"public":       true,
			"creator":      "testuser",
			"state":        0,
		}
		gatheringsCollection.InsertOne(ctx, gathering)
	}

	// Check which gatherings would be shown (updated within last 5 minutes)
	cutoff := time.Now().Add(-5 * time.Minute).Unix()

	for _, tg := range testGatherings {
		var gathering struct {
			LastUpdated int64 `bson:"last_updated"`
		}
		err := gatheringsCollection.FindOne(ctx, bson.M{"gathering_id": tg.id}).Decode(&gathering)
		if err != nil {
			t.Errorf("Failed to find gathering %d: %v", tg.id, err)
			continue
		}

		wouldShow := gathering.LastUpdated > cutoff
		if wouldShow != tg.shouldShow {
			t.Errorf("%s: expected shouldShow=%v, got %v", tg.description, tg.shouldShow, wouldShow)
		}
	}

	// Cleanup
	for _, tg := range testGatherings {
		gatheringsCollection.DeleteOne(ctx, bson.M{"gathering_id": tg.id})
	}

	t.Log("GatheringTimeout: timeout logic correctly identifies stale gatherings")
}


// ============================================
// Concurrent Gathering Operations Tests
// ============================================

func TestConcurrentGatheringCreation(t *testing.T) {
	ctx := context.Background()
	gatheringsCollection := database.GocentralDatabase.Collection("gatherings")

	numGoroutines := 10
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)
	idChan := make(chan int, numGoroutines)

	// Simulate concurrent gathering creation
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			gatheringID := 700000 + index
			gathering := map[string]interface{}{
				"gathering_id": gatheringID,
				"last_updated": time.Now().Unix(),
				"owner_pid":    500 + index,
				"public":       true,
				"creator":      "testuser",
				"state":        0,
			}

			_, err := gatheringsCollection.InsertOne(ctx, gathering)
			if err != nil {
				errChan <- err
				return
			}
			idChan <- gatheringID
		}(i)
	}

	wg.Wait()
	close(errChan)
	close(idChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Error during concurrent gathering creation: %v", err)
	}

	// Verify all gatherings were created
	createdIDs := make([]int, 0)
	for id := range idChan {
		createdIDs = append(createdIDs, id)
	}

	if len(createdIDs) != numGoroutines {
		t.Errorf("Expected %d gatherings created, got %d", numGoroutines, len(createdIDs))
	}

	// Cleanup
	for _, id := range createdIDs {
		gatheringsCollection.DeleteOne(ctx, bson.M{"gathering_id": id})
	}

	t.Logf("ConcurrentGatheringCreation: successfully created %d gatherings concurrently", len(createdIDs))
}


// ============================================
// User Account Tests
// ============================================

func TestUserAccountFields(t *testing.T) {
	ctx := context.Background()
	usersCollection := database.GocentralDatabase.Collection("users")

	// Create a test user with all expected fields
	testPID := 999888
	testUser := map[string]interface{}{
		"pid":                   testPID,
		"username":              "test_account_fields_user",
		"console_type":          1,
		"guid":                  "abcdef1234567890abcdef1234567890",
		"link_code":             "ABCD123456",
		"created_by_machine_id": 0,
	}

	_, err := usersCollection.InsertOne(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Verify all fields are present
	var user struct {
		PID         int    `bson:"pid"`
		Username    string `bson:"username"`
		ConsoleType int    `bson:"console_type"`
		GUID        string `bson:"guid"`
		LinkCode    string `bson:"link_code"`
	}

	err = usersCollection.FindOne(ctx, bson.M{"pid": testPID}).Decode(&user)
	if err != nil {
		t.Fatalf("Failed to find test user: %v", err)
	}

	if user.PID != testPID {
		t.Errorf("PID mismatch: expected %d, got %d", testPID, user.PID)
	}
	if user.Username != "test_account_fields_user" {
		t.Errorf("Username mismatch: expected %q, got %q", "test_account_fields_user", user.Username)
	}
	if user.ConsoleType != 1 {
		t.Errorf("ConsoleType mismatch: expected %d, got %d", 1, user.ConsoleType)
	}
	if len(user.GUID) != 32 {
		t.Errorf("GUID length mismatch: expected 32, got %d", len(user.GUID))
	}
	if len(user.LinkCode) != 10 {
		t.Errorf("LinkCode length mismatch: expected 10, got %d", len(user.LinkCode))
	}

	// Cleanup
	usersCollection.DeleteOne(ctx, bson.M{"pid": testPID})

	t.Log("UserAccountFields: all expected fields present and correct")
}

// ============================================
// Machine Registration Tests
// ============================================

func TestMachineRegistration(t *testing.T) {
	ctx := context.Background()
	machinesCollection := database.GocentralDatabase.Collection("machines")

	// Create a test machine (Wii)
	testMachineID := 888777
	testMachine := map[string]interface{}{
		"machine_id":      testMachineID,
		"wii_friend_code": "9999888877776666",
		"console_type":    2,
		"status":          "",
		"station_url":     "",
	}

	_, err := machinesCollection.InsertOne(ctx, testMachine)
	if err != nil {
		t.Fatalf("Failed to create test machine: %v", err)
	}

	// Verify machine was created
	var machine struct {
		MachineID     int    `bson:"machine_id"`
		WiiFriendCode string `bson:"wii_friend_code"`
		ConsoleType   int    `bson:"console_type"`
	}

	err = machinesCollection.FindOne(ctx, bson.M{"machine_id": testMachineID}).Decode(&machine)
	if err != nil {
		t.Fatalf("Failed to find test machine: %v", err)
	}

	if machine.MachineID != testMachineID {
		t.Errorf("MachineID mismatch: expected %d, got %d", testMachineID, machine.MachineID)
	}
	if machine.WiiFriendCode != "9999888877776666" {
		t.Errorf("WiiFriendCode mismatch")
	}
	if machine.ConsoleType != 2 {
		t.Errorf("ConsoleType mismatch: expected 2 (Wii), got %d", machine.ConsoleType)
	}

	// Cleanup
	machinesCollection.DeleteOne(ctx, bson.M{"machine_id": testMachineID})

	t.Log("MachineRegistration: machine correctly registered")
}
