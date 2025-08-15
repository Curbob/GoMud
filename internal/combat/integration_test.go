package combat_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// TestCombatModuleSwitching tests switching between combat systems at runtime
func TestCombatModuleSwitching(t *testing.T) {
	// This test demonstrates the module switching capability
	// In a real scenario, the modules would be registered during initialization

	// Test initial state - might have a system from other tests
	initialSystem := combat.GetActiveCombatSystem()
	initialName := combat.GetActiveCombatSystemName()

	// If we have existing systems registered from init(), test switching
	systems := combat.ListCombatSystems()
	if len(systems) < 2 {
		t.Skip("Not enough combat systems registered to test switching")
	}

	// Switch between available systems
	firstSystem := systems[0]
	err := combat.SetActiveCombatSystem(firstSystem)
	require.NoError(t, err, "Failed to set %s combat", firstSystem)

	assert.Equal(t, firstSystem, combat.GetActiveCombatSystemName(),
		"Expected %s system, got %s", firstSystem, combat.GetActiveCombatSystemName())

	// Switch to another system if available
	if len(systems) > 1 {
		secondSystem := systems[1]
		err = combat.SetActiveCombatSystem(secondSystem)
		require.NoError(t, err, "Failed to switch to %s combat", secondSystem)

		assert.Equal(t, secondSystem, combat.GetActiveCombatSystemName(),
			"Expected %s system after switch", secondSystem)
	}

	// Restore initial state if there was one
	if initialSystem != nil && initialName != "" {
		combat.SetActiveCombatSystem(initialName)
	}
}

// TestCombatDelegation tests that attacks are properly delegated
func TestCombatDelegation(t *testing.T) {
	// Ensure we have at least one combat system
	systems := combat.ListCombatSystems()
	if len(systems) == 0 {
		t.Skip("No combat systems registered")
	}

	// Use the first available system
	err := combat.SetActiveCombatSystem(systems[0])
	require.NoError(t, err, "Failed to set combat system")

	// Create test combatants
	user := createTestUserForIntegration(1, "TestPlayer", 100)
	mob := createTestMobForIntegration(1, "TestMob", 50)

	// Record initial health
	initialMobHealth := mob.Character.Health

	// Perform attack through delegator
	result := combat.AttackPlayerVsMob(user, mob)

	// Verify we got a result
	if result.Hit {
		assert.NotZero(t, result.DamageToTarget, "Hit reported but no damage")
	}

	// If hit, verify health changed
	if result.Hit && result.DamageToTarget > 0 {
		expectedHealth := initialMobHealth - result.DamageToTarget
		assert.Equal(t, expectedHealth, mob.Character.Health,
			"Mob health not updated correctly")
	}

	// Test messages were generated
	if result.Hit {
		assert.NotEmpty(t, result.MessagesToSource, "No messages to attacker on hit")
		assert.NotEmpty(t, result.MessagesToTarget, "No messages to target on hit")
	}
}

// TestMultipleCombatRounds tests multiple rounds of combat
func TestMultipleCombatRounds(t *testing.T) {
	// Ensure we have a combat system
	if len(combat.ListCombatSystems()) == 0 {
		t.Skip("No combat systems registered")
	}

	// Create combatants
	user := createTestUserForIntegration(2, "Fighter", 150)
	mob := createTestMobForIntegration(2, "Orc", 100)

	// Track damage across rounds
	totalDamage := 0
	hits := 0
	rounds := 10

	for i := 0; i < rounds; i++ {
		result := combat.AttackPlayerVsMob(user, mob)

		if result.Hit {
			hits++
			totalDamage += result.DamageToTarget
		}

		// Stop if mob is dead
		if mob.Character.Health <= 0 {
			break
		}
	}

	// Verify some attacks hit
	assert.NotZero(t, hits, "No hits in %d rounds", rounds)

	// Verify total damage matches health reduction
	expectedHealth := 100 - totalDamage
	if expectedHealth < 0 {
		expectedHealth = 0
	}
	assert.Equal(t, expectedHealth, mob.Character.Health,
		"Health mismatch after %d rounds (total damage: %d)", rounds, totalDamage)
}

// TestCombatWithNoActiveSystem tests fallback behavior
func TestCombatWithNoActiveSystem(t *testing.T) {
	// Skip this test as it requires full game initialization for the fallback combat
	// The fallback combat system uses races.GetRace() which requires data files to be loaded
	t.Skip("Fallback combat requires full game initialization")
}

// TestCombatTimers tests timer functionality if available
func TestCombatTimers(t *testing.T) {
	system := combat.GetActiveCombatSystem()
	if system == nil {
		t.Skip("No active combat system")
	}

	timer := system.GetTimer()
	if timer == nil {
		t.Skip("Active system has no timer")
	}

	// Test start/stop
	err := timer.Start()
	assert.NoError(t, err, "Failed to start timer")

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	err = timer.Stop()
	assert.NoError(t, err, "Failed to stop timer")

	// Test double stop (should be safe)
	err = timer.Stop()
	assert.NoError(t, err, "Double stop should not cause error")
}

// TestCombatSystemPersistence tests that damage persists across system switches
func TestCombatSystemPersistence(t *testing.T) {
	systems := combat.ListCombatSystems()
	if len(systems) < 2 {
		t.Skip("Need at least 2 combat systems for this test")
	}

	// Create combatants
	user := createTestUserForIntegration(4, "Warrior", 200)
	mob := createTestMobForIntegration(4, "Dragon", 300)
	initialMobHealth := mob.Character.Health

	// Set first system
	err := combat.SetActiveCombatSystem(systems[0])
	require.NoError(t, err)

	// Attack with first system
	result1 := combat.AttackPlayerVsMob(user, mob)
	damageFromFirst := 0
	if result1.Hit {
		damageFromFirst = result1.DamageToTarget
	}

	// Switch to second system
	err = combat.SetActiveCombatSystem(systems[1])
	require.NoError(t, err)

	// Attack with second system
	result2 := combat.AttackPlayerVsMob(user, mob)
	damageFromSecond := 0
	if result2.Hit {
		damageFromSecond = result2.DamageToTarget
	}

	// Verify total damage
	totalDamage := damageFromFirst + damageFromSecond
	expectedHealth := initialMobHealth - totalDamage
	assert.Equal(t, expectedHealth, mob.Character.Health,
		"Mob health should reflect damage from both combat systems")
}

// Helper functions

func createTestUserForIntegration(id int, name string, health int) *users.UserRecord {
	user := &users.UserRecord{
		UserId: id,
		Character: &characters.Character{
			RaceId: 1, // Set a valid race ID to avoid nil pointer
			Stats: stats.Statistics{
				Strength: stats.StatInfo{Value: 15},
				Speed:    stats.StatInfo{Value: 12},
				Vitality: stats.StatInfo{Value: 10},
			},
			HealthMax: stats.StatInfo{
				Base:  health,
				Value: health,
			},
			Equipment: characters.Worn{},
		},
	}
	user.Character.Name = name
	user.Character.Health = health
	user.Character.Level = 10

	return user
}

func createTestMobForIntegration(instanceId int, name string, health int) *mobs.Mob {
	mob := &mobs.Mob{
		InstanceId: instanceId,
		Character: characters.Character{
			RaceId: 1, // Set a valid race ID to avoid nil pointer
			Stats: stats.Statistics{
				Strength: stats.StatInfo{Value: 12},
				Speed:    stats.StatInfo{Value: 10},
				Vitality: stats.StatInfo{Value: 8},
			},
			HealthMax: stats.StatInfo{
				Base:  health,
				Value: health,
			},
			Equipment: characters.Worn{},
		},
	}
	mob.Character.Name = name
	mob.Character.Health = health
	mob.Character.Level = 8

	return mob
}
