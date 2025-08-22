package copyover

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// registry holds all registered subsystems
type registry struct {
	mu         sync.RWMutex
	subsystems map[string]*Subsystem
	order      []string // Maintains registration order
}

var globalRegistry = &registry{
	subsystems: make(map[string]*Subsystem),
	order:      make([]string, 0),
}

// Register adds a subsystem to participate in copyover
// This is the simple API - just provide gather and restore functions
func Register(name string, gather GatherFunc, restore RestoreFunc) error {
	return RegisterWithVeto(name, gather, restore, nil)
}

// RegisterWithVeto adds a subsystem with optional veto capability
func RegisterWithVeto(name string, gather GatherFunc, restore RestoreFunc, veto VetoFunc) error {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if name == "" {
		return fmt.Errorf("subsystem name cannot be empty")
	}

	if gather == nil || restore == nil {
		return fmt.Errorf("gather and restore functions are required")
	}

	if _, exists := globalRegistry.subsystems[name]; exists {
		return fmt.Errorf("subsystem %s already registered", name)
	}

	subsystem := &Subsystem{
		Name:    name,
		Gather:  gather,
		Restore: restore,
		Veto:    veto,
	}

	globalRegistry.subsystems[name] = subsystem
	globalRegistry.order = append(globalRegistry.order, name)

	// Don't log during init() as logger may not be initialized yet
	return nil
}

// Unregister removes a subsystem from copyover participation
func Unregister(name string) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	delete(globalRegistry.subsystems, name)

	// Remove from order slice
	newOrder := make([]string, 0, len(globalRegistry.order)-1)
	for _, n := range globalRegistry.order {
		if n != name {
			newOrder = append(newOrder, n)
		}
	}
	globalRegistry.order = newOrder
}

// GetRegisteredSubsystems returns a list of all registered subsystem names
func GetRegisteredSubsystems() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make([]string, len(globalRegistry.order))
	copy(result, globalRegistry.order)
	return result
}

// checkVetoes checks if any subsystem wants to veto the copyover
func checkVetoes() (canProceed bool, vetoes []VetoResult) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	canProceed = true
	vetoes = make([]VetoResult, 0)

	// Check in registration order
	for _, name := range globalRegistry.order {
		subsystem := globalRegistry.subsystems[name]
		if subsystem.Veto != nil {
			if ok, reason := subsystem.Veto(); !ok {
				canProceed = false
				vetoes = append(vetoes, VetoResult{
					Subsystem: name,
					Reason:    reason,
					Time:      time.Now(),
				})
				mudlog.Info("Copyover", "veto", name, "reason", reason)
			}
		}
	}

	return canProceed, vetoes
}

// gatherStates collects state from all registered subsystems
func gatherStates() (map[string]interface{}, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	states := make(map[string]interface{})

	// Gather in registration order
	for _, name := range globalRegistry.order {
		subsystem := globalRegistry.subsystems[name]

		mudlog.Info("Copyover", "gathering", name)
		state, err := subsystem.Gather()
		if err != nil {
			// Log error but continue with other subsystems
			mudlog.Error("Copyover", "subsystem", name, "error", "Failed to gather state", "err", err)
			continue
		}

		if state != nil {
			states[name] = state
		}
	}

	return states, nil
}

// restoreStates restores state to all registered subsystems
func restoreStates(states map[string]interface{}) error {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	var firstError error

	// Restore in registration order
	for _, name := range globalRegistry.order {
		subsystem := globalRegistry.subsystems[name]

		state, exists := states[name]
		if !exists {
			mudlog.Info("Copyover", "skipping", name, "reason", "no saved state")
			continue
		}

		mudlog.Info("Copyover", "restoring", name)
		if err := subsystem.Restore(state); err != nil {
			mudlog.Error("Copyover", "subsystem", name, "error", "Failed to restore state", "err", err)
			if firstError == nil {
				firstError = err
			}
		}
	}

	return firstError
}
