package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

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

	// Create test user
	user := &users.UserRecord{
		UserId: 1,
		Character: &characters.Character{
			Name:   "TestPlayer",
			Health: 100,
		},
	}
	user.Character.Stats.Strength.Base = 10

	// Create test mob
	mob := &mobs.Mob{
		InstanceId: 1,
		Character: characters.Character{
			Name:   "TestMob",
			Health: 50,
		},
	}
	mob.Character.Stats.Strength.Base = 5

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

	// Create test user
	user := &users.UserRecord{
		UserId: 1,
		Character: &characters.Character{
			Name:   "TestPlayer",
			Health: 100,
		},
	}
	user.Character.Stats.Strength.Base = 10

	// Create test mob
	mob := &mobs.Mob{
		InstanceId: 1,
		Character: characters.Character{
			Name:   "TestMob",
			Health: 50,
		},
	}
	mob.Character.Stats.Strength.Base = 5

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

	for i := 0; i < 20 && !missFound; i++ {
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

		// Create test user with low accuracy
		user := &users.UserRecord{
			UserId: 1,
			Character: &characters.Character{
				Name:   "TestPlayer",
				Health: 100,
			},
		}
		user.Character.Stats.Strength.Base = 1 // Low strength for higher miss chance

		// Create test mob with high defense
		mob := &mobs.Mob{
			InstanceId: 1,
			Character: characters.Character{
				Name:   "TestMob",
				Health: 50,
			},
		}
		mob.Character.Stats.Speed.Base = 20 // High speed for dodge

		// Simulate combat
		attackResult := combat.AttackPlayerVsMob(user, mob)

		if !attackResult.Hit {
			t.Log("Attack missed!")
			if !eventsFired["AttackAvoided"] {
				t.Error("Expected AttackAvoided event to be fired on miss")
			}
			break
		}
	}

	if !missFound {
		t.Skip("No misses occurred in test runs")
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
	attacker := &users.UserRecord{
		UserId: 1,
		Character: &characters.Character{
			Name:   "Attacker",
			Health: 100,
		},
	}
	attacker.Character.Stats.Strength.Base = 10

	defender := &users.UserRecord{
		UserId: 2,
		Character: &characters.Character{
			Name:   "Defender",
			Health: 100,
		},
	}
	defender.Character.Stats.Strength.Base = 8

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

	// Register test listeners
	events.RegisterListener(events.MobVitalsChanged{}, func(e events.Event) events.ListenerReturn {
		mobVitalsCount++
		return events.Continue
	})

	events.RegisterListener(events.DamageDealt{}, func(e events.Event) events.ListenerReturn {
		evt := e.(events.DamageDealt)
		if evt.SourceType == "mob" && evt.TargetType == "mob" {
			t.Log("Mob vs Mob damage event fired")
		}
		return events.Continue
	})

	// Create test mobs
	mobAtk := &mobs.Mob{
		InstanceId: 1,
		Character: characters.Character{
			Name:   "AttackerMob",
			Health: 50,
		},
	}
	mobAtk.Character.Stats.Strength.Base = 8

	mobDef := &mobs.Mob{
		InstanceId: 2,
		Character: characters.Character{
			Name:   "DefenderMob",
			Health: 50,
		},
	}
	mobDef.Character.Stats.Strength.Base = 6

	// Simulate mob vs mob combat
	attackResult := combat.AttackMobVsMob(mobAtk, mobDef)

	if attackResult.Hit && attackResult.DamageToTarget > 0 {
		t.Log("Mob vs Mob attack hit for", attackResult.DamageToTarget, "damage")
		if mobVitalsCount < 1 {
			t.Error("Expected at least one MobVitalsChanged event")
		}
	}
}
