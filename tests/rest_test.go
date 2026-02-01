package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rb3server/database"
	"rb3server/restapi"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Helper to make a request and return the response
func makeRequest(t *testing.T, method, path string, body interface{}, handler http.HandlerFunc) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

// Helper to make an authenticated request
func makeAuthRequest(t *testing.T, method, path string, body interface{}, handler http.Handler, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

// Helper to decode JSON response
func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("Failed to decode response: %v (body: %s)", err, rr.Body.String())
	}
}

// Tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	rr := makeRequest(t, "GET", "/health", nil, restapi.HealthHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]string
	decodeResponse(t, rr, &response)

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", response["status"])
	}

	t.Log("HealthHandler: returned healthy status")
}

// Tests the stats endpoint
func TestStatsHandler(t *testing.T) {
	rr := makeRequest(t, "GET", "/stats", nil, restapi.StatsHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var response restapi.Stats
	decodeResponse(t, rr, &response)

	// Just verify we got some stats back (values depend on test data)
	t.Logf("StatsHandler: Scores=%d, Machines=%d, Bands=%d, Characters=%d, Setlists=%d",
		response.Scores, response.Machines, response.Bands, response.Characters, response.Setlists)
}

// Tests the song list endpoint
func TestSongListHandler(t *testing.T) {
	rr := makeRequest(t, "GET", "/songs", nil, restapi.SongListHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string][]int
	decodeResponse(t, rr, &response)

	songs, ok := response["songs"]
	if !ok {
		t.Error("Response missing 'songs' field")
	}

	t.Logf("SongListHandler: returned %d unique songs", len(songs))
}

// Tests the MOTD endpoint
func TestMotdHandler(t *testing.T) {
	rr := makeRequest(t, "GET", "/motd", nil, restapi.MotdHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var response map[string]string
	decodeResponse(t, rr, &response)

	// MOTD fields should exist (may be empty)
	if _, ok := response["motd"]; !ok {
		t.Error("Response missing 'motd' field")
	}
	if _, ok := response["dlcmotd"]; !ok {
		t.Error("Response missing 'dlcmotd' field")
	}

	t.Logf("MotdHandler: motd=%q, dlcmotd=%q", response["motd"], response["dlcmotd"])
}

// Tests the battle list endpoint
func TestBattleListHandler(t *testing.T) {
	ctx := context.Background()
	setlistsCollection := database.GocentralDatabase.Collection("setlists")

	// Insert a test Harmonix battle
	testBattle := map[string]interface{}{
		"setlist_id":     555555,
		"type":           1002, // Harmonix battle
		"title":          "REST API Test Battle",
		"desc":           "A battle for testing",
		"created":        time.Now().Unix(),
		"time_end_val":   24,
		"time_end_units": "hours",
		"song_ids":       []int{100, 101, 102},
		"instrument":     1,
		"owner_pid":      500,
	}
	_, err := setlistsCollection.InsertOne(ctx, testBattle)
	if err != nil {
		t.Fatalf("Failed to insert test battle: %v", err)
	}

	// Make request
	rr := makeRequest(t, "GET", "/battles", nil, restapi.BattleListHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string][]restapi.GlobalBattleInfo
	decodeResponse(t, rr, &response)

	battles, ok := response["battles"]
	if !ok {
		t.Error("Response missing 'battles' field")
	}

	// Find our test battle
	found := false
	for _, battle := range battles {
		if battle.BattleID == 555555 {
			found = true
			if battle.Title != "REST API Test Battle" {
				t.Errorf("Expected title 'REST API Test Battle', got %q", battle.Title)
			}
			break
		}
	}

	if !found {
		t.Error("Test battle not found in response")
	}

	// Cleanup
	setlistsCollection.DeleteOne(ctx, bson.M{"setlist_id": 555555})

	t.Logf("BattleListHandler: returned %d battles", len(battles))
}

// Tests the leaderboard endpoint with valid parameters
func TestLeaderboardHandler_ValidParams(t *testing.T) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Insert some test scores
	testSongID := 444444
	for i := 0; i < 5; i++ {
		score := map[string]interface{}{
			"pid":             500 + i,
			"song_id":         testSongID,
			"role_id":         1,
			"score":           100000 - (i * 10000),
			"stars":           5,
			"diff_id":         2,
			"notespct":        95 - i,
			"instrument_mask": 1,
		}
		_, err := scoresCollection.InsertOne(ctx, score)
		if err != nil {
			t.Fatalf("Failed to insert test score: %v", err)
		}
	}

	// Make request
	req := httptest.NewRequest("GET", "/leaderboard?song_id=444444&role_id=1", nil)
	rr := httptest.NewRecorder()
	restapi.LeaderboardHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var response map[string][]restapi.LeaderboardEntry
	decodeResponse(t, rr, &response)

	leaderboard, ok := response["leaderboard"]
	if !ok {
		t.Error("Response missing 'leaderboard' field")
	}

	if len(leaderboard) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(leaderboard))
	}

	// Verify sorted by score descending
	for i := 1; i < len(leaderboard); i++ {
		if leaderboard[i].Score > leaderboard[i-1].Score {
			t.Error("Leaderboard not sorted by score descending")
			break
		}
	}

	// Cleanup
	scoresCollection.DeleteMany(ctx, bson.M{"song_id": testSongID})

	t.Logf("LeaderboardHandler: returned %d entries, correctly sorted", len(leaderboard))
}

// Tests the leaderboard endpoint with missing parameters
func TestLeaderboardHandler_MissingParams(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		expected int
	}{
		{"Missing both", "/leaderboard", http.StatusBadRequest},
		{"Missing song_id", "/leaderboard?role_id=1", http.StatusBadRequest},
		{"Missing role_id", "/leaderboard?song_id=100", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.query, nil)
			rr := httptest.NewRecorder()
			restapi.LeaderboardHandler(rr, req)

			if rr.Code != tc.expected {
				t.Errorf("Expected status %d, got %d", tc.expected, rr.Code)
			}
		})
	}
}

// Tests the leaderboard endpoint with pagination
func TestLeaderboardHandler_Pagination(t *testing.T) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	testSongID := 333333

	// Insert 25 test scores
	for i := 0; i < 25; i++ {
		score := map[string]interface{}{
			"pid":             600 + i,
			"song_id":         testSongID,
			"role_id":         1,
			"score":           100000 - (i * 1000),
			"stars":           5,
			"diff_id":         2,
			"notespct":        95,
			"instrument_mask": 1,
		}
		scoresCollection.InsertOne(ctx, score)
	}

	// Test page 1 with default page size (20)
	req := httptest.NewRequest("GET", "/leaderboard?song_id=333333&role_id=1&page=1", nil)
	rr := httptest.NewRecorder()
	restapi.LeaderboardHandler(rr, req)

	var response1 map[string][]restapi.LeaderboardEntry
	decodeResponse(t, rr, &response1)

	if len(response1["leaderboard"]) != 20 {
		t.Errorf("Page 1: expected 20 entries, got %d", len(response1["leaderboard"]))
	}

	// Test page 2
	req = httptest.NewRequest("GET", "/leaderboard?song_id=333333&role_id=1&page=2", nil)
	rr = httptest.NewRecorder()
	restapi.LeaderboardHandler(rr, req)

	var response2 map[string][]restapi.LeaderboardEntry
	decodeResponse(t, rr, &response2)

	if len(response2["leaderboard"]) != 5 {
		t.Errorf("Page 2: expected 5 entries, got %d", len(response2["leaderboard"]))
	}

	// Test custom page size
	req = httptest.NewRequest("GET", "/leaderboard?song_id=333333&role_id=1&page=1&page_size=10", nil)
	rr = httptest.NewRecorder()
	restapi.LeaderboardHandler(rr, req)

	var response3 map[string][]restapi.LeaderboardEntry
	decodeResponse(t, rr, &response3)

	if len(response3["leaderboard"]) != 10 {
		t.Errorf("Custom page size: expected 10 entries, got %d", len(response3["leaderboard"]))
	}

	// Cleanup
	scoresCollection.DeleteMany(ctx, bson.M{"song_id": testSongID})

	t.Log("LeaderboardHandler: pagination works correctly")
}

// Tests the battle leaderboard endpoint
func TestBattleLeaderboardHandler(t *testing.T) {
	ctx := context.Background()
	scoresCollection := database.GocentralDatabase.Collection("scores")

	testBattleID := 222222

	// Insert test scores for the battle
	for i := 0; i < 3; i++ {
		score := map[string]interface{}{
			"pid":             500 + i,
			"battle_id":       testBattleID,
			"song_id":         100,
			"role_id":         1,
			"score":           50000 - (i * 10000),
			"stars":           5,
			"diff_id":         2,
			"notespct":        95,
			"instrument_mask": 1,
		}
		scoresCollection.InsertOne(ctx, score)
	}

	req := httptest.NewRequest("GET", "/battle_leaderboard?battle_id=222222", nil)
	rr := httptest.NewRecorder()
	restapi.BattleLeaderboardHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string][]restapi.BattleLeaderboardEntry
	decodeResponse(t, rr, &response)

	leaderboard := response["leaderboard"]
	if len(leaderboard) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(leaderboard))
	}

	// Cleanup
	scoresCollection.DeleteMany(ctx, bson.M{"battle_id": testBattleID})

	t.Logf("BattleLeaderboardHandler: returned %d entries", len(leaderboard))
}

// Tests battle leaderboard with missing battle_id
func TestBattleLeaderboardHandler_MissingBattleID(t *testing.T) {
	req := httptest.NewRequest("GET", "/battle_leaderboard", nil)
	rr := httptest.NewRecorder()
	restapi.BattleLeaderboardHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// Tests the list banned players endpoint
func TestListBannedPlayersHandler(t *testing.T) {
	rr := makeRequest(t, "GET", "/admin/bans", nil, restapi.ListBannedPlayersHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	decodeResponse(t, rr, &response)

	if _, ok := response["banned_players"]; !ok {
		t.Error("Response missing 'banned_players' field")
	}

	t.Log("ListBannedPlayersHandler: returned ban list")
}

// Tests standard headers are added
func TestAddStandardHeaders(t *testing.T) {
	rr := httptest.NewRecorder()
	restapi.AddStandardHeaders(rr)

	expectedHeaders := map[string]string{
		"Server":                      "GoCentral",
		"X-Clacks-Overhead":           "GNU maxton",
		"Access-Control-Allow-Origin": "*",
	}

	for key, expected := range expectedHeaders {
		actual := rr.Header().Get(key)
		if actual != expected {
			t.Errorf("Header %s: expected %q, got %q", key, expected, actual)
		}
	}

	t.Log("AddStandardHeaders: all headers set correctly")
}

// Tests the ban player endpoint
func TestBanPlayerHandler(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	// Get initial ban count
	var initialConfig struct {
		BannedPlayers []interface{} `bson:"banned_players"`
	}
	configCollection.FindOne(ctx, bson.M{}).Decode(&initialConfig)
	initialBanCount := len(initialConfig.BannedPlayers)

	// Test banning a player
	banRequest := restapi.BanPlayerRequest{
		Username: "test_banned_user",
		Reason:   "Testing ban functionality",
		Duration: "1h",
	}

	rr := makeRequest(t, "POST", "/admin/ban", banRequest, restapi.BanPlayerHandler)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Verify ban was added
	var updatedConfig struct {
		BannedPlayers []interface{} `bson:"banned_players"`
	}
	configCollection.FindOne(ctx, bson.M{}).Decode(&updatedConfig)

	if len(updatedConfig.BannedPlayers) != initialBanCount+1 {
		t.Errorf("Expected %d bans, got %d", initialBanCount+1, len(updatedConfig.BannedPlayers))
	}

	// Cleanup: remove the test ban by pulling it from the array
	configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": "test_banned_user"}},
	})

	t.Log("BanPlayerHandler: successfully added ban")
}

// Tests ban player with invalid duration
func TestBanPlayerHandler_InvalidDuration(t *testing.T) {
	banRequest := restapi.BanPlayerRequest{
		Username: "test_user",
		Reason:   "Test",
		Duration: "invalid_duration",
	}

	rr := makeRequest(t, "POST", "/admin/ban", banRequest, restapi.BanPlayerHandler)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid duration, got %d", rr.Code)
	}
}

// Tests ban player with missing username
func TestBanPlayerHandler_MissingUsername(t *testing.T) {
	banRequest := restapi.BanPlayerRequest{
		Reason:   "Test",
		Duration: "1h",
	}

	rr := makeRequest(t, "POST", "/admin/ban", banRequest, restapi.BanPlayerHandler)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing username, got %d", rr.Code)
	}
}

// Tests permanent ban
func TestBanPlayerHandler_PermanentBan(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	banRequest := restapi.BanPlayerRequest{
		Username: "permanently_banned_user",
		Reason:   "Permanent ban test",
		Duration: "permanent",
	}

	rr := makeRequest(t, "POST", "/admin/ban", banRequest, restapi.BanPlayerHandler)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rr.Code)
	}

	// Cleanup
	configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": "permanently_banned_user"}},
	})

	t.Log("BanPlayerHandler: permanent ban works correctly")
}

// Tests the unban player endpoint
func TestUnbanPlayerHandler(t *testing.T) {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	// First, add a ban to unban
	banRequest := restapi.BanPlayerRequest{
		Username: "user_to_unban",
		Reason:   "Will be unbanned",
		Duration: "24h",
	}
	makeRequest(t, "POST", "/admin/ban", banRequest, restapi.BanPlayerHandler)

	// Now unban
	unbanRequest := restapi.UnbanPlayerRequest{
		Username: "user_to_unban",
	}

	rr := makeRequest(t, "POST", "/admin/unban", unbanRequest, restapi.UnbanPlayerHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// Verify response
	var response map[string]interface{}
	decodeResponse(t, rr, &response)

	if response["success"] != true {
		t.Error("Expected success to be true")
	}

	// Cleanup
	configCollection.UpdateOne(ctx, bson.M{}, bson.M{
		"$pull": bson.M{"banned_players": bson.M{"username": "user_to_unban"}},
	})

	t.Log("UnbanPlayerHandler: successfully unbanned player")
}

// Tests unban with non-existent user
func TestUnbanPlayerHandler_NotFound(t *testing.T) {
	unbanRequest := restapi.UnbanPlayerRequest{
		Username: "nonexistent_user_12345",
	}

	rr := makeRequest(t, "POST", "/admin/unban", unbanRequest, restapi.UnbanPlayerHandler)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

// Tests unban with missing username
func TestUnbanPlayerHandler_MissingUsername(t *testing.T) {
	unbanRequest := restapi.UnbanPlayerRequest{}

	rr := makeRequest(t, "POST", "/admin/unban", unbanRequest, restapi.UnbanPlayerHandler)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// Tests create battle endpoint validation
func TestCreateBattleHandler_Validation(t *testing.T) {
	testCases := []struct {
		name     string
		request  restapi.CreateBattleRequest
		expected int
	}{
		{
			name: "Missing title",
			request: restapi.CreateBattleRequest{
				SongIDs:   []int{100},
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "Missing song IDs",
			request: restapi.CreateBattleRequest{
				Title:     "Test Battle",
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "Expired time in past",
			request: restapi.CreateBattleRequest{
				Title:     "Test Battle",
				SongIDs:   []int{100},
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := makeRequest(t, "POST", "/admin/battles", tc.request, restapi.CreateBattleHandler)

			if rr.Code != tc.expected {
				t.Errorf("Expected status %d, got %d (body: %s)", tc.expected, rr.Code, rr.Body.String())
			}
		})
	}
}

// Tests create battle endpoint with valid data
func TestCreateBattleHandler_Valid(t *testing.T) {
	ctx := context.Background()
	setlistsCollection := database.GocentralDatabase.Collection("setlists")

	request := restapi.CreateBattleRequest{
		Title:       "API Created Battle",
		Description: "Created via REST API test",
		SongIDs:     []int{100, 101},
		ExpiresAt:   time.Now().Add(48 * time.Hour),
		Instrument:  1,
	}

	rr := makeRequest(t, "POST", "/admin/battles", request, restapi.CreateBattleHandler)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	decodeResponse(t, rr, &response)

	if response["success"] != true {
		t.Error("Expected success to be true")
	}

	battleID, ok := response["battle_id"].(float64)
	if !ok || battleID == 0 {
		t.Error("Expected valid battle_id in response")
	}

	// Cleanup
	setlistsCollection.DeleteOne(ctx, bson.M{"setlist_id": int(battleID)})

	t.Logf("CreateBattleHandler: created battle with ID %d", int(battleID))
}

// Tests delete battle endpoint
func TestDeleteBattleHandler(t *testing.T) {
	ctx := context.Background()
	setlistsCollection := database.GocentralDatabase.Collection("setlists")
	scoresCollection := database.GocentralDatabase.Collection("scores")

	// Create a battle to delete
	testBattleID := 111111
	battle := map[string]interface{}{
		"setlist_id":     testBattleID,
		"type":           1002,
		"title":          "Battle to Delete",
		"created":        time.Now().Unix(),
		"time_end_val":   24,
		"time_end_units": "hours",
	}
	setlistsCollection.InsertOne(ctx, battle)

	// Add some scores
	for i := 0; i < 3; i++ {
		score := map[string]interface{}{
			"pid":       500 + i,
			"battle_id": testBattleID,
			"score":     10000,
			"song_id":   100,
			"role_id":   1,
			"stars":     5,
			"diff_id":   2,
			"notespct":  95,
		}
		scoresCollection.InsertOne(ctx, score)
	}

	// Delete the battle
	request := restapi.DeleteBattleRequest{
		BattleID: testBattleID,
	}

	rr := makeRequest(t, "DELETE", "/admin/battles", request, restapi.DeleteBattleHandler)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	decodeResponse(t, rr, &response)

	if response["success"] != true {
		t.Error("Expected success to be true")
	}

	scoresDeleted, ok := response["scores_deleted"].(float64)
	if !ok {
		t.Error("Expected scores_deleted in response")
	}

	if int(scoresDeleted) != 3 {
		t.Errorf("Expected 3 scores deleted, got %d", int(scoresDeleted))
	}

	// Verify battle is gone
	count, _ := setlistsCollection.CountDocuments(ctx, bson.M{"setlist_id": testBattleID})
	if count != 0 {
		t.Error("Battle still exists after deletion")
	}

	// Verify scores are gone
	scoreCount, _ := scoresCollection.CountDocuments(ctx, bson.M{"battle_id": testBattleID})
	if scoreCount != 0 {
		t.Error("Battle scores still exist after deletion")
	}

	t.Log("DeleteBattleHandler: successfully deleted battle and scores")
}

// Tests delete battle with non-existent battle
func TestDeleteBattleHandler_NotFound(t *testing.T) {
	request := restapi.DeleteBattleRequest{
		BattleID: 99999999,
	}

	rr := makeRequest(t, "DELETE", "/admin/battles", request, restapi.DeleteBattleHandler)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

// Tests delete battle with missing battle_id
func TestDeleteBattleHandler_MissingBattleID(t *testing.T) {
	request := restapi.DeleteBattleRequest{}

	rr := makeRequest(t, "DELETE", "/admin/battles", request, restapi.DeleteBattleHandler)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// ============================================
// Admin Authentication Middleware Tests
// ============================================

// Sets up a test admin token in the config
func setupTestAdminToken(t *testing.T, token string) func() {
	ctx := context.Background()
	configCollection := database.GocentralDatabase.Collection("config")

	// Store the original token to restore later
	var originalConfig struct {
		AdminAPIToken string `bson:"admin_api_token"`
	}
	configCollection.FindOne(ctx, bson.M{}).Decode(&originalConfig)

	// Set the test token
	_, err := configCollection.UpdateOne(ctx, bson.M{}, bson.M{"$set": bson.M{"admin_api_token": token}})
	if err != nil {
		t.Fatalf("Failed to set test admin token: %v", err)
	}

	// Invalidate the config cache so the new token is picked up
	database.InvalidateConfigCache()

	// Return cleanup function
	return func() {
		configCollection.UpdateOne(ctx, bson.M{}, bson.M{"$set": bson.M{"admin_api_token": originalConfig.AdminAPIToken}})
		database.InvalidateConfigCache()
	}
}

// Tests that requests without Authorization header are rejected
func TestAdminTokenAuth_NoHeader(t *testing.T) {
	cleanup := setupTestAdminToken(t, "test-secret-token")
	defer cleanup()

	// Create a protected handler
	protectedHandler := restapi.AdminTokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))

	req := httptest.NewRequest("GET", "/admin/test", nil)
	rr := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 (auth error), got %d", rr.Code)
	}

	var response map[string]string
	json.NewDecoder(rr.Body).Decode(&response)

	if response["error"] != "Could not verify authorization" {
		t.Errorf("Expected auth error message, got %q", response["error"])
	}

	t.Log("AdminTokenAuth: correctly rejects requests without Authorization header")
}

// Tests that requests with malformed Authorization header are rejected
func TestAdminTokenAuth_MalformedHeader(t *testing.T) {
	cleanup := setupTestAdminToken(t, "test-secret-token")
	defer cleanup()

	protectedHandler := restapi.AdminTokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	testCases := []struct {
		name   string
		header string
	}{
		{"No Bearer prefix", "test-secret-token"},
		{"Basic auth instead", "Basic dXNlcjpwYXNz"},
		{"Empty Bearer", "Bearer "},
		{"Double Bearer", "Bearer Bearer token"},
		{"Just Bearer", "Bearer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin/test", nil)
			req.Header.Set("Authorization", tc.header)
			rr := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 500 for %q, got %d", tc.name, rr.Code)
			}
		})
	}

	t.Log("AdminTokenAuth: correctly rejects malformed Authorization headers")
}

// Tests that requests with wrong token are rejected
func TestAdminTokenAuth_WrongToken(t *testing.T) {
	cleanup := setupTestAdminToken(t, "correct-secret-token")
	defer cleanup()

	protectedHandler := restapi.AdminTokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for wrong token, got %d", rr.Code)
	}

	t.Log("AdminTokenAuth: correctly rejects wrong tokens")
}

// Tests that requests with correct token are allowed
func TestAdminTokenAuth_ValidToken(t *testing.T) {
	cleanup := setupTestAdminToken(t, "valid-admin-token")
	defer cleanup()

	protectedHandler := restapi.AdminTokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))

	req := httptest.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer valid-admin-token")
	rr := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid token, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	t.Log("AdminTokenAuth: correctly allows valid tokens")
}

// Tests that empty admin token in config rejects all requests
func TestAdminTokenAuth_EmptyConfigToken(t *testing.T) {
	cleanup := setupTestAdminToken(t, "") // Empty token in config
	defer cleanup()

	protectedHandler := restapi.AdminTokenAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rr := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when config token is empty, got %d", rr.Code)
	}

	t.Log("AdminTokenAuth: correctly rejects all tokens when config token is empty")
}

// Tests that admin endpoints are protected when accessed through the middleware
func TestAdminEndpoints_RequireAuth(t *testing.T) {
	cleanup := setupTestAdminToken(t, "admin-secret")
	defer cleanup()

	// Define admin endpoints and their handlers
	adminEndpoints := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
		body    interface{}
	}{
		{
			name:    "Ban Player",
			method:  "POST",
			path:    "/admin/ban",
			handler: restapi.BanPlayerHandler,
			body:    restapi.BanPlayerRequest{Username: "test", Duration: "1h"},
		},
		{
			name:    "Unban Player",
			method:  "POST",
			path:    "/admin/unban",
			handler: restapi.UnbanPlayerHandler,
			body:    restapi.UnbanPlayerRequest{Username: "test"},
		},
		{
			name:    "Create Battle",
			method:  "POST",
			path:    "/admin/battles",
			handler: restapi.CreateBattleHandler,
			body:    restapi.CreateBattleRequest{Title: "Test", SongIDs: []int{1}, ExpiresAt: time.Now().Add(time.Hour)},
		},
		{
			name:    "Delete Battle",
			method:  "DELETE",
			path:    "/admin/battles",
			handler: restapi.DeleteBattleHandler,
			body:    restapi.DeleteBattleRequest{BattleID: 1},
		},
	}

	for _, ep := range adminEndpoints {
		t.Run(ep.name+" without auth", func(t *testing.T) {
			// Wrap handler with auth middleware
			protectedHandler := restapi.AdminTokenAuth(ep.handler)

			// Make request without auth
			rr := makeAuthRequest(t, ep.method, ep.path, ep.body, protectedHandler, "")

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("%s: expected 500 without auth, got %d", ep.name, rr.Code)
			}
		})

		t.Run(ep.name+" with wrong auth", func(t *testing.T) {
			protectedHandler := restapi.AdminTokenAuth(ep.handler)

			rr := makeAuthRequest(t, ep.method, ep.path, ep.body, protectedHandler, "wrong-token")

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("%s: expected 500 with wrong auth, got %d", ep.name, rr.Code)
			}
		})

		t.Run(ep.name+" with valid auth", func(t *testing.T) {
			protectedHandler := restapi.AdminTokenAuth(ep.handler)

			rr := makeAuthRequest(t, ep.method, ep.path, ep.body, protectedHandler, "admin-secret")

			// Should NOT be 500 (auth error) - might be 400/404 due to validation, but auth passed
			if rr.Code == http.StatusInternalServerError {
				var response map[string]string
				json.NewDecoder(rr.Body).Decode(&response)
				if response["error"] == "Could not verify authorization" {
					t.Errorf("%s: auth should have passed with valid token", ep.name)
				}
			}
		})
	}

	t.Log("AdminEndpoints: all admin endpoints properly require authentication")
}
