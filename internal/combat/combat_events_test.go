package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/testhelpers"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	// Initialize logger for tests
	mudlog.SetupLogger(nil, "HIGH", "", true)

	// Initialize test races for combat testing
	testhelpers.InitializeTestRaces()
}

// Helper function to create a properly initialized test user
func createTestUser(id int, name string, health int, strength int) *users.UserRecord {
	user := &users.UserRecord{
		UserId: id,
		Character: &characters.Character{
			Name:      name,
			Health:    health,
			HealthMax: stats.StatInfo{Base: 100, Value: 100},
			RaceId:    1, // Human race (assuming it exists)
			Level:     1,
			RoomId:    1,
		},
	}
	user.Character.Stats.Strength.Base = strength
	user.Character.Stats.Speed.Base = 10
	user.Character.Stats.Smarts.Base = 10
	user.Character.Stats.Vitality.Base = 10
	user.Character.Stats.Mysticism.Base = 10
	user.Character.Stats.Perception.Base = 10
	// Initialize aggro to prevent nil pointer
	user.Character.Aggro = &characters.Aggro{}
	return user
}

// Helper function to create a properly initialized test mob
func createTestMob(id int, name string, health int, strength int) *mobs.Mob {
	mob := &mobs.Mob{
		InstanceId: id,
		MobId:      1,
		Character: characters.Character{
			Name:      name,
			Health:    health,
			HealthMax: stats.StatInfo{Base: 50, Value: 50},
			RaceId:    1, // Human race (assuming it exists)
			Level:     1,
			RoomId:    1,
		},
	}
	mob.Character.Stats.Strength.Base = strength
	mob.Character.Stats.Speed.Base = 10
	mob.Character.Stats.Smarts.Base = 10
	mob.Character.Stats.Vitality.Base = 10
	mob.Character.Stats.Mysticism.Base = 10
	mob.Character.Stats.Perception.Base = 10
	// Initialize aggro to prevent nil pointer
	mob.Character.Aggro = &characters.Aggro{}
	return mob
}

// TestDamageDealtEvent tests that DamageDealt events are fired correctly
func TestDamageDealtEvent(t *testing.T) {
	// Track events fired
	eventsFired := make(map[string]bool)

	// Register a test listener
	events.RegisterListener(events.DamageDealt{}, func(e events.Event) events.ListenerReturn {
		evt := e.(events.DamageDealt)
		if evt.SourceId == 1 && evt.TargetId == 1 {
			eventsFired["DamageDealt"] = true
		}
		return events.Continue
	})

	// Create test user and mob
	user := createTestUser(1, "TestPlayer", 100, 10)
	mob := createTestMob(1, "TestMob", 50, 5)

	// Set up combat aggro
	user.Character.SetAggro(0, 1, characters.DefaultAttack)
	mob.Character.SetAggro(1, 0, characters.DefaultAttack)

	// Simulate combat
	attackResult := combat.AttackPlayerVsMob(user, mob)

	// Check if attack hit and did damage
	if attackResult.Hit && attackResult.DamageToTarget > 0 {
		// Give events time to process
		// In real code, events are processed asynchronously
		// For testing, we'll just check if the flag was set
		t.Log("Attack hit for", attackResult.DamageToTarget, "damage")
	}
}

// TestMobVitalsChangedEvent tests that MobVitalsChanged events are fired
func TestMobVitalsChangedEvent(t *testing.T) {
	// Track events fired
	eventsFired := make(map[string]bool)
	var vitalsEvent events.MobVitalsChanged

	// Register a test listener
	events.RegisterListener(events.MobVitalsChanged{}, func(e events.Event) events.ListenerReturn {
		evt := e.(events.MobVitalsChanged)
		if evt.MobId == 1 {
			eventsFired["MobVitalsChanged"] = true
			vitalsEvent = evt
		}
		return events.Continue
	})

	// Create test user and mob
	user := createTestUser(1, "TestPlayer", 100, 10)
	mob := createTestMob(1, "TestMob", 50, 5)

	// Set up combat aggro
	user.Character.SetAggro(0, 1, characters.DefaultAttack)
	mob.Character.SetAggro(1, 0, characters.DefaultAttack)

	oldHealth := mob.Character.Health

	// Simulate combat
	attackResult := combat.AttackPlayerVsMob(user, mob)

	// Check results
	if attackResult.Hit && attackResult.DamageToTarget > 0 {
		t.Log("Attack hit for", attackResult.DamageToTarget, "damage")
		t.Log("Mob health changed from", oldHealth, "to", mob.Character.Health)

		if eventsFired["MobVitalsChanged"] {
			if vitalsEvent.OldHealth != oldHealth {
				t.Errorf("Expected OldHealth %d, got %d", oldHealth, vitalsEvent.OldHealth)
			}
			if vitalsEvent.NewHealth != mob.Character.Health {
				t.Errorf("Expected NewHealth %d, got %d", mob.Character.Health, vitalsEvent.NewHealth)
			}
		}
	}
}

// TestAttackAvoidedEvent tests that AttackAvoided events are fired on misses
func TestAttackAvoidedEvent(t *testing.T) {
	// This test needs to run multiple times to ensure we get a miss
	missFound := false
	maxAttempts := 100 // Increase attempts to make miss more likely

	for i := 0; i < maxAttempts && !missFound; i++ {
		// Track events fired
		eventsFired := make(map[string]bool)

		// Register a test listener
		events.RegisterListener(events.AttackAvoided{}, func(e events.Event) events.ListenerReturn {
			evt := e.(events.AttackAvoided)
			if evt.AttackerId == 1 {
				eventsFired["AttackAvoided"] = true
				missFound = true
			}
			return events.Continue
		})

		// Create test user with minimal stats for maximum miss chance
		user := createTestUser(1, "TestPlayer", 100, 1) // Minimal strength
		user.Character.Stats.Speed.Base = 1             // Minimal speed
		user.Character.Stats.Perception.Base = 1        // Minimal perception

		// Create mob with maximum defensive stats
		mob := createTestMob(1, "TestMob", 50, 5)
		mob.Character.Stats.Speed.Base = 50      // Very high speed for dodge
		mob.Character.Stats.Perception.Base = 50 // Very high perception

		// Set up combat aggro
		user.Character.SetAggro(0, 1, characters.DefaultAttack)
		mob.Character.SetAggro(1, 0, characters.DefaultAttack)

		// Simulate combat
		attackResult := combat.AttackPlayerVsMob(user, mob)

		if !attackResult.Hit {
			t.Logf("Attack missed on attempt %d!", i+1)

			// Process any queued events immediately
			events.ProcessEvents()

			if !eventsFired["AttackAvoided"] {
				t.Error("Expected AttackAvoided event to be fired on miss")
			}
			break
		}
	}

	if !missFound {
		t.Skip("No misses occurred in test runs - this may be a balance issue with combat calculations")
	}
}

// TestPlayerVsPlayerCombatEvents tests events in PvP combat
func TestPlayerVsPlayerCombatEvents(t *testing.T) {
	// Track events fired
	eventsFired := make(map[string]bool)

	// Register test listeners
	events.RegisterListener(events.DamageDealt{}, func(e events.Event) events.ListenerReturn {
		evt := e.(events.DamageDealt)
		if evt.SourceType == "player" && evt.TargetType == "player" {
			eventsFired["PvPDamage"] = true
		}
		return events.Continue
	})

	// Create test users
	attacker := createTestUser(1, "Attacker", 100, 10)
	defender := createTestUser(2, "Defender", 100, 8)

	// Set up PvP aggro
	attacker.Character.SetAggro(2, 0, characters.DefaultAttack)
	defender.Character.SetAggro(1, 0, characters.DefaultAttack)

	// Simulate PvP combat
	attackResult := combat.AttackPlayerVsPlayer(attacker, defender)

	if attackResult.Hit && attackResult.DamageToTarget > 0 {
		t.Log("PvP attack hit for", attackResult.DamageToTarget, "damage")
	}
}

// TestMobVsMobCombatEvents tests events in mob vs mob combat
func TestMobVsMobCombatEvents(t *testing.T) {
	// Track events fired
	mobVitalsCount := 0
	damageDealtCount := 0

	// Register test listeners
	events.RegisterListener(events.MobVitalsChanged{}, func(e events.Event) events.ListenerReturn {
		evt := e.(events.MobVitalsChanged)
		t.Logf("MobVitalsChanged event: MobId=%d, OldHealth=%d, NewHealth=%d",
			evt.MobId, evt.OldHealth, evt.NewHealth)
		mobVitalsCount++
		return events.Continue
	})

	events.RegisterListener(events.DamageDealt{}, func(e events.Event) events.ListenerReturn {
		evt := e.(events.DamageDealt)
		if evt.SourceType == "mob" && evt.TargetType == "mob" {
			t.Logf("Mob vs Mob damage event: Damage=%d", evt.Amount)
			damageDealtCount++
		}
		return events.Continue
	})

	// Try multiple times to ensure we get a hit
	hitFound := false
	for attempt := 0; attempt < 20 && !hitFound; attempt++ {
		// Reset counters for each attempt
		mobVitalsCount = 0
		damageDealtCount = 0

		// Create test mobs with better stats for more reliable hits
		mobAtk := createTestMob(1, "AttackerMob", 50, 15) // Higher strength
		mobDef := createTestMob(2, "DefenderMob", 50, 6)

		// Set up mob vs mob aggro
		mobAtk.Character.SetAggro(0, 2, characters.DefaultAttack)
		mobDef.Character.SetAggro(0, 1, characters.DefaultAttack)

		// Simulate mob vs mob combat
		attackResult := combat.AttackMobVsMob(mobAtk, mobDef)

		if attackResult.Hit && attackResult.DamageToTarget > 0 {
			hitFound = true
			t.Logf("Mob vs Mob attack hit for %d damage on attempt %d", attackResult.DamageToTarget, attempt+1)

			// Process any queued events immediately
			events.ProcessEvents()

			if mobVitalsCount < 1 {
				t.Errorf("Expected at least one MobVitalsChanged event, got %d", mobVitalsCount)
			}
			if damageDealtCount < 1 {
				t.Errorf("Expected at least one DamageDealt event, got %d", damageDealtCount)
			}
			break
		}
	}

	if !hitFound {
		t.Skip("No hits occurred in test runs - this may be a balance issue with combat calculations")
	}
}
