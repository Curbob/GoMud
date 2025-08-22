package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// MockCombatSystem implements ICombatSystem for testing
type MockCombatSystem struct {
	name string
}

func (m *MockCombatSystem) Initialize() error {
	return nil
}

func (m *MockCombatSystem) Shutdown() error {
	return nil
}

func (m *MockCombatSystem) GetName() string {
	return m.name
}

func (m *MockCombatSystem) ProcessCombatRound() {
	// Mock implementation
}

func (m *MockCombatSystem) GetCalculator() combat.ICombatCalculator {
	return nil
}

func (m *MockCombatSystem) GetTimer() combat.ICombatTimer {
	return nil
}

// Remove ExecuteCombatAction as it's not part of the interface

func TestCombatRegistry(t *testing.T) {
	// Initialize mudlog for testing
	mudlog.SetupLogger(nil, "LOW", "", false)

	// Clear any existing registrations
	combat.ClearRegistry()

	// Test registering a combat system
	mockSystem := &MockCombatSystem{name: "test-combat"}
	err := combat.RegisterCombatSystem("test-combat", mockSystem)
	if err != nil {
		t.Fatalf("Failed to register combat system: %v", err)
	}

	// Test setting active combat system
	err = combat.SetActiveCombatSystem("test-combat")
	if err != nil {
		t.Fatalf("Failed to set active combat system: %v", err)
	}

	// Test getting active combat system
	activeSystem := combat.GetActiveCombatSystem()
	if activeSystem == nil {
		t.Fatal("Expected active combat system, got nil")
	}
	if activeSystem.GetName() != "test-combat" {
		t.Fatalf("Expected active system name 'test-combat', got %s", activeSystem.GetName())
	}

	// Test listing combat systems
	systems := combat.ListCombatSystems()
	found := false
	for _, name := range systems {
		if name == "test-combat" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected 'test-combat' in list of combat systems")
	}
}

func TestCombatRegistryErrors(t *testing.T) {
	// Test duplicate registration
	mockSystem := &MockCombatSystem{name: "duplicate"}
	err := combat.RegisterCombatSystem("duplicate", mockSystem)
	if err != nil {
		t.Fatalf("Failed to register combat system: %v", err)
	}

	// Try to register again
	err = combat.RegisterCombatSystem("duplicate", mockSystem)
	if err == nil {
		t.Fatal("Expected error when registering duplicate combat system")
	}

	// Test setting non-existent combat system
	err = combat.SetActiveCombatSystem("non-existent")
	if err == nil {
		t.Fatal("Expected error when setting non-existent combat system")
	}
}
