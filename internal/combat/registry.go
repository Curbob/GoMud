package combat

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	combatRegistry         = make(map[string]ICombatSystem)
	activeCombatSystem     ICombatSystem
	activeCombatSystemName string
	registryMutex          sync.RWMutex
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

	// Validate input
	if name == "" {
		return fmt.Errorf("combat system name cannot be empty")
	}

	// Shutdown current system if active
	if activeCombatSystem != nil {
		oldName := activeCombatSystemName
		if err := activeCombatSystem.Shutdown(); err != nil {
			// Only log if mudlog is initialized (it might not be during tests)
			if mudlog.IsInitialized() {
				mudlog.Error("Combat Registry", "action", "shutdown failed", "system", oldName, "error", err)
			}
			// Continue anyway - we don't want to block switching due to shutdown errors
		}
	}

	// Get new system
	system, exists := combatRegistry[name]
	if !exists {
		availableSystems := make([]string, 0, len(combatRegistry))
		for k := range combatRegistry {
			availableSystems = append(availableSystems, k)
		}
		return fmt.Errorf("combat system '%s' not found, available systems: %v", name, availableSystems)
	}

	// Initialize new system
	if err := system.Initialize(); err != nil {
		// Try to restore previous system on failure
		if activeCombatSystem != nil && activeCombatSystemName != "" {
			if restoreErr := activeCombatSystem.Initialize(); restoreErr != nil {
				if mudlog.IsInitialized() {
					mudlog.Error("Combat Registry", "action", "restore failed", "error", restoreErr)
				}
			}
		}
		return fmt.Errorf("failed to initialize combat system %s: %w", name, err)
	}

	activeCombatSystem = system
	activeCombatSystemName = name
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

// GetActiveCombatSystemName returns the name of the currently active combat system
func GetActiveCombatSystemName() string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	return activeCombatSystemName
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

	systemName := activeCombatSystemName
	err := activeCombatSystem.Shutdown()
	if err != nil {
		if mudlog.IsInitialized() {
			mudlog.Error("Combat Registry", "action", "shutdown error", "system", systemName, "error", err)
		}
	}

	// Clear active system regardless of error
	activeCombatSystem = nil
	activeCombatSystemName = ""

	if mudlog.IsInitialized() {
		mudlog.Info("Combat Registry", "action", "shutdown complete", "system", systemName)
	}

	return err
}

// ClearRegistry removes all registered combat systems (for testing)
func ClearRegistry() {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	combatRegistry = make(map[string]ICombatSystem)
	activeCombatSystem = nil
}
