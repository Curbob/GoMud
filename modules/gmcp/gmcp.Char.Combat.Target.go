// Package gmcp handles Combat Target updates for GMCP.
//
// Tracks the player's current combat target and HP changes, sending updates only when:
// - Target changes (different mob or cleared)
// - Target HP changes (damage/healing)
// - Target dies or moves away
package gmcp

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GMCPCombatTargetUpdate is sent when a player's combat target changes or target HP changes
type GMCPCombatTargetUpdate struct {
	UserId          int
	TargetName      string // Name of current target
	TargetHpCurrent int    // Current HP of target
	TargetHpMax     int    // Max HP of target
}

func (g GMCPCombatTargetUpdate) Type() string { return `GMCPCombatTargetUpdate` }

var (
	// targetMutex protects the target tracking maps
	targetMutex sync.RWMutex

	// userTargets tracks the current target for each user
	userTargets = make(map[int]int) // userId -> mobInstanceId

	// lastTargetHP tracks the last HP sent for each user's target
	lastTargetHP = make(map[int]int) // userId -> lastHP

)

// NOTE: Race condition mitigated by defensive cleanup in all handlers.
// If user disconnects between validateUserForGMCP and map operations,
// the cleanup functions handle it gracefully without data corruption.

func init() {
	events.RegisterListener(GMCPCombatTargetUpdate{}, handleCombatTargetUpdate)
	events.RegisterListener(events.NewRound{}, handleTargetNewRound)
	events.RegisterListener(events.MobVitalsChanged{}, handleTargetMobVitalsChanged)
	events.RegisterListener(events.MobDeath{}, handleTargetMobDeath)
	events.RegisterListener(events.RoomChange{}, handleTargetRoomChange)
}

func handleCombatTargetUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatTargetUpdate)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleCombatTargetUpdate", "error", "type assertion failed", "expectedType", "GMCPCombatTargetUpdate", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	_, valid := validateUserForGMCP(evt.UserId, "GMCPCombatTarget")
	if !valid {
		return events.Continue
	}

	payload := map[string]interface{}{
		"name": evt.TargetName,
	}

	if evt.TargetName != "" {
		payload["hp_current"] = fmt.Sprintf("%d", evt.TargetHpCurrent)
		payload["hp_max"] = fmt.Sprintf("%d", evt.TargetHpMax)
	} else {
		payload["hp_current"] = ""
		payload["hp_max"] = ""
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.Combat.Target",
		Payload: payload,
	})

	return events.Continue
}

// handleTargetNewRound checks for target changes each round
func handleTargetNewRound(e events.Event) events.ListenerReturn {
	_, typeOk := e.(events.NewRound)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetNewRound", "error", "type assertion failed", "expectedType", "events.NewRound", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Check all users currently in combat
	trackedUsers := GetUsersInCombat()

	for _, userId := range trackedUsers {
		user, valid := validateUserForGMCP(userId, "GMCPCombatTarget")
		if !valid {
			targetMutex.Lock()
			if _, exists := userTargets[userId]; exists {
				delete(userTargets, userId)
				delete(lastTargetHP, userId)
			}
			targetMutex.Unlock()
			continue
		}

		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
			targetMutex.Lock()
			oldTarget := userTargets[userId]
			newTarget := user.Character.Aggro.MobInstanceId

			// Update target if changed
			if oldTarget != newTarget {
				userTargets[userId] = newTarget
				delete(lastTargetHP, userId) // Reset HP tracking for new target
			}
			targetMutex.Unlock()

			// Send update if target changed
			if oldTarget != newTarget {
				sendTargetUpdate(userId)
			}
		} else {
			// Not in combat, clear target
			targetMutex.Lock()
			hadTarget := userTargets[userId] > 0
			delete(userTargets, userId)
			delete(lastTargetHP, userId)
			targetMutex.Unlock()

			if hadTarget {
				// Send empty target update
				handleCombatTargetUpdate(GMCPCombatTargetUpdate{
					UserId:     userId,
					TargetName: "",
				})
			}
		}
	}

	return events.Continue
}

// handleTargetMobVitalsChanged sends updates when a mob's vitals change
func handleTargetMobVitalsChanged(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MobVitalsChanged)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetMobVitalsChanged", "error", "type assertion failed", "expectedType", "events.MobVitalsChanged", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	targetMutex.RLock()
	userTargetsCopy := make(map[int]int)
	for k, v := range userTargets {
		userTargetsCopy[k] = v
	}
	targetMutex.RUnlock()

	for userId, targetId := range userTargetsCopy {
		if targetId == evt.MobId {
			// Validate user still exists before sending update
			user := users.GetByUserId(userId)
			if user == nil {
				targetMutex.Lock()
				delete(userTargets, userId)
				delete(lastTargetHP, userId)
				targetMutex.Unlock()
				mudlog.Warn("GMCPCombatTarget", "action", "handleTargetMobVitalsChanged", "issue", "user not found, cleaning up stale target tracking", "userId", userId)
				continue
			}
			sendTargetUpdate(userId)
		}
	}

	return events.Continue
}

// handleTargetMobDeath handles when a target dies
func handleTargetMobDeath(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MobDeath)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetMobDeath", "error", "type assertion failed", "expectedType", "events.MobDeath", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Check all users to see if this was their target
	targetMutex.RLock()
	userTargetsCopy := make(map[int]int)
	for k, v := range userTargets {
		userTargetsCopy[k] = v
	}
	targetMutex.RUnlock()

	for userId, targetId := range userTargetsCopy {
		if targetId == evt.InstanceId {
			// Validate user still exists before sending update
			user := users.GetByUserId(userId)
			if user == nil {
				// Clean up stale target tracking
				targetMutex.Lock()
				delete(userTargets, userId)
				delete(lastTargetHP, userId)
				targetMutex.Unlock()
				mudlog.Warn("GMCPCombatTarget", "action", "handleTargetMobDeath", "issue", "user not found, cleaning up stale target tracking", "userId", userId)
				continue
			}

			targetMutex.Lock()
			delete(userTargets, userId)
			delete(lastTargetHP, userId)
			targetMutex.Unlock()

			handleCombatTargetUpdate(GMCPCombatTargetUpdate{
				UserId:     userId,
				TargetName: "",
			})
		}
	}

	return events.Continue
}

// handleTargetRoomChange handles when a target moves away
func handleTargetRoomChange(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetRoomChange", "error", "type assertion failed", "expectedType", "events.RoomChange", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	if evt.MobInstanceId == 0 {
		return events.Continue
	}

	// Check all users to see if this was their target
	targetMutex.RLock()
	userTargetsCopy := make(map[int]int)
	for k, v := range userTargets {
		userTargetsCopy[k] = v
	}
	targetMutex.RUnlock()

	for userId, targetId := range userTargetsCopy {
		if targetId == evt.MobInstanceId {
			user := users.GetByUserId(userId)
			if user == nil {
				// Clean up stale target tracking
				targetMutex.Lock()
				delete(userTargets, userId)
				delete(lastTargetHP, userId)
				targetMutex.Unlock()
				mudlog.Warn("GMCPCombatTarget", "action", "handleTargetRoomChange", "issue", "user not found, cleaning up stale target tracking", "userId", userId)
				continue
			}

			if evt.ToRoomId != user.Character.RoomId {
				targetMutex.Lock()
				delete(userTargets, userId)
				delete(lastTargetHP, userId)
				targetMutex.Unlock()

				handleCombatTargetUpdate(GMCPCombatTargetUpdate{
					UserId:     userId,
					TargetName: "",
				})
			}
		}
	}

	return events.Continue
}

// sendTargetUpdate sends current target info for a user
func sendTargetUpdate(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		targetMutex.Lock()
		delete(userTargets, userId)
		delete(lastTargetHP, userId)
		targetMutex.Unlock()
		mudlog.Warn("GMCPCombatTarget", "action", "sendTargetUpdate", "issue", "user not found, cleaning up stale target tracking", "userId", userId)
		return
	}

	targetMutex.RLock()
	targetId := userTargets[userId]
	targetMutex.RUnlock()

	if targetId == 0 {
		return
	}

	mob := mobs.GetInstance(targetId)
	if mob == nil {
		mudlog.Error("GMCPCombatTarget", "action", "sendTargetUpdate", "error", "mob lookup failed", "mobId", targetId, "userId", userId)
		return
	}

	targetMutex.RLock()
	lastHP := lastTargetHP[userId]
	targetMutex.RUnlock()
	currentHP := mob.Character.Health
	if lastHP != currentHP {
		targetMutex.Lock()
		lastTargetHP[userId] = currentHP
		targetMutex.Unlock()

		handleCombatTargetUpdate(GMCPCombatTargetUpdate{
			UserId:          userId,
			TargetName:      util.StripANSI(mob.Character.Name),
			TargetHpCurrent: currentHP,
			TargetHpMax:     int(mob.Character.HealthMax.Value),
		})
	}
}

// cleanupCombatTarget removes all target tracking for a user
func cleanupCombatTarget(userId int) {
	targetMutex.Lock()
	delete(userTargets, userId)
	delete(lastTargetHP, userId)
	targetMutex.Unlock()
}
