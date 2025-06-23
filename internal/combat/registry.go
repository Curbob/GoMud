package combat

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	combatRegistry     = make(map[string]ICombatSystem)
	activeCombatSystem ICombatSystem
	registryMutex      sync.RWMutex
)

// RegisterCombatSystem registers a combat system implementation
func RegisterCombatSystem(name string, system ICombatSystem) error {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if _, exists := combatRegistry[name]; exists {
		return fmt.Errorf("combat system %s already registered", name)
	}

	combatRegistry[name] = system
	// Only log if mudlog is initialized (it might not be during tests)
	if mudlog.IsInitialized() {
		mudlog.Info("Combat Registry", "action", "registered", "system", name)
	}
	return nil
}

// GetCombatSystem returns a registered combat system by name
func GetCombatSystem(name string) (ICombatSystem, error) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	system, exists := combatRegistry[name]
	if !exists {
		return nil, fmt.Errorf("combat system %s not found", name)
	}

	return system, nil
}

// SetActiveCombatSystem sets the active combat system
func SetActiveCombatSystem(name string) error {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	// Shutdown current system if active
	if activeCombatSystem != nil {
		if err := activeCombatSystem.Shutdown(); err != nil {
			// Only log if mudlog is initialized (it might not be during tests)
			if mudlog.IsInitialized() {
				mudlog.Error("Combat Registry", "action", "shutdown failed", "error", err)
			}
		}
	}

	// Get new system
	system, exists := combatRegistry[name]
	if !exists {
		return fmt.Errorf("combat system %s not found", name)
	}

	// Initialize new system
	if err := system.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize combat system %s: %w", name, err)
	}

	activeCombatSystem = system
	// Only log if mudlog is initialized (it might not be during tests)
	if mudlog.IsInitialized() {
		mudlog.Info("Combat Registry", "action", "activated", "system", name)
	}
	return nil
}

// GetActiveCombatSystem returns the currently active combat system
func GetActiveCombatSystem() ICombatSystem {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	return activeCombatSystem
}

// ListCombatSystems returns all registered combat system names
func ListCombatSystems() []string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	names := make([]string, 0, len(combatRegistry))
	for name := range combatRegistry {
		names = append(names, name)
	}

	return names
}

// ShutdownCombatSystem shuts down the active combat system
func ShutdownCombatSystem() error {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if activeCombatSystem == nil {
		return nil
	}

	err := activeCombatSystem.Shutdown()
	activeCombatSystem = nil
	return err
}

// ClearRegistry removes all registered combat systems (for testing)
func ClearRegistry() {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	combatRegistry = make(map[string]ICombatSystem)
	activeCombatSystem = nil
}
