package combat

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	// NOTE: Cannot use mudlog here as this may be called during init()
	// before mudlog is initialized. Logging happens later when systems are activated.
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
	// Validate input
	if name == "" {
		return fmt.Errorf("combat system name cannot be empty")
	}

	// Get references while holding lock briefly
	registryMutex.Lock()
	oldSystem := activeCombatSystem
	oldName := activeCombatSystemName

	// Get new system
	system, exists := combatRegistry[name]
	if !exists {
		availableSystems := make([]string, 0, len(combatRegistry))
		for k := range combatRegistry {
			availableSystems = append(availableSystems, k)
		}
		registryMutex.Unlock()
		return fmt.Errorf("combat system '%s' not found, available systems: %v", name, availableSystems)
	}

	// Clear active system to prevent access during transition
	activeCombatSystem = nil
	activeCombatSystemName = ""
	registryMutex.Unlock()

	// Shutdown current system if active (outside of lock)
	if oldSystem != nil {
		if err := oldSystem.Shutdown(); err != nil {
			mudlog.Error("Combat Registry", "action", "shutdown failed", "system", oldName, "error", err)
			// Continue anyway - we don't want to block switching due to shutdown errors
		}
	}

	// Initialize new system (outside of lock)
	if true {
		mudlog.Info("Combat Registry", "action", "initializing", "system", name)
	}
	if err := system.Initialize(); err != nil {
		// Try to restore previous system on failure
		if oldSystem != nil && oldName != "" {
			if restoreErr := oldSystem.Initialize(); restoreErr != nil {
				if true {
					mudlog.Error("Combat Registry", "action", "restore failed", "error", restoreErr)
				}
			} else {
				// Restore the old system
				registryMutex.Lock()
				activeCombatSystem = oldSystem
				activeCombatSystemName = oldName
				registryMutex.Unlock()
			}
		}
		return fmt.Errorf("failed to initialize combat system %s: %w", name, err)
	}

	// Set the new active system
	registryMutex.Lock()
	activeCombatSystem = system
	activeCombatSystemName = name
	registryMutex.Unlock()

	// Only log if mudlog is initialized (it might not be during tests)
	if true {
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
		if true {
			mudlog.Error("Combat Registry", "action", "shutdown error", "system", systemName, "error", err)
		}
	}

	// Clear active system regardless of error
	activeCombatSystem = nil
	activeCombatSystemName = ""

	if true {
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

// InitializeRegistry sets up the combat registry event handlers
// ARCHITECTURAL NOTE: This must be called explicitly from main() rather than
// using init() to avoid event handler registration during init phase, which
// could cause initialization order issues. The combat system needs the event
// system to be fully initialized before registering handlers.
func InitializeRegistry() {
	events.RegisterListener(SwitchCombatSystemEvent{}, handleCombatSystemSwitch)
	mudlog.Info("Combat Registry", "action", "event handler registered")
}

// handleCombatSystemSwitch processes combat system switch events
func handleCombatSystemSwitch(e events.Event) events.ListenerReturn {
	if true {
		mudlog.Info("Combat Registry", "action", "handleCombatSystemSwitch started")
	}

	evt, ok := e.(SwitchCombatSystemEvent)
	if !ok {
		return events.Continue
	}

	// Get the requesting user
	user := users.GetByUserId(evt.RequestingUser)
	if user == nil {
		return events.Continue
	}

	if true {
		mudlog.Info("Combat Registry", "action", "calling SetActiveCombatSystem", "newSystem", evt.NewSystem)
	}

	// Attempt to switch
	if err := SetActiveCombatSystem(evt.NewSystem); err != nil {
		// Restore combat states on failure
		for _, state := range evt.SavedStates {
			if u := users.GetByUserId(state.UserId); u != nil {
				u.Character.Aggro = state.Aggro
			}
		}
		for mobId, aggro := range evt.SavedMobStates {
			if mob := mobs.GetInstance(mobId); mob != nil {
				mob.Character.Aggro = aggro
			}
		}

		user.SendText(fmt.Sprintf(`<ansi fg="red">Failed to switch combat system: %s</ansi>`, err.Error()))
		mudlog.Error("Combat Registry", "action", "switch failed", "error", err.Error())
		return events.Continue
	}

	// ARCHITECTURAL NOTE: Combat configuration is integrated into the main config
	// rather than using plugin-specific config. This is intentional as combat
	// is a core system that needs to be initialized before plugins are loaded.
	// The active combat system must be persisted globally to ensure correct
	// initialization on server restart.
	if err := configs.SetVal("GamePlay.Combat.Style", evt.NewSystem); err != nil {
		user.SendText(fmt.Sprintf(`<ansi fg="red">Failed to save combat system setting: %s</ansi>`, err.Error()))
	}

	user.SendText(fmt.Sprintf(`<ansi fg="green">Combat system switched from %s to %s</ansi>`, evt.OldSystem, evt.NewSystem))

	// Restore combat states with new system
	for _, state := range evt.SavedStates {
		if u := users.GetByUserId(state.UserId); u != nil {
			u.Character.Aggro = state.Aggro

			// Register with new combat system if it has timers
			if evt.NewSystem == "combat-rounds" {
				// Round-based combat will pick up players on next combat check
				u.SendText(`<ansi fg="yellow">[SYSTEM] Combat resumed with round-based system.</ansi>`)
			} else {
				u.SendText(`<ansi fg="yellow">[SYSTEM] Combat resumed with twitch system.</ansi>`)
			}
		}
	}

	// Restore mob aggro
	for mobId, aggro := range evt.SavedMobStates {
		if mob := mobs.GetInstance(mobId); mob != nil {
			mob.Character.Aggro = aggro
		}
	}

	// Broadcast to non-combat players
	for _, u := range users.GetAllActiveUsers() {
		if u.UserId != user.UserId && u.Character.Aggro == nil {
			u.SendText(fmt.Sprintf(`<ansi fg="yellow">[SYSTEM] Combat system changed to: %s</ansi>`, evt.NewSystem))
		}
	}

	return events.Continue
}
