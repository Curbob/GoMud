package combat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// MockCombatCalculator implements ICombatCalculator for testing
type MockCombatCalculator struct {
	hitResult       bool
	damage          int
	critResult      bool
	defense         int
	initiative      int
	attackCount     int
	powerRanking    int
}

func (m *MockCombatCalculator) CalculateHitChance(attacker, defender *characters.Character) bool {
	return m.hitResult
}

func (m *MockCombatCalculator) CalculateDamage(attacker, defender *characters.Character, weapon *items.Item) int {
	return m.damage
}

func (m *MockCombatCalculator) CalculateCriticalChance(attacker, defender *characters.Character) bool {
	return m.critResult
}

func (m *MockCombatCalculator) CalculateDefense(defender *characters.Character) int {
	return m.defense
}

func (m *MockCombatCalculator) CalculateInitiative(char *characters.Character) int {
	return m.initiative
}

func (m *MockCombatCalculator) CalculateAttackCount(attacker, defender *characters.Character) int {
	return m.attackCount
}

func (m *MockCombatCalculator) PowerRanking(attacker, defender *characters.Character) float64 {
	return float64(m.powerRanking)
}

// MockCombatSystem implements ICombatSystem for testing
type MockCombatSystem struct {
	calculator ICombatCalculator
	timer      ICombatTimer
}

func (m *MockCombatSystem) GetName() string {
	return "mock"
}

func (m *MockCombatSystem) Initialize() error {
	return nil
}

func (m *MockCombatSystem) Shutdown() error {
	return nil
}

func (m *MockCombatSystem) ProcessCombatRound() {
	// No-op for testing
}

func (m *MockCombatSystem) GetCalculator() ICombatCalculator {
	return m.calculator
}

func (m *MockCombatSystem) GetTimer() ICombatTimer {
	return m.timer
}

func TestPerformAttack_BasicHit(t *testing.T) {
	// Create basic test characters
	attacker := &characters.Character{
		Name:   "Attacker",
		Health: 100,
	}
	defender := &characters.Character{
		Name:   "Defender", 
		Health: 100,
	}

	// Create mock calculator
	calc := &MockCombatCalculator{
		hitResult:   true,
		damage:      25,
		critResult:  false,
		defense:     0,
		attackCount: 1,
	}

	// Track post-attack callback
	callbackCalled := false
	var resultDamage int
	postAttack := func(result *AttackResult) {
		callbackCalled = true
		resultDamage = result.DamageToTarget
	}

	// Perform attack
	result := performAttack(attacker, defender, User, Mob, calc, postAttack)

	// Verify results using testify
	assert.True(t, result.Hit, "Expected attack to hit")
	assert.Equal(t, 25, result.DamageToTarget, "Expected 25 damage")
	assert.True(t, callbackCalled, "Post-attack callback should have been called")
	assert.Equal(t, 25, resultDamage, "Callback should receive correct damage")
}

func TestPerformAttack_Miss(t *testing.T) {
	// Create test characters
	attacker := &characters.Character{Name: "Attacker"}
	defender := &characters.Character{Name: "Defender"}

	// Create mock calculator that always misses
	calc := &MockCombatCalculator{
		hitResult:   false,
		attackCount: 1,
	}

	// Perform attack
	result := performAttack(attacker, defender, User, Mob, calc, nil)

	// Verify results
	assert.False(t, result.Hit, "Expected attack to miss")
	assert.Zero(t, result.DamageToTarget, "No damage should be dealt on miss")
}

func TestPerformAttack_Critical(t *testing.T) {
	// Create test characters
	attacker := &characters.Character{Name: "Attacker"}
	defender := &characters.Character{Name: "Defender"}

	// Create mock calculator with critical hit
	calc := &MockCombatCalculator{
		hitResult:   true,
		damage:      25,
		critResult:  true,
		attackCount: 1,
	}

	// Perform attack
	result := performAttack(attacker, defender, User, Mob, calc, nil)

	// Verify results
	assert.True(t, result.Hit, "Expected hit")
	assert.True(t, result.Crit, "Expected critical hit")
}

func TestPerformAttack_WithDefense(t *testing.T) {
	// Create test characters
	attacker := &characters.Character{Name: "Attacker"}
	defender := &characters.Character{Name: "Defender"}

	// Create mock calculator with defense
	calc := &MockCombatCalculator{
		hitResult:   true,
		damage:      30,
		defense:     10, // 10 defense = 5 damage reduction
		attackCount: 1,
	}

	// Perform attack
	result := performAttack(attacker, defender, User, Mob, calc, nil)

	// Verify results
	assert.Equal(t, 25, result.DamageToTarget, "Expected damage after reduction")
	assert.Equal(t, 5, result.DamageToTargetReduction, "Expected damage reduction")
}

func TestPerformAttack_MultipleAttacks(t *testing.T) {
	// Create test characters
	attacker := &characters.Character{Name: "Attacker"}
	defender := &characters.Character{Name: "Defender"}

	// Create mock calculator with multiple attacks
	calc := &MockCombatCalculator{
		hitResult:   true,
		damage:      10,
		attackCount: 3, // 3 attacks
	}

	// Perform attack
	result := performAttack(attacker, defender, User, Mob, calc, nil)

	// Verify results - 3 attacks * 10 damage each
	assert.Equal(t, 30, result.DamageToTarget, "Expected total damage from multiple attacks")
}

func TestAttackMessages(t *testing.T) {
	// Create test characters
	attacker := &characters.Character{Name: "Attacker"}
	defender := &characters.Character{Name: "Defender"}

	// Test miss message generation
	calc := &MockCombatCalculator{
		hitResult:   false,
		attackCount: 1,
	}
	
	result := performAttack(attacker, defender, User, Mob, calc, nil)
	
	// Should have miss messages
	assert.NotEmpty(t, result.MessagesToSource, "Expected miss message for attacker")
	assert.NotEmpty(t, result.MessagesToTarget, "Expected miss message for defender")
	assert.NotEmpty(t, result.MessagesToSourceRoom, "Expected miss message for room")

	// Test hit message generation
	calc.hitResult = true
	calc.damage = 20
	
	result = performAttack(attacker, defender, User, Mob, calc, nil)
	
	// Should have hit messages
	assert.NotEmpty(t, result.MessagesToSource, "Expected hit message for attacker")
	assert.NotEmpty(t, result.MessagesToTarget, "Expected hit message for defender")
	assert.NotEmpty(t, result.MessagesToSourceRoom, "Expected hit message for room")
}

func TestGetActiveCombatSystem(t *testing.T) {
	// Save original state
	originalSystem := activeCombatSystem
	originalName := activeCombatSystemName
	defer func() {
		activeCombatSystem = originalSystem
		activeCombatSystemName = originalName
	}()

	// Test with no active system
	activeCombatSystem = nil
	activeCombatSystemName = ""
	
	assert.Nil(t, GetActiveCombatSystem(), "Expected nil when no system is active")
	assert.Empty(t, GetActiveCombatSystemName(), "Expected empty name when no system is active")

	// Test with active system
	mockSystem := &MockCombatSystem{}
	activeCombatSystem = mockSystem
	activeCombatSystemName = "test-system"
	
	assert.Equal(t, mockSystem, GetActiveCombatSystem(), "Expected to get the active system")
	assert.Equal(t, "test-system", GetActiveCombatSystemName(), "Expected correct system name")
}

func TestCombatSystemRegistry(t *testing.T) {
	// Create a unique name for this test
	testName := "test-registry-system"
	
	// Clean up after test
	defer func() {
		registryMutex.Lock()
		delete(combatRegistry, testName)
		registryMutex.Unlock()
	}()

	// Test registering a system
	mockSystem := &MockCombatSystem{}
	err := RegisterCombatSystem(testName, mockSystem)
	require.NoError(t, err, "Failed to register system")

	// Test duplicate registration
	err = RegisterCombatSystem(testName, mockSystem)
	assert.Error(t, err, "Expected error when registering duplicate system")

	// Test getting the system
	system, err := GetCombatSystem(testName)
	require.NoError(t, err, "Failed to get registered system")
	assert.Equal(t, mockSystem, system, "Got wrong system from registry")

	// Test getting non-existent system
	_, err = GetCombatSystem("non-existent")
	assert.Error(t, err, "Expected error when getting non-existent system")
}

func TestSetActiveCombatSystem(t *testing.T) {
	// Save original state
	originalSystem := activeCombatSystem
	originalName := activeCombatSystemName
	originalRegistry := make(map[string]ICombatSystem)
	
	// Copy registry
	registryMutex.Lock()
	for k, v := range combatRegistry {
		originalRegistry[k] = v
	}
	registryMutex.Unlock()
	
	// Restore state after test
	defer func() {
		registryMutex.Lock()
		combatRegistry = originalRegistry
		activeCombatSystem = originalSystem
		activeCombatSystemName = originalName
		registryMutex.Unlock()
	}()

	// Register a test system
	testName := "test-active-system"
	mockSystem := &MockCombatSystem{
		calculator: &MockCombatCalculator{},
	}
	
	err := RegisterCombatSystem(testName, mockSystem)
	require.NoError(t, err, "Failed to register system")

	// Set it as active
	err = SetActiveCombatSystem(testName)
	require.NoError(t, err, "Failed to set active system")

	// Verify it's active
	assert.Equal(t, testName, GetActiveCombatSystemName(), "Wrong active system name")

	// Test setting non-existent system
	err = SetActiveCombatSystem("non-existent")
	assert.Error(t, err, "Expected error when setting non-existent system as active")
}

func TestListCombatSystems(t *testing.T) {
	// Save original registry
	originalRegistry := make(map[string]ICombatSystem)
	registryMutex.Lock()
	for k, v := range combatRegistry {
		originalRegistry[k] = v
	}
	registryMutex.Unlock()
	
	// Restore after test
	defer func() {
		registryMutex.Lock()
		combatRegistry = originalRegistry
		registryMutex.Unlock()
	}()

	// Clear registry and add test systems
	registryMutex.Lock()
	combatRegistry = make(map[string]ICombatSystem)
	combatRegistry["system-a"] = &MockCombatSystem{}
	combatRegistry["system-b"] = &MockCombatSystem{}
	combatRegistry["system-c"] = &MockCombatSystem{}
	registryMutex.Unlock()

	// Get list
	systems := ListCombatSystems()
	
	// Verify
	assert.Len(t, systems, 3, "Expected 3 systems")
	assert.Contains(t, systems, "system-a")
	assert.Contains(t, systems, "system-b")
	assert.Contains(t, systems, "system-c")
}