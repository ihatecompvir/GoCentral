package tests

import (
	"context"
	"log"
	"os"
	"rb3server/database"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error

	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	database.GocentralDatabase = client.Database("gocentral_test")

	// create some mock data for testing
	usersCollection := database.GocentralDatabase.Collection("users")
	_, err = usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":          500,
		"username":     "testuser",
		"console_type": 1,
		"friends":      []int{501, 502},
		"groups":       []string{"admin"},
	})

	_, err = usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":                   501,
		"username":              "testuser2",
		"console_type":          2,
		"created_by_machine_id": 1000000000,
		"friends":               []int{500, 502},
	})

	_, err = usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":                   502,
		"username":              "testuser3",
		"console_type":          2,
		"created_by_machine_id": 1000000000,
		"friends":               []int{500, 501},
	})

	_, err = usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":                   999,
		"username":              "Master User (1234567891234567)",
		"console_type":          2,
		"created_by_machine_id": 1000000000,
	})

	// broken/invalid users
	// to test error handling and how the code will handle these
	_, err = usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":      600,
		"username": "user_no_console",
	})
	if err != nil {
		log.Fatalf("Failed to insert test user: %v", err)
	}

	_, err = usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":          601,
		"username":     "user_bad_console",
		"console_type": 99, // an invalid console type
	})
	if err != nil {
		log.Fatalf("Failed to insert test user: %v", err)
	}

	_, err = usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":                   602,
		"username":              "Master User (0000000000000000)", // a master user with a non-existent friend code
		"console_type":          2,
		"created_by_machine_id": 2000000000,
	})
	if err != nil {
		log.Fatalf("Failed to insert test user: %v", err)
	}

	// insert a test machine
	machinesCollection := database.GocentralDatabase.Collection("machines")
	_, err = machinesCollection.InsertOne(ctx, map[string]interface{}{
		"wii_friend_code": "1234567891234567",
		"console_type":    2,
		"machine_id":      1000000000,
		"status":          ":501:Online:502:Offline:",
		"station_url":     "prudp:/address=192.168.1.69;port=9103;PID=501;sid=15;type=3;RVCID=45",
	})

	// insert a MOTD
	motdCollection := database.GocentralDatabase.Collection("motd")
	_, err = motdCollection.InsertOne(ctx, map[string]interface{}{
		"version": 3,
		"dta":     "{}",
	})

	// insert a test band
	bandsCollection := database.GocentralDatabase.Collection("bands")
	_, err = bandsCollection.InsertOne(ctx, map[string]interface{}{
		"band_id":   1,
		"name":      "T. Wrecks the Test",
		"owner_pid": 500,
	})

	if err != nil {
		log.Fatalf("Failed to insert test band: %v", err)
	}

	// insert another band for batch lookup tests
	_, err = bandsCollection.InsertOne(ctx, map[string]interface{}{
		"band_id":   2,
		"name":      "The Testifiers",
		"owner_pid": 502,
	})
	if err != nil {
		log.Fatalf("Failed to insert test band: %v", err)
	}

	// insert a config document for counter tests
	configCollection := database.GocentralDatabase.Collection("config")
	_, err = configCollection.InsertOne(ctx, map[string]interface{}{
		"last_pid":          1000,
		"last_band_id":      100,
		"last_character_id": 200,
		"last_setlist_id":   300,
		"last_machine_id":   400,
	})
	if err != nil {
		log.Fatalf("Failed to insert test config: %v", err)
	}

	// insert a test setlist/battle for battle expiry tests
	setlistsCollection := database.GocentralDatabase.Collection("setlists")
	_, err = setlistsCollection.InsertOne(ctx, map[string]interface{}{
		"setlist_id":     1,
		"name":           "Test Battle",
		"owner_pid":      500,
		"created":        time.Now().Unix() - 3600, // created 1 hour ago
		"time_end_val":   2,
		"time_end_units": "hours",
	})
	if err != nil {
		log.Fatalf("Failed to insert test setlist: %v", err)
	}

	// insert an expired setlist/battle
	_, err = setlistsCollection.InsertOne(ctx, map[string]interface{}{
		"setlist_id":     2,
		"name":           "Expired Battle",
		"owner_pid":      500,
		"created":        time.Now().Unix() - 86400, // created 24 hours ago
		"time_end_val":   1,
		"time_end_units": "hours",
	})
	if err != nil {
		log.Fatalf("Failed to insert test setlist: %v", err)
	}

	// insert test scores for GetCoolFact tests
	scoresCollection := database.GocentralDatabase.Collection("scores")
	_, err = scoresCollection.InsertOne(ctx, map[string]interface{}{
		"pid":   500,
		"stars": 5,
		"score": 100000,
	})
	if err != nil {
		log.Fatalf("Failed to insert test score: %v", err)
	}

	// insert test characters for GetCoolFact tests
	charactersCollection := database.GocentralDatabase.Collection("characters")
	_, err = charactersCollection.InsertOne(ctx, map[string]interface{}{
		"character_id": 1,
		"owner_pid":    500,
		"name":         "Test Character",
	})
	if err != nil {
		log.Fatalf("Failed to insert test character: %v", err)
	}

	code := m.Run()

	// drop the test database after running tests
	if err := client.Database("gocentral_test").Drop(ctx); err != nil {
		log.Fatalf("Failed to drop test database: %v", err)
	}

	_ = client.Disconnect(ctx)
	os.Exit(code)
}

// Gets the username from the test database for a given PID
func TestGetUsernameForPID(t *testing.T) {
	pid := 500
	expected := "testuser"

	username := database.GetUsernameForPID(pid)
	if username == "" {
		t.Errorf("Expected a non-empty username for PID %d, got empty string", pid)
	} else {
		t.Logf("Username for PID %d: %s", pid, username)
	}

	if username != expected {
		t.Errorf("Expected username %q for PID %d, got %q", expected, pid, username)
	}
}

func TestGetUsernameForPID_Missing(t *testing.T) {
	pid := 99999
	expected := "Player"

	username := database.GetUsernameForPID(pid)
	if username != expected {
		t.Errorf("Expected username %q for missing PID %d, got %q", expected, pid, username)
	}
}

// Gets the usernames for a list of PIDs from the test database
func TestGetUsernamesForPIDs(t *testing.T) {
	// pids 500, 501, 502
	pids := []int{500, 501, 502}
	expected := map[int]string{
		500: "testuser",
		501: "testuser2",
		502: "testuser3",
	}

	usernames, err := database.GetUsernamesByPIDs(context.Background(), database.GocentralDatabase, pids)
	if err != nil {
		t.Fatalf("Failed to get usernames for PIDs %v: %v", pids, err)
	}

	// check we got something back
	for _, pid := range pids {
		if username, exists := usernames[pid]; exists {
			t.Logf("Username for PID %d: %s", pid, username)
		} else {
			t.Errorf("Expected a username for PID %d, but got none", pid)
		}
	}

	// check that we got the right stuff back
	for pid, expectedUsername := range expected {
		actual, ok := usernames[pid]
		if !ok {
			t.Errorf("Expected username for PID %d, but none found", pid)
			continue
		}
		if actual != expectedUsername {
			t.Errorf("Expected username %q for PID %d, got %q", expectedUsername, pid, actual)
		}
	}
}

// Gets the usernames for a list of PIDs from the test database, including some that do not exist
func TestGetUsernamesByPIDs_Mixed(t *testing.T) {
	t.Run("Mixed existing and missing PIDs", func(t *testing.T) {
		pids := []int{500, 99999, 502, 88888} // 500 and 502 exist, others do not
		expectedCount := 2
		expectedUsernames := map[int]string{
			500: "testuser",
			502: "testuser3",
		}

		usernames, err := database.GetUsernamesByPIDs(context.Background(), database.GocentralDatabase, pids)
		if err != nil {
			t.Fatalf("Got unexpected error: %v", err)
		}

		if len(usernames) != expectedCount {
			t.Errorf("Expected %d usernames, but got %d", expectedCount, len(usernames))
		}

		for pid, expectedName := range expectedUsernames {
			if actualName, ok := usernames[pid]; !ok || actualName != expectedName {
				t.Errorf("For PID %d, expected username %q, got %q (found: %v)", pid, expectedName, actualName, ok)
			}
		}
		t.Logf("Successfully retrieved usernames for mixed PIDs, got %d entries", len(usernames))
	})

	t.Run("Empty PID list", func(t *testing.T) {
		pids := []int{}
		usernames, err := database.GetUsernamesByPIDs(context.Background(), database.GocentralDatabase, pids)
		expectedCount := 0
		if err != nil {
			t.Fatalf("Got unexpected error for empty PID list: %v", err)
		}
		if len(usernames) != 0 {
			t.Errorf("Expected 0 usernames for empty PID list, got %d", len(usernames))
		}
		if len(usernames) != expectedCount {
			t.Errorf("Expected %d usernames for empty PID list, got %d", expectedCount, len(usernames))
		}
		t.Logf("Successfully retrieved usernames for empty PID list, got %d entries", len(usernames))
	})
}

// Gets the PID for a given username from the test database
func TestGetPIDForUsername(t *testing.T) {
	username := "testuser"
	expectedPID := 500

	pid := database.GetPIDForUsername(username)
	if pid == 0 {
		t.Errorf("Expected a non-zero PID for username %q, got 0", username)
	} else {
		t.Logf("PID for username %q: %d", username, pid)
	}

	if pid != expectedPID {
		t.Errorf("Expected PID %d for username %q, got %d", expectedPID, username, pid)
	}
}

// Gets the PID for a given username that does not exist in the test database
func TestGetPIDForUsername_Missing(t *testing.T) {
	username := "this_user_does_not_exist"
	expectedPID := 0

	pid := database.GetPIDForUsername(username)
	if pid != expectedPID {
		t.Errorf("Expected PID %d for missing username %q, got %d", expectedPID, username, pid)
	}
	t.Logf("PID for missing username %q: %d", username, pid)
}

// Tests that GetPIDForUsername is case-insensitive
func TestGetPIDForUsername_CaseInsensitive(t *testing.T) {
	expectedPID := 500

	testCases := []struct {
		name     string
		username string
	}{
		{"Exact case", "testuser"},
		{"All uppercase", "TESTUSER"},
		{"All lowercase", "testuser"},
		{"Mixed case", "TestUser"},
		{"Alternating case", "tEsTuSeR"},
		{"First letter uppercase", "Testuser"},
		{"Last letter uppercase", "testuseR"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pid := database.GetPIDForUsername(tc.username)
			if pid != expectedPID {
				t.Errorf("Expected PID %d for username %q, got %d", expectedPID, tc.username, pid)
			} else {
				t.Logf("Successfully found PID %d for username %q", pid, tc.username)
			}
		})
	}
}

// Tests that CaseInsensitiveUsername properly escapes regex special characters
func TestCaseInsensitiveUsername_SpecialCharacters(t *testing.T) {
	// Insert a user with special regex characters in the username
	ctx := context.Background()
	usersCollection := database.GocentralDatabase.Collection("users")

	// Insert user with regex special characters
	_, err := usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":      700,
		"username": "test.user+name",
	})
	if err != nil {
		t.Fatalf("Failed to insert test user with special chars: %v", err)
	}
	defer usersCollection.DeleteOne(ctx, map[string]interface{}{"pid": 700})

	testCases := []struct {
		name        string
		username    string
		expectedPID int
	}{
		{"Exact match with special chars", "test.user+name", 700},
		{"Case insensitive with special chars", "TEST.USER+NAME", 700},
		{"Mixed case with special chars", "Test.User+Name", 700},
		// These should NOT match because . and + are escaped
		{"Dot as wildcard should not match", "testXuser+name", 0},
		{"Plus as quantifier should not match", "test.userrrname", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pid := database.GetPIDForUsername(tc.username)
			if pid != tc.expectedPID {
				t.Errorf("Expected PID %d for username %q, got %d", tc.expectedPID, tc.username, pid)
			} else {
				t.Logf("Got expected PID %d for username %q", pid, tc.username)
			}
		})
	}
}

// Tests that CaseInsensitiveUsername handles various edge cases
func TestCaseInsensitiveUsername_EdgeCases(t *testing.T) {
	ctx := context.Background()
	usersCollection := database.GocentralDatabase.Collection("users")

	// Insert users with edge case usernames
	edgeCaseUsers := []struct {
		pid      int
		username string
	}{
		{701, "user(with)parens"},
		{702, "user[with]brackets"},
		{703, "user{with}braces"},
		{704, "user$with^special*chars"},
		{705, "user\\with\\backslash"},
	}

	for _, user := range edgeCaseUsers {
		_, err := usersCollection.InsertOne(ctx, map[string]interface{}{
			"pid":      user.pid,
			"username": user.username,
		})
		if err != nil {
			t.Fatalf("Failed to insert edge case user %q: %v", user.username, err)
		}
	}
	defer func() {
		for _, user := range edgeCaseUsers {
			usersCollection.DeleteOne(ctx, map[string]interface{}{"pid": user.pid})
		}
	}()

	for _, user := range edgeCaseUsers {
		t.Run(user.username, func(t *testing.T) {
			// Test exact match
			pid := database.GetPIDForUsername(user.username)
			if pid != user.pid {
				t.Errorf("Exact match: Expected PID %d for username %q, got %d", user.pid, user.username, pid)
			}

			// Test uppercase version
			upperUsername := strings.ToUpper(user.username)
			pid = database.GetPIDForUsername(upperUsername)
			if pid != user.pid {
				t.Errorf("Uppercase: Expected PID %d for username %q, got %d", user.pid, upperUsername, pid)
			}
		})
	}
}

// Gets the console prefixed username for a given PID from the test database
func TestGetConsolePrefixedUsernameForPID(t *testing.T) {
	t.Run("Valid PS3 User", func(t *testing.T) {
		pid := 500
		expected := "testuser [PS3]"
		username := database.GetConsolePrefixedUsernameForPID(pid)
		if username != expected {
			t.Errorf("Expected console-prefixed username %q for PID %d, got %q", expected, pid, username)
		}
		t.Logf("Console-prefixed username for PID %d: %s", pid, username)
	})

	t.Run("Missing User", func(t *testing.T) {
		pid := 99999 // A PID that does not exist
		expected := "Unnamed Player"
		username := database.GetConsolePrefixedUsernameForPID(pid)
		if username != expected {
			t.Errorf("Expected %q for missing PID %d, got %q", expected, pid, username)
		}
	})

	t.Run("User with missing console_type field", func(t *testing.T) {
		pid := 600
		expected := "user_no_console [360]"
		username := database.GetConsolePrefixedUsernameForPID(pid)
		if username != expected {
			t.Errorf("Expected %q for user with missing console_type, got %q", expected, username)
		}
	})

	t.Run("User with invalid console_type value", func(t *testing.T) {
		pid := 601 // The user we added with console_type: 99
		expected := "user_bad_console"
		username := database.GetConsolePrefixedUsernameForPID(pid)
		if username != expected {
			t.Errorf("Expected %q for user with invalid console_type, got %q", expected, username)
		}
	})
}

// Gets the console prefixed usernames for a list of PIDs from the test database
func TestGetConsolePrefixedUsernamesByPIDs(t *testing.T) {
	// pids 500, 501, 502
	pids := []int{500, 501, 502}
	expected := map[int]string{
		500: "testuser [PS3]",
		501: "testuser2 [Wii]",
		502: "testuser3 [Wii]",
	}

	usernames, err := database.GetConsolePrefixedUsernamesByPIDs(context.Background(), database.GocentralDatabase, pids)
	if err != nil {
		t.Fatalf("Failed to get console-prefixed usernames for PIDs %v: %v", pids, err)
	}
	// check we got something back
	for _, pid := range pids {
		if username, exists := usernames[pid]; exists {
			t.Logf("Console-prefixed username for PID %d: %s", pid, username)
		} else {
			t.Errorf("Expected a console-prefixed username for PID %d, but got none", pid)
		}
	}
	// check that we got the right stuff back
	for pid, expectedUsername := range expected {
		actual, ok := usernames[pid]
		if !ok {
			t.Errorf("Expected console-prefixed username for PID %d, but none found", pid)
			continue
		}
		if actual != expectedUsername {
			t.Errorf("Expected console-prefixed username %q for PID %d, got %q", expectedUsername, pid, actual)
		}
	}
}

// Gets the band name for a given band ID from the test database
func TestGetBandNameForBandID(t *testing.T) {
	bandID := 1
	expected := "T. Wrecks the Test"

	bandName := database.GetBandNameForBandID(bandID)
	if bandName == "" {
		t.Errorf("Expected a non-empty band name for band ID %d, got empty string", bandID)
	} else {
		t.Logf("Band name for band ID %d: %s", bandID, bandName)
	}

	if bandName != expected {
		t.Errorf("Expected band name %q for band ID %d, got %q", expected, bandID, bandName)
	}
}

func TestGetBandNameForBandID_Missing(t *testing.T) {
	t.Run("Missing band ID but matching PID exists", func(t *testing.T) {
		bandID := 501
		expected := "testuser2's Band"
		bandName := database.GetBandNameForBandID(bandID)
		if bandName != expected {
			t.Errorf("Expected fallback band name %q, got %q", expected, bandName)
		}
		t.Logf("Band name for band ID %d: %s", bandID, bandName)
	})

	t.Run("Missing band ID and no matching PID", func(t *testing.T) {
		bandID := 99999
		expected := "Player's Band"
		bandName := database.GetBandNameForBandID(bandID)
		if bandName != expected {
			t.Errorf("Expected fallback band name %q, got %q", expected, bandName)
		}
		t.Logf("Band name for missing band ID %d: %s", bandID, bandName)
	})
}

// Gets the band name for a given owner PID from the test database
func TestGetBandNameForOwnerPID(t *testing.T) {
	ownerPID := 500
	expected := "T. Wrecks the Test"

	bandName := database.GetBandNameForOwnerPID(ownerPID)
	if bandName == "" {
		t.Errorf("Expected a non-empty band name for owner PID %d, got empty string", ownerPID)
	} else {
		t.Logf("Band name for owner PID %d: %s", ownerPID, bandName)
	}

	if bandName != expected {
		t.Errorf("Expected band name %q for owner PID %d, got %q", expected, ownerPID, bandName)
	}
}

func TestGetBandNameForOwnerPID_Missing(t *testing.T) {
	t.Run("Owner PID has no band but user exists", func(t *testing.T) {
		ownerPID := 501 // This user exists but does not own the test band
		expected := "testuser2's Band"
		bandName := database.GetBandNameForOwnerPID(ownerPID)
		if bandName != expected {
			t.Errorf("Expected fallback band name %q for owner PID %d, got %q", expected, ownerPID, bandName)
		}
		t.Logf("Band name for owner PID %d: %s", ownerPID, bandName)
	})

	t.Run("Owner PID does not exist as a user", func(t *testing.T) {
		ownerPID := 99999
		expected := "Player's Band"
		bandName := database.GetBandNameForOwnerPID(ownerPID)
		if bandName != expected {
			t.Errorf("Expected fallback band name %q for non-existent owner PID %d, got %q", expected, ownerPID, bandName)
		}
		t.Logf("Band name for non-existent owner PID %d: %s", ownerPID, bandName)
	})
}

// Gets whether or not a PID is a friend of another PID from the test database
func TestIsPIDAFriendOfPID(t *testing.T) {
	pid1 := 500
	pid2 := 501
	ok, err := database.IsPIDAFriendOfPID(pid1, pid2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("Expected PID 501 to be a friend of PID 500")
	}
	t.Logf("PID %d is a friend of PID %d", pid2, pid1)
}

func TestIsPIDAFriendOfPID_NegativeCases(t *testing.T) {
	t.Run("PID is not a friend", func(t *testing.T) {
		pid1 := 500
		pid2 := 999
		ok, err := database.IsPIDAFriendOfPID(pid1, pid2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if ok {
			t.Errorf("Expected PID %d to NOT be a friend of PID %d", pid2, pid1)
		}
	})

	t.Run("Checking against non-existent PID", func(t *testing.T) {
		pid1 := 500
		pid2 := 99999
		ok, err := database.IsPIDAFriendOfPID(pid1, pid2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if ok {
			t.Errorf("Expected non-existent PID %d to NOT be a friend of PID %d", pid2, pid1)
		}
	})
}

// Gets whether or not a PID is in a group from the test database
func TestIsPIDInGroup(t *testing.T) {
	pid := 500
	group := "admin"
	ok := database.IsPIDInGroup(pid, group)
	if !ok {
		t.Errorf("Expected PID %d to be in group '%s'", pid, group)
	}
	t.Logf("PID %d is in group '%s'", pid, group)
}

func TestIsPIDInGroup_NegativeCases(t *testing.T) {
	t.Run("User not in group", func(t *testing.T) {
		pid := 500
		group := "superadmin" // 500 is in 'admin', not 'superadmin'
		ok := database.IsPIDInGroup(pid, group)
		if ok {
			t.Errorf("Expected PID %d to NOT be in group '%s'", pid, group)
		}
	})

	t.Run("User with no groups field", func(t *testing.T) {
		pid := 501 // 501 has no groups
		group := "admin"
		ok := database.IsPIDInGroup(pid, group)
		if ok {
			t.Errorf("Expected PID %d with no groups field to NOT be in any group", pid)
		}
	})

	t.Run("Non-existent user PID", func(t *testing.T) {
		pid := 99999
		group := "admin"
		ok := database.IsPIDInGroup(pid, group)
		if ok {
			t.Errorf("Expected non-existent PID %d to NOT be in any group", pid)
		}
	})
}

// Gets whether or not a PID is a master user from the test database
func TestIsPIDAMasterUser(t *testing.T) {
	pid := 999
	ok := database.IsPIDAMasterUser(pid)
	if !ok {
		t.Errorf("Expected PID 999 to be a master user")
	}

	t.Logf("PID %d is a master user", pid)
}

// Gets the username for a given PID and checks if it is a master user
func TestIsUsernameAMasterUser(t *testing.T) {
	pid := 999
	username := database.GetUsernameForPID(pid)
	ok := database.IsUsernameAMasterUser(username)
	if !ok {
		t.Errorf("Expected username to be recognized as master user")
	}
	t.Logf("Username '%s' is recognized as a master user", username)
}

func TestIsUsernameAMasterUser_Malformed(t *testing.T) {
	testCases := []struct {
		name     string
		username string
		expected bool
	}{
		{"Regular username", "testuser", false},
		{"Malformed with text", "Master User (not a number)", false},
		{"Missing parentheses", "Master User 1234567891234567", false},
		{"Empty string", "", false},
		{"Just a number", "1234567891234567", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ok := database.IsUsernameAMasterUser(tc.username)
			if ok != tc.expected {
				t.Errorf("For username %q, expected %v but got %v", tc.username, tc.expected, ok)
			}
			if ok {
				t.Logf("Username '%s' is recognized as a master user", tc.username)
			} else {
				t.Logf("Username '%s' is NOT recognized as a master user", tc.username)
			}
		})
	}
}

// Gets the machine ID for a given username from the test database
func TestGetMachineIDFromUsername(t *testing.T) {
	pid := 999
	username := database.GetUsernameForPID(pid)
	expected := 1000000000
	if username == "" {
		t.Fatalf("Expected a non-empty username for PID %d, got empty string", pid)
	}
	t.Logf("Username for PID %d: %s", pid, username)

	// Now test the function
	machineID := database.GetMachineIDFromUsername(username)
	if machineID != expected {
		t.Errorf("Expected machine ID %d, got %d", expected, machineID)
	}
}

func TestGetMachineIDFromUsername_EdgeCases(t *testing.T) {
	t.Run("Username is not a master user", func(t *testing.T) {
		username := "testuser"
		expected := 0
		machineID := database.GetMachineIDFromUsername(username)
		if machineID != expected {
			t.Errorf("Expected machine ID %d for non-master user, got %d", expected, machineID)
		}
	})

	t.Run("Master user with non-existent friend code", func(t *testing.T) {
		// This user was added in TestMain
		username := "Master User (0000000000000000)"
		expected := 0
		machineID := database.GetMachineIDFromUsername(username)
		if machineID != expected {
			t.Errorf("Expected machine ID %d for master user with no matching machine, got %d", expected, machineID)
		}
	})

	t.Run("Malformed master user string", func(t *testing.T) {
		username := "Master User (badformat)"
		expected := 0
		machineID := database.GetMachineIDFromUsername(username)
		if machineID != expected {
			t.Errorf("Expected machine ID %d for malformed master user string, got %d", expected, machineID)
		}
	})
}

// Tests the GetBandNamesByOwnerPIDs batch lookup function
func TestGetBandNamesByOwnerPIDs(t *testing.T) {
	t.Run("Valid owner PIDs", func(t *testing.T) {
		ownerPIDs := []int{500, 502}
		expected := map[int]string{
			500: "T. Wrecks the Test",
			502: "The Testifiers",
		}

		bandNames, err := database.GetBandNamesByOwnerPIDs(context.Background(), database.GocentralDatabase, ownerPIDs)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		for pid, expectedName := range expected {
			if actualName, ok := bandNames[pid]; !ok || actualName != expectedName {
				t.Errorf("For owner PID %d, expected band name %q, got %q (found: %v)", pid, expectedName, actualName, ok)
			}
		}
		t.Logf("Successfully retrieved band names for owner PIDs: %v", bandNames)
	})

	t.Run("Mixed existing and missing owner PIDs", func(t *testing.T) {
		ownerPIDs := []int{500, 99999}
		expectedCount := 1

		bandNames, err := database.GetBandNamesByOwnerPIDs(context.Background(), database.GocentralDatabase, ownerPIDs)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(bandNames) != expectedCount {
			t.Errorf("Expected %d band names, got %d", expectedCount, len(bandNames))
		}

		if bandNames[500] != "T. Wrecks the Test" {
			t.Errorf("Expected band name 'T. Wrecks the Test' for PID 500, got %q", bandNames[500])
		}
	})

	t.Run("Empty owner PID list", func(t *testing.T) {
		ownerPIDs := []int{}

		bandNames, err := database.GetBandNamesByOwnerPIDs(context.Background(), database.GocentralDatabase, ownerPIDs)
		if err != nil {
			t.Fatalf("Unexpected error for empty owner PID list: %v", err)
		}

		if len(bandNames) != 0 {
			t.Errorf("Expected 0 band names for empty owner PID list, got %d", len(bandNames))
		}
	})
}

// Tests the GenerateLinkCode function
func TestGenerateLinkCode(t *testing.T) {
	t.Run("Generates correct length", func(t *testing.T) {
		lengths := []int{5, 10, 15, 20}
		for _, length := range lengths {
			code := database.GenerateLinkCode(length)
			if len(code) != length {
				t.Errorf("Expected link code of length %d, got %d", length, len(code))
			}
		}
	})

	t.Run("Contains only valid characters", func(t *testing.T) {
		validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
		code := database.GenerateLinkCode(100) // Generate a long code to test randomness

		for _, char := range code {
			found := false
			for _, validChar := range validChars {
				if char == validChar {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Link code contains invalid character: %c", char)
			}
		}
	})

	t.Run("Generates unique codes", func(t *testing.T) {
		codes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code := database.GenerateLinkCode(10)
			if codes[code] {
				t.Logf("Warning: Duplicate code generated: %s (this can happen rarely due to randomness)", code)
			}
			codes[code] = true
		}
		// With 62^10 possible combinations, duplicates in 100 tries should be extremely rare
		if len(codes) < 95 {
			t.Errorf("Too many duplicate codes generated: only %d unique codes out of 100", len(codes))
		}
	})

	t.Run("Zero length returns empty string", func(t *testing.T) {
		code := database.GenerateLinkCode(0)
		if code != "" {
			t.Errorf("Expected empty string for length 0, got %q", code)
		}
	})
}

// Tests the GetBattleExpiryInfo function
func TestGetBattleExpiryInfo(t *testing.T) {
	t.Run("Active battle (not expired)", func(t *testing.T) {
		// Battle ID 1 was created 1 hour ago with 2 hour duration
		expired, expiryTime := database.GetBattleExpiryInfo(1)

		if expired {
			t.Errorf("Expected battle 1 to NOT be expired, but it was")
		}

		if expiryTime.IsZero() {
			t.Error("Expected non-zero expiry time")
		}

		t.Logf("Battle 1 expires at: %v (expired: %v)", expiryTime, expired)
	})

	t.Run("Expired battle", func(t *testing.T) {
		// Battle ID 2 was created 24 hours ago with 1 hour duration
		expired, expiryTime := database.GetBattleExpiryInfo(2)

		if !expired {
			t.Errorf("Expected battle 2 to be expired, but it was not")
		}

		if expiryTime.IsZero() {
			t.Error("Expected non-zero expiry time")
		}

		t.Logf("Battle 2 expired at: %v (expired: %v)", expiryTime, expired)
	})

	t.Run("Non-existent battle", func(t *testing.T) {
		// Battle ID 99999 does not exist
		expired, expiryTime := database.GetBattleExpiryInfo(99999)

		// When battle doesn't exist, created time is 0 (Unix epoch)
		// and expiry time should also be zero-ish (Unix epoch + duration)
		t.Logf("Non-existent battle expiry: %v (expired: %v)", expiryTime, expired)
	})
}

// Tests the GetCachedConfig and InvalidateConfigCache functions
func TestGetCachedConfig(t *testing.T) {
	t.Run("Fetches config successfully", func(t *testing.T) {
		config, err := database.GetCachedConfig(context.Background())
		if err != nil {
			t.Fatalf("Failed to get cached config: %v", err)
		}

		if config == nil {
			t.Fatal("Expected non-nil config")
		}

		// Don't check for specific values since other tests may have incremented counters
		// Just verify the config has reasonable values (counters should be > 0)
		if config.LastPID <= 0 {
			t.Errorf("Expected LastPID to be positive, got %d", config.LastPID)
		}

		t.Logf("Config fetched: LastPID=%d, LastBandID=%d", config.LastPID, config.LastBandID)
	})

	t.Run("Returns cached config on subsequent calls", func(t *testing.T) {
		config1, err := database.GetCachedConfig(context.Background())
		if err != nil {
			t.Fatalf("Failed to get first cached config: %v", err)
		}

		config2, err := database.GetCachedConfig(context.Background())
		if err != nil {
			t.Fatalf("Failed to get second cached config: %v", err)
		}

		// Both calls should return the same values
		if config1.LastPID != config2.LastPID {
			t.Errorf("Cached config values differ: %d vs %d", config1.LastPID, config2.LastPID)
		}
	})

	t.Run("Invalidate cache works", func(t *testing.T) {
		// Get config to populate cache
		_, err := database.GetCachedConfig(context.Background())
		if err != nil {
			t.Fatalf("Failed to get cached config: %v", err)
		}

		// Invalidate cache
		database.InvalidateConfigCache()

		// Fetch again - should not panic or error
		config, err := database.GetCachedConfig(context.Background())
		if err != nil {
			t.Fatalf("Failed to get config after invalidation: %v", err)
		}

		if config == nil {
			t.Fatal("Expected non-nil config after invalidation")
		}
	})
}

// Tests the counter functions (GetNextPID, GetNextBandID, etc.)
func TestGetNextPID(t *testing.T) {
	ctx := context.Background()

	pid1, err := database.GetNextPID(ctx)
	if err != nil {
		t.Fatalf("Failed to get next PID: %v", err)
	}

	pid2, err := database.GetNextPID(ctx)
	if err != nil {
		t.Fatalf("Failed to get second PID: %v", err)
	}

	if pid2 != pid1+1 {
		t.Errorf("Expected sequential PIDs, got %d and %d", pid1, pid2)
	}

	t.Logf("Got PIDs: %d, %d", pid1, pid2)
}

func TestGetNextBandID(t *testing.T) {
	ctx := context.Background()

	bandID1, err := database.GetNextBandID(ctx)
	if err != nil {
		t.Fatalf("Failed to get next band ID: %v", err)
	}

	bandID2, err := database.GetNextBandID(ctx)
	if err != nil {
		t.Fatalf("Failed to get second band ID: %v", err)
	}

	if bandID2 != bandID1+1 {
		t.Errorf("Expected sequential band IDs, got %d and %d", bandID1, bandID2)
	}

	t.Logf("Got band IDs: %d, %d", bandID1, bandID2)
}

func TestGetNextCharacterID(t *testing.T) {
	ctx := context.Background()

	charID1, err := database.GetNextCharacterID(ctx)
	if err != nil {
		t.Fatalf("Failed to get next character ID: %v", err)
	}

	charID2, err := database.GetNextCharacterID(ctx)
	if err != nil {
		t.Fatalf("Failed to get second character ID: %v", err)
	}

	if charID2 != charID1+1 {
		t.Errorf("Expected sequential character IDs, got %d and %d", charID1, charID2)
	}

	t.Logf("Got character IDs: %d, %d", charID1, charID2)
}

func TestGetNextSetlistID(t *testing.T) {
	ctx := context.Background()

	setlistID1, err := database.GetNextSetlistID(ctx)
	if err != nil {
		t.Fatalf("Failed to get next setlist ID: %v", err)
	}

	setlistID2, err := database.GetNextSetlistID(ctx)
	if err != nil {
		t.Fatalf("Failed to get second setlist ID: %v", err)
	}

	if setlistID2 != setlistID1+1 {
		t.Errorf("Expected sequential setlist IDs, got %d and %d", setlistID1, setlistID2)
	}

	t.Logf("Got setlist IDs: %d, %d", setlistID1, setlistID2)
}

func TestGetNextMachineID(t *testing.T) {
	ctx := context.Background()

	machineID1, err := database.GetNextMachineID(ctx)
	if err != nil {
		t.Fatalf("Failed to get next machine ID: %v", err)
	}

	machineID2, err := database.GetNextMachineID(ctx)
	if err != nil {
		t.Fatalf("Failed to get second machine ID: %v", err)
	}

	if machineID2 != machineID1+1 {
		t.Errorf("Expected sequential machine IDs, got %d and %d", machineID1, machineID2)
	}

	t.Logf("Got machine IDs: %d, %d", machineID1, machineID2)
}

// Tests the GetCoolFact function
func TestGetCoolFact(t *testing.T) {
	// Run multiple times to exercise different random paths
	facts := make(map[string]int)
	for i := 0; i < 20; i++ {
		fact := database.GetCoolFact()
		if fact == "" {
			t.Error("GetCoolFact returned empty string")
		}
		facts[fact]++
	}

	t.Logf("Got %d unique facts from 20 calls", len(facts))
	for fact, count := range facts {
		t.Logf("  %q (appeared %d times)", fact, count)
	}

	// Should have at least 1 fact (could be same if random keeps returning same number)
	if len(facts) == 0 {
		t.Error("Expected at least one fact to be returned")
	}
}

// Tests that IsPIDBanned is case-insensitive
func TestIsPIDBanned_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")
	usersCollection := database.GocentralDatabase.Collection("users")

	// Create a test user with mixed case username
	testPID := 800
	testUsername := "MixedCaseUser"
	_, err := usersCollection.InsertOne(ctx, map[string]interface{}{
		"pid":      testPID,
		"username": testUsername,
	})
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}
	defer usersCollection.DeleteOne(ctx, map[string]interface{}{"pid": testPID})

	// Add a ban with DIFFERENT case than the actual username
	bannedUsername := "MIXEDCASEUSER" // All uppercase, but user is "MixedCaseUser"
	_, err = configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$push": bson.M{"banned_players": bson.M{
			"username":   bannedUsername,
			"reason":     "Test case-insensitive ban",
			"expires_at": time.Time{}, // Permanent
			"created_at": time.Now(),
		}},
	})
	if err != nil {
		t.Fatalf("Failed to add ban: %v", err)
	}
	defer configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": bannedUsername}},
	})

	// Invalidate cache to pick up the new ban
	database.InvalidateConfigCache()

	// Test that the user is detected as banned even with different case
	isBanned := database.IsPIDBanned(testPID)
	if !isBanned {
		t.Errorf("Expected PID %d (username %q) to be banned when ban list has %q", testPID, testUsername, bannedUsername)
	} else {
		t.Logf("Successfully detected PID %d as banned (username %q matched ban %q)", testPID, testUsername, bannedUsername)
	}
}

// Tests that IsUsernameBanned is case-insensitive
func TestIsUsernameBanned_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	// Add a ban with specific casing
	bannedUsername := "BannedTestPlayer"
	_, err := configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$push": bson.M{"banned_players": bson.M{
			"username":   bannedUsername,
			"reason":     "Test case-insensitive ban check",
			"expires_at": time.Time{}, // Permanent
			"created_at": time.Now(),
		}},
	})
	if err != nil {
		t.Fatalf("Failed to add ban: %v", err)
	}
	defer configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": bannedUsername}},
	})

	// Invalidate cache
	database.InvalidateConfigCache()

	testCases := []struct {
		name     string
		username string
	}{
		{"Exact case", "BannedTestPlayer"},
		{"All lowercase", "bannedtestplayer"},
		{"All uppercase", "BANNEDTESTPLAYER"},
		{"Mixed case", "bANNEDtESTpLAYER"},
		{"Alternating", "BaNnEdTeStPlAyEr"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isBanned := database.IsUsernameBanned(tc.username)
			if !isBanned {
				t.Errorf("Expected username %q to be detected as banned (ban list has %q)", tc.username, bannedUsername)
			} else {
				t.Logf("Successfully detected %q as banned", tc.username)
			}
		})
	}
}

// Tests edge cases for IsPIDInGroup
func TestIsPIDInGroup_EdgeCases(t *testing.T) {
	t.Run("Zero PID", func(t *testing.T) {
		ok := database.IsPIDInGroup(0, "admin")
		if ok {
			t.Error("Expected false for zero PID")
		}
	})

	t.Run("Empty group", func(t *testing.T) {
		ok := database.IsPIDInGroup(500, "")
		if ok {
			t.Error("Expected false for empty group")
		}
	})

	t.Run("Both zero PID and empty group", func(t *testing.T) {
		ok := database.IsPIDInGroup(0, "")
		if ok {
			t.Error("Expected false for zero PID and empty group")
		}
	})
}

// Tests edge cases for IsPIDAMasterUser
func TestIsPIDAMasterUser_NegativeCases(t *testing.T) {
	t.Run("Regular user is not a master user", func(t *testing.T) {
		ok := database.IsPIDAMasterUser(500)
		if ok {
			t.Error("Expected PID 500 to NOT be a master user")
		}
	})

	t.Run("Non-existent PID", func(t *testing.T) {
		ok := database.IsPIDAMasterUser(99999)
		if ok {
			t.Error("Expected non-existent PID to NOT be a master user")
		}
	})

	t.Run("Zero PID", func(t *testing.T) {
		ok := database.IsPIDAMasterUser(0)
		if ok {
			t.Error("Expected zero PID to NOT be a master user")
		}
	})
}
