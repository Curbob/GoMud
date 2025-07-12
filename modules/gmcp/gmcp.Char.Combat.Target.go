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
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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

	// mobHealthTracking tracks HP for all mobs in combat with users
	mobHealthTracking = make(map[int]int) // mobInstanceId -> lastKnownHP
)

func init() {
	// Register listener for combat target updates
	events.RegisterListener(GMCPCombatTargetUpdate{}, handleCombatTargetUpdate)

	// Listen for events that affect targets
	events.RegisterListener(events.NewRound{}, handleTargetNewRound)
	events.RegisterListener(events.CharacterVitalsChanged{}, handleTargetVitalsChanged)
	events.RegisterListener(events.MobVitalsChanged{}, handleTargetMobVitalsChanged)
	events.RegisterListener(events.MobDeath{}, handleTargetMobDeath)
	events.RegisterListener(events.RoomChange{}, handleTargetRoomChange)
	
	// Clean up when player disconnects
	events.RegisterListener(events.PlayerDespawn{}, handleTargetPlayerDespawn)
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

	// Build the payload
	payload := map[string]interface{}{
		"name": evt.TargetName,
	}

	// Include HP info if we have a target
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

	// Check all online users for target changes
	for _, userId := range users.GetOnlineUserIds() {
		user, valid := validateUserForGMCP(userId, "GMCPCombatTarget")
		if !valid {
			// Clean up stale target tracking
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

// handleTargetVitalsChanged sends updates when in combat and vitals change
func handleTargetVitalsChanged(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.CharacterVitalsChanged)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetVitalsChanged", "error", "type assertion failed", "expectedType", "events.CharacterVitalsChanged", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Only care about players in combat
	if evt.UserId < 1 {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		// Clean up stale target tracking if user no longer exists
		targetMutex.Lock()
		if _, exists := userTargets[evt.UserId]; exists {
			delete(userTargets, evt.UserId)
			delete(lastTargetHP, evt.UserId)
			mudlog.Warn("GMCPCombatTarget", "action", "handleTargetVitalsChanged", "issue", "user not found, cleaning up stale target tracking", "userId", evt.UserId)
		}
		targetMutex.Unlock()
		return events.Continue
	}

	if user.Character.Aggro == nil {
		// Not in combat - this is normal flow, not an error
		return events.Continue
	}

	// Send target update to capture any HP changes
	sendTargetUpdate(evt.UserId)

	// Also check if we should send damage updates for any mobs in the room
	checkForMobDamage(evt.UserId)

	return events.Continue
}

// handleTargetMobVitalsChanged sends updates when a mob's vitals change
func handleTargetMobVitalsChanged(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MobVitalsChanged)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetMobVitalsChanged", "error", "type assertion failed", "expectedType", "events.MobVitalsChanged", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Check all users to see if this mob is their target
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
				// Clean up stale target tracking
				targetMutex.Lock()
				delete(userTargets, userId)
				delete(lastTargetHP, userId)
				targetMutex.Unlock()
				mudlog.Warn("GMCPCombatTarget", "action", "handleTargetMobVitalsChanged", "issue", "user not found, cleaning up stale target tracking", "userId", userId)
				continue
			}
			// This mob is someone's target, send update
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

			// Their target died, clear it
			targetMutex.Lock()
			delete(userTargets, userId)
			delete(lastTargetHP, userId)
			targetMutex.Unlock()

			// Send empty target update
			handleCombatTargetUpdate(GMCPCombatTargetUpdate{
				UserId:     userId,
				TargetName: "",
			})
		}
	}

	// Clean up health tracking for dead mob
	cleanupMobTracking(evt.InstanceId)

	return events.Continue
}

// handleTargetRoomChange handles when a target moves away
func handleTargetRoomChange(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetRoomChange", "error", "type assertion failed", "expectedType", "events.RoomChange", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Only care about mob movements
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

			// If mob moved to a different room than the player, clear target
			if evt.ToRoomId != user.Character.RoomId {
				targetMutex.Lock()
				delete(userTargets, userId)
				delete(lastTargetHP, userId)
				targetMutex.Unlock()

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

// sendTargetUpdate sends current target info for a user
func sendTargetUpdate(userId int) {
	// Validate user exists before processing
	user := users.GetByUserId(userId)
	if user == nil {
		// Clean up stale target tracking
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

	// Check if HP changed
	targetMutex.RLock()
	lastHP := lastTargetHP[userId]
	targetMutex.RUnlock()
	currentHP := mob.Character.Health
	if lastHP != currentHP {
		targetMutex.Lock()
		lastTargetHP[userId] = currentHP
		targetMutex.Unlock()

		// Send update with current HP
		handleCombatTargetUpdate(GMCPCombatTargetUpdate{
			UserId:          userId,
			TargetName:      mob.Character.Name,
			TargetHpCurrent: currentHP,
			TargetHpMax:     int(mob.Character.HealthMax.Value),
		})
	}
}

// checkForMobDamage checks all mobs in combat and sends damage updates if HP changed
func checkForMobDamage(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		mudlog.Warn("GMCPCombatTarget", "action", "checkForMobDamage", "issue", "user not found", "userId", userId)
		return
	}

	// Import gmcp package to use SendCombatDamage
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		mudlog.Error("GMCPCombatTarget", "action", "checkForMobDamage", "error", "room lookup failed", "roomId", user.Character.RoomId, "userId", userId)
		return
	}

	// Check all mobs in the room
	for _, mobId := range room.GetMobs() {
		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		// Check if this mob is in combat with any player
		if mob.Character.Aggro != nil && mob.Character.Aggro.UserId > 0 {
			targetMutex.RLock()
			lastHP, tracked := mobHealthTracking[mobId]
			targetMutex.RUnlock()
			currentHP := mob.Character.Health

			// If HP changed, send damage update
			if tracked && lastHP != currentHP {
				hpDiff := lastHP - currentHP
				targetMutex.Lock()
				mobHealthTracking[mobId] = currentHP
				targetMutex.Unlock()

				// Determine source and target for the damage message
				if hpDiff > 0 {
					// Mob took damage
					SendCombatDamage(
						userId,
						hpDiff,
						"physical",
						user.Character.Name,
						mob.Character.Name,
					)
				} else if hpDiff < 0 {
					// Mob was healed (rare but possible)
					SendCombatDamage(
						userId,
						hpDiff,
						"heal",
						"unknown",
						mob.Character.Name,
					)
				}
			} else if !tracked {
				// Start tracking this mob
				targetMutex.Lock()
				mobHealthTracking[mobId] = currentHP
				targetMutex.Unlock()
			}
		}
	}
}

// Clean up mob health tracking when mob dies
func cleanupMobTracking(mobId int) {
	targetMutex.Lock()
	delete(mobHealthTracking, mobId)
	targetMutex.Unlock()
}

// handleTargetPlayerDespawn cleans up when player leaves
func handleTargetPlayerDespawn(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		mudlog.Error("GMCPCombatTarget", "action", "handleTargetPlayerDespawn", "error", "type assertion failed", "expectedType", "events.PlayerDespawn", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Clean up tracking maps for this user
	targetMutex.Lock()
	delete(userTargets, evt.UserId)
	delete(lastTargetHP, evt.UserId)
	targetMutex.Unlock()

	return events.Continue
}
