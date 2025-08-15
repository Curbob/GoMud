package combatrounds_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

func TestRoundBasedCombat(t *testing.T) {
	// Initialize mudlog for testing
	mudlog.SetupLogger(nil, "LOW", "", false)

	// The module should already be registered via init()
	// Check if it's registered
	systems := combat.ListCombatSystems()
	found := false
	for _, sys := range systems {
		if sys == "combat-rounds" {
			found = true
			break
		}
	}
	if !found {
		t.Skip("combat-rounds module not registered, skipping test")
	}

	// Activate the combat system
	err := combat.SetActiveCombatSystem("combat-rounds")
	if err != nil {
		t.Fatalf("Failed to set active combat system: %v", err)
	}

	// Verify it's active
	active := combat.GetActiveCombatSystem()
	if active == nil {
		t.Fatal("Expected active combat system, got nil")
	}
	if active.GetName() != "combat-rounds" {
		t.Fatalf("Expected combat-rounds, got %s", active.GetName())
	}

	// Test the calculator
	calc := active.GetCalculator()
	if calc == nil {
		t.Fatal("Expected calculator, got nil")
	}

	// Create test characters
	attacker := createTestCharacter()
	defender := createTestCharacter()

	// Test PowerRanking
	ranking := calc.PowerRanking(attacker, defender)
	if ranking <= 0 {
		t.Fatalf("Expected positive power ranking, got %f", ranking)
	}

	// Test that delegated PowerRanking works
	delegatedRanking := combat.PowerRanking(*attacker, *defender)
	if delegatedRanking != ranking {
		t.Fatalf("Delegated PowerRanking (%f) doesn't match calculator (%f)", delegatedRanking, ranking)
	}
}

func createTestCharacter() *characters.Character {
	char := &characters.Character{
		Name:  "TestCharacter",
		Level: 10,
		Stats: stats.Statistics{
			Strength: stats.StatInfo{
				Base:     50,
				ValueAdj: 50,
			},
			Speed: stats.StatInfo{
				Base:     50,
				ValueAdj: 50,
			},
			Vitality: stats.StatInfo{
				Base:     50,
				ValueAdj: 50,
			},
			Perception: stats.StatInfo{
				Base:     50,
				ValueAdj: 50,
			},
		},
		Health: 100,
		HealthMax: stats.StatInfo{
			Base:  100,
			Value: 100,
		},
	}

	return char
}
