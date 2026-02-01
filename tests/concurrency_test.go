package tests

import (
	"context"
	"rb3server/database"
	"sync"
	"testing"
)

// Tests that concurrent calls to GetNextPID return unique, sequential IDs
func TestGetNextPID_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 10
	callsPerGoroutine := 50
	totalCalls := numGoroutines * callsPerGoroutine

	var wg sync.WaitGroup
	pidChan := make(chan int, totalCalls)
	errChan := make(chan error, totalCalls)

	// Launch multiple goroutines that all try to get PIDs simultaneously
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				pid, err := database.GetNextPID(ctx)
				if err != nil {
					errChan <- err
					return
				}
				pidChan <- pid
			}
		}()
	}

	wg.Wait()
	close(pidChan)
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Fatalf("Error during concurrent PID generation: %v", err)
	}

	// Collect all PIDs and verify uniqueness
	pids := make(map[int]bool)
	for pid := range pidChan {
		if pids[pid] {
			t.Errorf("Duplicate PID generated: %d", pid)
		}
		pids[pid] = true
	}

	if len(pids) != totalCalls {
		t.Errorf("Expected %d unique PIDs, got %d", totalCalls, len(pids))
	}

	t.Logf("Successfully generated %d unique PIDs concurrently", len(pids))
}

// Tests that concurrent calls to GetNextBandID return unique IDs
func TestGetNextBandID_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 10
	callsPerGoroutine := 50
	totalCalls := numGoroutines * callsPerGoroutine

	var wg sync.WaitGroup
	idChan := make(chan int, totalCalls)
	errChan := make(chan error, totalCalls)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				id, err := database.GetNextBandID(ctx)
				if err != nil {
					errChan <- err
					return
				}
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)
	close(errChan)

	for err := range errChan {
		t.Fatalf("Error during concurrent band ID generation: %v", err)
	}

	ids := make(map[int]bool)
	for id := range idChan {
		if ids[id] {
			t.Errorf("Duplicate band ID generated: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != totalCalls {
		t.Errorf("Expected %d unique band IDs, got %d", totalCalls, len(ids))
	}

	t.Logf("Successfully generated %d unique band IDs concurrently", len(ids))
}

// Tests that concurrent calls to GetNextCharacterID return unique IDs
func TestGetNextCharacterID_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 10
	callsPerGoroutine := 50
	totalCalls := numGoroutines * callsPerGoroutine

	var wg sync.WaitGroup
	idChan := make(chan int, totalCalls)
	errChan := make(chan error, totalCalls)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				id, err := database.GetNextCharacterID(ctx)
				if err != nil {
					errChan <- err
					return
				}
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)
	close(errChan)

	for err := range errChan {
		t.Fatalf("Error during concurrent character ID generation: %v", err)
	}

	ids := make(map[int]bool)
	for id := range idChan {
		if ids[id] {
			t.Errorf("Duplicate character ID generated: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != totalCalls {
		t.Errorf("Expected %d unique character IDs, got %d", totalCalls, len(ids))
	}

	t.Logf("Successfully generated %d unique character IDs concurrently", len(ids))
}

// Tests that concurrent calls to GetNextSetlistID return unique IDs
func TestGetNextSetlistID_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 10
	callsPerGoroutine := 50
	totalCalls := numGoroutines * callsPerGoroutine

	var wg sync.WaitGroup
	idChan := make(chan int, totalCalls)
	errChan := make(chan error, totalCalls)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				id, err := database.GetNextSetlistID(ctx)
				if err != nil {
					errChan <- err
					return
				}
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)
	close(errChan)

	for err := range errChan {
		t.Fatalf("Error during concurrent setlist ID generation: %v", err)
	}

	ids := make(map[int]bool)
	for id := range idChan {
		if ids[id] {
			t.Errorf("Duplicate setlist ID generated: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != totalCalls {
		t.Errorf("Expected %d unique setlist IDs, got %d", totalCalls, len(ids))
	}

	t.Logf("Successfully generated %d unique setlist IDs concurrently", len(ids))
}

// Tests that concurrent calls to GetNextMachineID return unique IDs
func TestGetNextMachineID_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 10
	callsPerGoroutine := 50
	totalCalls := numGoroutines * callsPerGoroutine

	var wg sync.WaitGroup
	idChan := make(chan int, totalCalls)
	errChan := make(chan error, totalCalls)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				id, err := database.GetNextMachineID(ctx)
				if err != nil {
					errChan <- err
					return
				}
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)
	close(errChan)

	for err := range errChan {
		t.Fatalf("Error during concurrent machine ID generation: %v", err)
	}

	ids := make(map[int]bool)
	for id := range idChan {
		if ids[id] {
			t.Errorf("Duplicate machine ID generated: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != totalCalls {
		t.Errorf("Expected %d unique machine IDs, got %d", totalCalls, len(ids))
	}

	t.Logf("Successfully generated %d unique machine IDs concurrently", len(ids))
}

// Tests concurrent username lookups don't cause issues
func TestGetUsernameForPID_Concurrent(t *testing.T) {
	numGoroutines := 20
	lookupsPerGoroutine := 100

	var wg sync.WaitGroup
	errChan := make(chan string, numGoroutines*lookupsPerGoroutine)

	// All goroutines look up the same PIDs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < lookupsPerGoroutine; j++ {
				// Lookup existing users
				username := database.GetUsernameForPID(500)
				if username != "testuser" {
					errChan <- "Expected 'testuser' for PID 500, got " + username
				}

				username = database.GetUsernameForPID(501)
				if username != "testuser2" {
					errChan <- "Expected 'testuser2' for PID 501, got " + username
				}

				// Lookup non-existent user
				username = database.GetUsernameForPID(99999)
				if username != "Player" {
					errChan <- "Expected 'Player' for missing PID, got " + username
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	errorCount := 0
	for errMsg := range errChan {
		t.Error(errMsg)
		errorCount++
		if errorCount >= 10 {
			t.Fatal("Too many errors, stopping")
		}
	}

	t.Logf("Completed %d concurrent username lookups", numGoroutines*lookupsPerGoroutine*3)
}

// Tests concurrent config cache access
func TestGetCachedConfig_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 20
	accessesPerGoroutine := 100

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*accessesPerGoroutine)

	// Mix of cache reads and invalidations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < accessesPerGoroutine; j++ {
				config, err := database.GetCachedConfig(ctx)
				if err != nil {
					errChan <- err
					continue
				}
				if config == nil {
					t.Error("Got nil config")
				}

				// Every 10th call, invalidate the cache to test cache refresh under load
				if j%10 == 0 && id%2 == 0 {
					database.InvalidateConfigCache()
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Error during concurrent config access: %v", err)
	}

	t.Logf("Completed %d concurrent config accesses with periodic invalidations", numGoroutines*accessesPerGoroutine)
}

// Tests concurrent friend checks
func TestIsPIDAFriendOfPID_Concurrent(t *testing.T) {
	numGoroutines := 20
	checksPerGoroutine := 100

	var wg sync.WaitGroup
	errChan := make(chan string, numGoroutines*checksPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < checksPerGoroutine; j++ {
				// Check a valid friendship
				ok, err := database.IsPIDAFriendOfPID(500, 501)
				if err != nil {
					errChan <- "Error checking friendship: " + err.Error()
					continue
				}
				if !ok {
					errChan <- "Expected 501 to be friend of 500"
				}

				// Check a non-friendship
				ok, err = database.IsPIDAFriendOfPID(500, 999)
				if err != nil {
					errChan <- "Error checking non-friendship: " + err.Error()
					continue
				}
				if ok {
					errChan <- "Expected 999 to NOT be friend of 500"
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	errorCount := 0
	for errMsg := range errChan {
		t.Error(errMsg)
		errorCount++
		if errorCount >= 10 {
			t.Fatal("Too many errors, stopping")
		}
	}

	t.Logf("Completed %d concurrent friend checks", numGoroutines*checksPerGoroutine*2)
}

// Tests concurrent group membership checks
func TestIsPIDInGroup_Concurrent(t *testing.T) {
	numGoroutines := 20
	checksPerGoroutine := 100

	var wg sync.WaitGroup
	errChan := make(chan string, numGoroutines*checksPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < checksPerGoroutine; j++ {
				// Check valid group membership
				ok := database.IsPIDInGroup(500, "admin")
				if !ok {
					errChan <- "Expected PID 500 to be in 'admin' group"
				}

				// Check non-membership
				ok = database.IsPIDInGroup(501, "admin")
				if ok {
					errChan <- "Expected PID 501 to NOT be in 'admin' group"
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	errorCount := 0
	for errMsg := range errChan {
		t.Error(errMsg)
		errorCount++
		if errorCount >= 10 {
			t.Fatal("Too many errors, stopping")
		}
	}

	t.Logf("Completed %d concurrent group checks", numGoroutines*checksPerGoroutine*2)
}

// Tests concurrent batch username lookups
func TestGetUsernamesByPIDs_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 20
	lookupsPerGoroutine := 50

	var wg sync.WaitGroup
	errChan := make(chan string, numGoroutines*lookupsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < lookupsPerGoroutine; j++ {
				pids := []int{500, 501, 502}
				usernames, err := database.GetUsernamesByPIDs(ctx, database.GocentralDatabase, pids)
				if err != nil {
					errChan <- "Error in batch lookup: " + err.Error()
					continue
				}

				if len(usernames) != 3 {
					errChan <- "Expected 3 usernames in batch result"
					continue
				}

				if usernames[500] != "testuser" {
					errChan <- "Wrong username for PID 500"
				}
				if usernames[501] != "testuser2" {
					errChan <- "Wrong username for PID 501"
				}
				if usernames[502] != "testuser3" {
					errChan <- "Wrong username for PID 502"
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	errorCount := 0
	for errMsg := range errChan {
		t.Error(errMsg)
		errorCount++
		if errorCount >= 10 {
			t.Fatal("Too many errors, stopping")
		}
	}

	t.Logf("Completed %d concurrent batch username lookups", numGoroutines*lookupsPerGoroutine)
}

// Tests concurrent band name lookups
func TestGetBandNamesByOwnerPIDs_Concurrent(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 20
	lookupsPerGoroutine := 50

	var wg sync.WaitGroup
	errChan := make(chan string, numGoroutines*lookupsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < lookupsPerGoroutine; j++ {
				ownerPIDs := []int{500, 502}
				bandNames, err := database.GetBandNamesByOwnerPIDs(ctx, database.GocentralDatabase, ownerPIDs)
				if err != nil {
					errChan <- "Error in batch band lookup: " + err.Error()
					continue
				}

				if len(bandNames) != 2 {
					errChan <- "Expected 2 band names in batch result"
					continue
				}

				if bandNames[500] != "T. Wrecks the Test" {
					errChan <- "Wrong band name for owner PID 500"
				}
				if bandNames[502] != "The Testifiers" {
					errChan <- "Wrong band name for owner PID 502"
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	errorCount := 0
	for errMsg := range errChan {
		t.Error(errMsg)
		errorCount++
		if errorCount >= 10 {
			t.Fatal("Too many errors, stopping")
		}
	}

	t.Logf("Completed %d concurrent batch band name lookups", numGoroutines*lookupsPerGoroutine)
}

// Tests mixed concurrent operations simulating real server load
func TestMixedConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	numGoroutines := 10
	operationsPerGoroutine := 50

	var wg sync.WaitGroup
	errChan := make(chan string, numGoroutines*operationsPerGoroutine*5)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of different operations based on iteration
				switch j % 5 {
				case 0:
					// Get next PID
					_, err := database.GetNextPID(ctx)
					if err != nil {
						errChan <- "GetNextPID error: " + err.Error()
					}
				case 1:
					// Username lookup
					username := database.GetUsernameForPID(500)
					if username != "testuser" {
						errChan <- "Wrong username in mixed test"
					}
				case 2:
					// Friend check
					ok, err := database.IsPIDAFriendOfPID(500, 501)
					if err != nil || !ok {
						errChan <- "Friend check failed in mixed test"
					}
				case 3:
					// Config access
					config, err := database.GetCachedConfig(ctx)
					if err != nil || config == nil {
						errChan <- "Config access failed in mixed test"
					}
				case 4:
					// Batch lookup
					pids := []int{500, 501}
					usernames, err := database.GetUsernamesByPIDs(ctx, database.GocentralDatabase, pids)
					if err != nil || len(usernames) != 2 {
						errChan <- "Batch lookup failed in mixed test"
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	errorCount := 0
	for errMsg := range errChan {
		t.Error(errMsg)
		errorCount++
		if errorCount >= 10 {
			t.Fatal("Too many errors, stopping")
		}
	}

	t.Logf("Completed %d mixed concurrent operations", numGoroutines*operationsPerGoroutine)
}
