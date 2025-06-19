package tests

import (
	"context"
	"log"
	"os"
	"rb3server/database"
	"testing"

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
		log.Fatalf("Failed to insert test user: %v", err)
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
