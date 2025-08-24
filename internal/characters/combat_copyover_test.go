package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func init() {
	// Initialize mudlog for tests
	mudlog.SetupLogger(nil, "ERROR", "", false)
}

func TestCombatStateSerialization(t *testing.T) {
	t.Run("Basic aggro preservation", func(t *testing.T) {
		// Create character with basic aggro
		char := &Character{
			Aggro: &Aggro{
				UserId:        123,
				MobInstanceId: 0,
				Type:          DefaultAttack,
				RoundsWaiting: 3,
				ExitName:      "north",
			},
			PlayerDamage:     map[int]int{123: 50, 456: 25},
			LastPlayerDamage: 1000,
		}

		// Save combat state
		state := char.SaveCombatState()
		if state == nil {
			t.Fatal("SaveCombatState returned nil")
		}

		// Verify saved state
		if len(state.Aggro) != 1 {
			t.Errorf("Expected 1 aggro entry, got %d", len(state.Aggro))
		}
		if state.Aggro[0].UserId != 123 {
			t.Errorf("Expected UserId 123, got %d", state.Aggro[0].UserId)
		}
		if state.Aggro[0].Type != "default" {
			t.Errorf("Expected type 'default', got %s", state.Aggro[0].Type)
		}
		if len(state.PlayerDamage) != 2 {
			t.Errorf("Expected 2 damage entries, got %d", len(state.PlayerDamage))
		}

		// Create new character and restore
		newChar := &Character{}
		err := newChar.RestoreCombatState(state)
		if err != nil {
			t.Fatalf("RestoreCombatState failed: %v", err)
		}

		// Verify restored state
		if newChar.Aggro == nil {
			t.Fatal("Aggro not restored")
		}
		if newChar.Aggro.UserId != 123 {
			t.Errorf("Expected restored UserId 123, got %d", newChar.Aggro.UserId)
		}
		if newChar.Aggro.Type != DefaultAttack {
			t.Errorf("Expected restored type DefaultAttack, got %v", newChar.Aggro.Type)
		}
		if newChar.Aggro.RoundsWaiting != 3 {
			t.Errorf("Expected restored RoundsWaiting 3, got %d", newChar.Aggro.RoundsWaiting)
		}
		if newChar.PlayerDamage[123] != 50 {
			t.Errorf("Expected damage 50 for user 123, got %d", newChar.PlayerDamage[123])
		}
	})

	t.Run("Mob aggro preservation", func(t *testing.T) {
		// Create character with mob aggro
		char := &Character{
			Aggro: &Aggro{
				UserId:        0,
				MobInstanceId: 999,
				Type:          Shooting,
				RoundsWaiting: 1,
				ExitName:      "east",
			},
		}

		// Save and restore
		state := char.SaveCombatState()
		newChar := &Character{}
		err := newChar.RestoreCombatState(state)
		if err != nil {
			t.Fatalf("RestoreCombatState failed: %v", err)
		}

		// Verify
		if newChar.Aggro.MobInstanceId != 999 {
			t.Errorf("Expected mob ID 999, got %d", newChar.Aggro.MobInstanceId)
		}
		if newChar.Aggro.Type != Shooting {
			t.Errorf("Expected type Shooting, got %v", newChar.Aggro.Type)
		}
		if newChar.Aggro.ExitName != "east" {
			t.Errorf("Expected exit 'east', got %s", newChar.Aggro.ExitName)
		}
	})

	t.Run("Spell casting state preservation", func(t *testing.T) {
		// Create character casting a spell
		char := &Character{
			Aggro: &Aggro{
				Type:          SpellCast,
				UserId:        456,
				RoundsWaiting: 2,
				SpellInfo: SpellAggroInfo{
					SpellId:              "fireball",
					SpellRest:            "all",
					TargetUserIds:        []int{789, 101},
					TargetMobInstanceIds: []int{202, 303},
				},
			},
		}

		// Save state
		state := char.SaveCombatState()
		if state == nil {
			t.Fatal("SaveCombatState returned nil")
		}

		// Verify spell casting state was saved
		if state.SpellCasting == nil {
			t.Fatal("SpellCasting not saved")
		}
		if state.SpellCasting.SpellId != "fireball" {
			t.Errorf("Expected spell ID 'fireball', got %s", state.SpellCasting.SpellId)
		}
		if state.SpellCasting.RoundsLeft != 2 {
			t.Errorf("Expected 2 rounds left, got %d", state.SpellCasting.RoundsLeft)
		}
		if len(state.SpellCasting.TargetUserIds) != 2 {
			t.Errorf("Expected 2 user targets, got %d", len(state.SpellCasting.TargetUserIds))
		}

		// Restore to new character
		newChar := &Character{}
		err := newChar.RestoreCombatState(state)
		if err != nil {
			t.Fatalf("RestoreCombatState failed: %v", err)
		}

		// Verify restored spell info
		if newChar.Aggro.Type != SpellCast {
			t.Errorf("Expected type SpellCast, got %v", newChar.Aggro.Type)
		}
		if newChar.Aggro.SpellInfo.SpellId != "fireball" {
			t.Errorf("Expected restored spell ID 'fireball', got %s", newChar.Aggro.SpellInfo.SpellId)
		}
		if len(newChar.Aggro.SpellInfo.TargetUserIds) != 2 {
			t.Errorf("Expected 2 restored user targets, got %d", len(newChar.Aggro.SpellInfo.TargetUserIds))
		}
		if newChar.Aggro.RoundsWaiting != 2 {
			t.Errorf("Expected restored rounds waiting 2, got %d", newChar.Aggro.RoundsWaiting)
		}
	})

	t.Run("Nil aggro handling", func(t *testing.T) {
		// Character with no aggro
		char := &Character{
			Aggro: nil,
		}

		// Save state should return nil
		state := char.SaveCombatState()
		if state != nil {
			t.Error("Expected nil state for character with no aggro")
		}

		// Restore nil state should not error
		newChar := &Character{}
		err := newChar.RestoreCombatState(nil)
		if err != nil {
			t.Errorf("RestoreCombatState with nil state failed: %v", err)
		}
		if newChar.Aggro != nil {
			t.Error("Expected nil aggro after restoring nil state")
		}
	})

	t.Run("Combat type conversion", func(t *testing.T) {
		testCases := []struct {
			name     string
			aggType  AggroType
			expected string
		}{
			{"DefaultAttack", DefaultAttack, "default"},
			{"Shooting", Shooting, "shooting"},
			{"BackStab", BackStab, "backstab"},
			{"SpellCast", SpellCast, "spellcast"},
			{"Flee", Flee, "flee"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				char := &Character{
					Aggro: &Aggro{
						Type:   tc.aggType,
						UserId: 1,
					},
				}

				state := char.SaveCombatState()
				if state.Aggro[0].Type != tc.expected {
					t.Errorf("Expected type string '%s', got '%s'", tc.expected, state.Aggro[0].Type)
				}

				// Verify it converts back
				newChar := &Character{}
				newChar.RestoreCombatState(state)
				if newChar.Aggro.Type != tc.aggType {
					t.Errorf("Expected restored type %v, got %v", tc.aggType, newChar.Aggro.Type)
				}
			})
		}
	})

	t.Run("Aggro target validation", func(t *testing.T) {
		// Create character with aggro
		char := &Character{
			Aggro: &Aggro{
				UserId: 999,
				Type:   DefaultAttack,
			},
		}

		// Test validation with non-existent user
		char.ValidateAggroTargets(
			func(userId int) bool {
				return userId != 999 // User 999 doesn't exist
			},
			nil,
		)

		// Aggro should be cleared
		if char.Aggro != nil {
			t.Error("Expected aggro to be cleared for non-existent user")
		}

		// Test with mob target
		char.Aggro = &Aggro{
			MobInstanceId: 888,
			Type:          DefaultAttack,
		}

		char.ValidateAggroTargets(
			nil,
			func(mobId int) bool {
				return mobId != 888 // Mob 888 doesn't exist
			},
		)

		if char.Aggro != nil {
			t.Error("Expected aggro to be cleared for non-existent mob")
		}

		// Test with valid targets
		char.Aggro = &Aggro{
			UserId: 777,
			Type:   DefaultAttack,
		}

		char.ValidateAggroTargets(
			func(userId int) bool {
				return true // All users exist
			},
			nil,
		)

		if char.Aggro == nil {
			t.Error("Expected aggro to be preserved for existing user")
		}
	})

	t.Run("Player damage preservation", func(t *testing.T) {
		char := &Character{
			Aggro: &Aggro{
				MobInstanceId: 1,
				Type:          DefaultAttack,
			},
			PlayerDamage: map[int]int{
				111: 100,
				222: 50,
				333: 25,
			},
			LastPlayerDamage: 5000,
		}

		// Save and restore
		state := char.SaveCombatState()
		if len(state.PlayerDamage) != 3 {
			t.Errorf("Expected 3 damage entries, got %d", len(state.PlayerDamage))
		}

		newChar := &Character{}
		newChar.RestoreCombatState(state)

		// Verify all damage entries restored
		if newChar.PlayerDamage[111] != 100 {
			t.Errorf("Expected damage 100 for user 111, got %d", newChar.PlayerDamage[111])
		}
		if newChar.PlayerDamage[222] != 50 {
			t.Errorf("Expected damage 50 for user 222, got %d", newChar.PlayerDamage[222])
		}
		if newChar.LastPlayerDamage != 5000 {
			t.Errorf("Expected LastPlayerDamage 5000, got %d", newChar.LastPlayerDamage)
		}
	})
}

func TestCombatStatePrepareCopyover(t *testing.T) {
	// Test the copyover preparation/restoration flow
	char := &Character{
		Aggro: &Aggro{
			UserId:        123,
			Type:          DefaultAttack,
			RoundsWaiting: 2,
		},
		PlayerDamage: map[int]int{123: 75},
	}

	// Prepare for copyover
	char.PrepareCombatForCopyover()

	// CombatStateData should be populated
	if char.CombatStateData == nil {
		t.Fatal("CombatStateData not populated during prepare")
	}

	// Simulate copyover - the character would be saved to YAML here
	// Then loaded back...

	// Restore after copyover
	err := char.RestoreCombatAfterCopyover()
	if err != nil {
		t.Fatalf("RestoreCombatAfterCopyover failed: %v", err)
	}

	// CombatStateData should be cleared
	if char.CombatStateData != nil {
		t.Error("CombatStateData not cleared after restore")
	}

	// Aggro should be restored
	if char.Aggro == nil {
		t.Fatal("Aggro not restored after copyover")
	}
	if char.Aggro.UserId != 123 {
		t.Errorf("Expected UserId 123 after copyover, got %d", char.Aggro.UserId)
	}
}
