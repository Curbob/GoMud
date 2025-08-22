package combatrounds

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// CombatRoundsState represents the state of round-based combat during copyover
type CombatRoundsState struct {
	Active         bool                       `json:"active"`
	CurrentRound   int                        `json:"current_round"`
	RoundStartTime time.Time                  `json:"round_start_time"`
	ActiveCombats  []ActiveCombat             `json:"active_combats"`
	PendingActions map[string][]PendingAction `json:"pending_actions"` // userId/mobId -> actions
	TimerRunning   bool                       `json:"timer_running"`
}

// ActiveCombat tracks an ongoing combat
type ActiveCombat struct {
	AttackerId   string    `json:"attacker_id"`   // "user:123" or "mob:456"
	AttackerType string    `json:"attacker_type"` // "user" or "mob"
	TargetId     string    `json:"target_id"`
	TargetType   string    `json:"target_type"`
	StartedAt    time.Time `json:"started_at"`
	LastAction   time.Time `json:"last_action"`
}

// PendingAction represents a queued combat action
type PendingAction struct {
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Timestamp time.Time `json:"timestamp"`
}

var combatInstance *RoundBasedCombat

func init() {
	// Register with copyover system
	copyover.RegisterWithVeto("combat-rounds", gatherCombatState, restoreCombatState, vetoCombatCopyover)
}

// SetCombatInstance stores the combat instance for copyover
func SetCombatInstance(rbc *RoundBasedCombat) {
	combatInstance = rbc
}

// gatherCombatState collects state before copyover
func gatherCombatState() (interface{}, error) {
	if combatInstance == nil {
		mudlog.Info("Copyover", "subsystem", "combat-rounds", "status", "no active instance")
		return nil, nil
	}

	state := CombatRoundsState{
		Active:         combatInstance.active,
		ActiveCombats:  make([]ActiveCombat, 0),
		PendingActions: make(map[string][]PendingAction),
		TimerRunning:   combatInstance.timer != nil && combatInstance.timer.IsRunning(),
	}

	// Stop the timer gracefully
	if combatInstance.timer != nil && combatInstance.timer.IsRunning() {
		mudlog.Info("Copyover", "subsystem", "combat-rounds", "action", "stopping timer")
		combatInstance.timer.Stop()

		// Save timer state
		state.CurrentRound = combatInstance.timer.GetCurrentRound()
		state.RoundStartTime = combatInstance.timer.GetRoundStartTime()
	}

	// Collect active combats from tracking data
	state.ActiveCombats = collectActiveCombats()

	// Collect pending actions
	state.PendingActions = collectPendingActions()

	mudlog.Info("Copyover", "subsystem", "combat-rounds",
		"gathered", len(state.ActiveCombats),
		"combats", len(state.PendingActions), "pending")

	return state, nil
}

// restoreCombatState restores state after copyover
func restoreCombatState(data interface{}) error {
	if data == nil {
		mudlog.Info("Copyover", "subsystem", "combat-rounds", "status", "no state to restore")
		return nil
	}

	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal combat data: %w", err)
	}

	var state CombatRoundsState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal combat state: %w", err)
	}

	if combatInstance == nil {
		mudlog.Warn("Copyover", "subsystem", "combat-rounds", "warning", "no combat instance to restore to")
		return nil
	}

	// Restore active state
	combatInstance.active = state.Active

	// Restart timer if it was running
	if state.TimerRunning && combatInstance.timer != nil {
		mudlog.Info("Copyover", "subsystem", "combat-rounds", "action", "restarting timer")

		// Restore timer state
		if state.CurrentRound > 0 {
			combatInstance.timer.SetCurrentRound(state.CurrentRound)
		}

		// Restart the timer
		combatInstance.timer.Start()
	}

	// Log restored combats
	if len(state.ActiveCombats) > 0 {
		mudlog.Info("Copyover", "subsystem", "combat-rounds",
			"restored", len(state.ActiveCombats), "active combats")

		// Note: Actual combat restoration would need to reconnect
		// the combatants based on their IDs
		for _, combat := range state.ActiveCombats {
			mudlog.Info("Copyover", "combat", "restored",
				"attacker", combat.AttackerId,
				"target", combat.TargetId)
		}
	}

	mudlog.Info("Copyover", "subsystem", "combat-rounds", "status", "restore complete")
	return nil
}

// vetoCombatCopyover checks if it's safe to copyover
func vetoCombatCopyover() (bool, string) {
	if combatInstance == nil {
		return true, ""
	}

	// Check if we're in the middle of processing combat
	activeCombats := collectActiveCombats()
	if len(activeCombats) > 10 {
		// Many active combats - soft veto
		return true, fmt.Sprintf("%d active combats", len(activeCombats))
	}

	// Check if timer is in critical section
	if combatInstance.timer != nil && combatInstance.timer.IsProcessing() {
		// Hard veto if processing combat round
		return false, "combat round in progress"
	}

	return true, ""
}

// Helper functions

func collectActiveCombats() []ActiveCombat {
	// This would collect actual combat data from the combat tracking system
	// For now, return empty slice
	return []ActiveCombat{}
}

func collectPendingActions() map[string][]PendingAction {
	// This would collect queued actions from the combat system
	// For now, return empty map
	return make(map[string][]PendingAction)
}

// Timer helper methods that would need to be added to RoundBasedTimer

func (t *RoundBasedTimer) IsRunning() bool {
	// Check if timer is running using BaseTimer's IsActive method
	return t != nil && t.BaseTimer != nil && t.BaseTimer.IsActive()
}

func (t *RoundBasedTimer) IsProcessing() bool {
	// Check if currently processing a round
	return false // Simplified
}

func (t *RoundBasedTimer) GetCurrentRound() int {
	// Return current round number
	return 0 // Simplified
}

func (t *RoundBasedTimer) GetRoundStartTime() time.Time {
	// Return when current round started
	return time.Now() // Simplified
}

func (t *RoundBasedTimer) SetCurrentRound(round int) {
	// Set the current round number
	// Implementation would update internal state
}
