package combattwitch

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// CombatTwitchState represents the state of twitch-based combat during copyover
type CombatTwitchState struct {
	Active       bool            `json:"active"`
	Cooldowns    []CooldownState `json:"cooldowns"`
	UserTargets  map[int]string  `json:"user_targets"` // userId -> target name
	TimerRunning bool            `json:"timer_running"`
}

// CooldownState represents a saved cooldown
type CooldownState struct {
	ActorKey    string        `json:"actor_key"` // "user:123" or "mob:456"
	ActorId     int           `json:"actor_id"`
	ActorType   string        `json:"actor_type"` // "user" or "mob"
	NextAction  time.Time     `json:"next_action"`
	OffBalance  bool          `json:"off_balance"`
	MaxDuration time.Duration `json:"max_duration"`
}

var combatInstance *TwitchCombat

func init() {
	// Register with copyover system
	copyover.RegisterWithVeto("combat-twitch", gatherCombatState, restoreCombatState, vetoCombatCopyover)
}

// SetCombatInstance stores the combat instance for copyover
func SetCombatInstance(tc *TwitchCombat) {
	combatInstance = tc
}

// gatherCombatState collects state before copyover
func gatherCombatState() (interface{}, error) {
	if combatInstance == nil {
		mudlog.Info("Copyover", "subsystem", "combat-twitch", "status", "no active instance")
		return nil, nil
	}

	state := CombatTwitchState{
		Active:       combatInstance.active,
		Cooldowns:    make([]CooldownState, 0),
		UserTargets:  make(map[int]string),
		TimerRunning: false,
	}

	// Check if timer is running
	if combatInstance.timer != nil {
		combatInstance.timer.timerMutex.Lock()
		state.TimerRunning = combatInstance.timer.timerStarted
		combatInstance.timer.timerMutex.Unlock()
	}

	// Stop the timer gracefully
	if state.TimerRunning {
		mudlog.Info("Copyover", "subsystem", "combat-twitch", "action", "stopping timer")
		combatInstance.timer.Stop()
	}

	// Collect cooldowns
	if combatInstance.timer != nil {
		combatInstance.timer.cooldownMutex.RLock()
		for key, cooldown := range combatInstance.timer.cooldowns {
			actorType := "user"
			if cooldown.ActorType == combat.Mob {
				actorType = "mob"
			}

			state.Cooldowns = append(state.Cooldowns, CooldownState{
				ActorKey:    key,
				ActorId:     cooldown.ActorId,
				ActorType:   actorType,
				NextAction:  cooldown.NextAction,
				OffBalance:  cooldown.OffBalance,
				MaxDuration: cooldown.MaxDuration,
			})
		}
		combatInstance.timer.cooldownMutex.RUnlock()
	}

	// Collect user targets
	combatInstance.userTargetMutex.RLock()
	for userId, target := range combatInstance.userTargets {
		state.UserTargets[userId] = target
	}
	combatInstance.userTargetMutex.RUnlock()

	mudlog.Info("Copyover", "subsystem", "combat-twitch",
		"gathered", len(state.Cooldowns), "cooldowns",
		"targets", len(state.UserTargets))

	return state, nil
}

// restoreCombatState restores state after copyover
func restoreCombatState(data interface{}) error {
	if data == nil {
		mudlog.Info("Copyover", "subsystem", "combat-twitch", "status", "no state to restore")
		return nil
	}

	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal combat data: %w", err)
	}

	var state CombatTwitchState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal combat state: %w", err)
	}

	if combatInstance == nil {
		mudlog.Warn("Copyover", "subsystem", "combat-twitch", "warning", "no combat instance to restore to")
		return nil
	}

	// Restore active state
	combatInstance.active = state.Active

	// Restore user targets
	combatInstance.userTargetMutex.Lock()
	combatInstance.userTargets = state.UserTargets
	combatInstance.userTargetMutex.Unlock()

	// Restore cooldowns
	if combatInstance.timer != nil && len(state.Cooldowns) > 0 {
		combatInstance.timer.cooldownMutex.Lock()
		for _, cd := range state.Cooldowns {
			actorType := combat.User
			if cd.ActorType == "mob" {
				actorType = combat.Mob
			}

			cooldown := &ActorCooldown{
				NextAction:  cd.NextAction,
				OffBalance:  cd.OffBalance,
				ActorId:     cd.ActorId,
				ActorType:   actorType,
				MaxDuration: cd.MaxDuration,
			}

			combatInstance.timer.cooldowns[cd.ActorKey] = cooldown

			mudlog.Info("Copyover", "subsystem", "combat-twitch",
				"restored", "cooldown",
				"actor", cd.ActorKey,
				"offBalance", cd.OffBalance)
		}
		combatInstance.timer.cooldownMutex.Unlock()

		// Restart timer if we have active cooldowns
		if state.TimerRunning {
			mudlog.Info("Copyover", "subsystem", "combat-twitch", "action", "restarting timer")
			combatInstance.timer.timerMutex.Lock()
			combatInstance.timer.timerStarted = true
			combatInstance.timer.timerMutex.Unlock()
			combatInstance.timer.BaseTimer.Start()
		}
	}

	mudlog.Info("Copyover", "subsystem", "combat-twitch",
		"restored", len(state.Cooldowns), "cooldowns",
		"targets", len(state.UserTargets))

	return nil
}

// vetoCombatCopyover checks if it's safe to copyover
func vetoCombatCopyover() (bool, string) {
	if combatInstance == nil {
		return true, ""
	}

	// Check number of active cooldowns
	activeCooldowns := 0
	if combatInstance.timer != nil {
		combatInstance.timer.cooldownMutex.RLock()
		activeCooldowns = len(combatInstance.timer.cooldowns)
		combatInstance.timer.cooldownMutex.RUnlock()
	}

	if activeCooldowns > 50 {
		// Many active cooldowns - soft veto
		return true, fmt.Sprintf("%d active cooldowns", activeCooldowns)
	}

	// Check if any cooldowns are about to expire (within 1 second)
	now := time.Now()
	expiringSoon := 0

	if combatInstance.timer != nil {
		combatInstance.timer.cooldownMutex.RLock()
		for _, cooldown := range combatInstance.timer.cooldowns {
			if cooldown.NextAction.Sub(now) < time.Second {
				expiringSoon++
			}
		}
		combatInstance.timer.cooldownMutex.RUnlock()
	}

	if expiringSoon > 5 {
		// Many cooldowns about to expire - soft veto
		return true, fmt.Sprintf("%d cooldowns expiring soon", expiringSoon)
	}

	return true, ""
}
