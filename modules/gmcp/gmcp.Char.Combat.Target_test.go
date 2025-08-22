package gmcp

import (
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func init() {
	// Initialize logger for tests
	mudlog.SetupLogger(nil, "HIGH", "", true)
}

// TestValidateUserAndLock tests the race condition fix
func TestValidateUserAndLock(t *testing.T) {
	// We can't easily test validateUserAndLock directly because it depends on
	// users.GetByUserId and isGMCPEnabled which we can't mock easily.
	// Instead, we'll test the cleanup behavior by manipulating the maps directly.

	// Test cleanup behavior
	targetMutexNew.Lock()
	userTargetsNew[888] = &TargetInfo{Id: 123, Name: "Test Mob"}
	targetMutexNew.Unlock()

	// When validateUserAndLock can't find a user, it should clean up
	// We'll simulate this by calling the cleanup function directly
	cleanupCombatTargetNew(888)

	// Check cleanup happened
	targetMutexNew.RLock()
	if _, exists := userTargetsNew[888]; exists {
		t.Error("Expected userTargetsNew to be cleaned up")
	}
	targetMutexNew.RUnlock()
}

// TestCombatTargetCleanup tests that cleanup functions work correctly
func TestCombatTargetCleanup(t *testing.T) {
	// Set up test data
	targetMutexNew.Lock()
	userTargetsNew[100] = &TargetInfo{Id: 200, Name: "Test Target", LastHP: 50}
	targetMutexNew.Unlock()

	// Call cleanup
	cleanupCombatTargetNew(100)

	// Verify cleanup
	targetMutexNew.RLock()
	if _, exists := userTargetsNew[100]; exists {
		t.Error("Expected userTargetsNew[100] to be deleted")
	}
	targetMutexNew.RUnlock()
}

// TestConcurrentAccess tests that the maps are safe for concurrent access
func TestConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup

	// Simulate multiple goroutines accessing the maps
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(userId int) {
			defer wg.Done()

			// Write
			targetMutexNew.Lock()
			userTargetsNew[userId] = &TargetInfo{Id: userId * 10, Name: "Test", LastHP: userId * 100}
			targetMutexNew.Unlock()

			// Read
			targetMutexNew.RLock()
			_ = userTargetsNew[userId]
			targetMutexNew.RUnlock()

			// Cleanup
			cleanupCombatTargetNew(userId)
		}(i)
	}

	wg.Wait()

	// Verify all cleaned up
	targetMutexNew.RLock()
	if len(userTargetsNew) != 0 {
		t.Errorf("Expected empty userTargetsNew, got %d entries", len(userTargetsNew))
	}
	targetMutexNew.RUnlock()
}

// TestGMCPCombatTargetUpdateType tests the event type
func TestGMCPCombatTargetUpdateType(t *testing.T) {
	update := GMCPCombatTargetUpdate{
		UserId:          1,
		TargetName:      "TestMob",
		TargetHpCurrent: 50,
		TargetHpMax:     100,
	}

	if update.Type() != "GMCPCombatTargetUpdate" {
		t.Errorf("Expected Type() to return GMCPCombatTargetUpdate, got %s", update.Type())
	}
}
