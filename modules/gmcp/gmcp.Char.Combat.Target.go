package gmcp

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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
}

func handleCombatTargetUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatTargetUpdate)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Make sure they have GMCP enabled
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
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

	// Send the GMCP update
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
		return events.Continue
	}

	// Check all online users for target changes
	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil {
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
		return events.Continue
	}

	// Only care about players in combat
	if evt.UserId < 1 {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil || user.Character.Aggro == nil {
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
	targetMutex.RLock()
	targetId := userTargets[userId]
	targetMutex.RUnlock()

	if targetId == 0 {
		return
	}

	mob := mobs.GetInstance(targetId)
	if mob == nil {
		return
	}

	// Check if HP changed
	targetMutex.Lock()
	lastHP := lastTargetHP[userId]
	currentHP := mob.Character.Health
	if lastHP != currentHP {
		lastTargetHP[userId] = currentHP
		targetMutex.Unlock()

		// Send update with current HP
		handleCombatTargetUpdate(GMCPCombatTargetUpdate{
			UserId:          userId,
			TargetName:      mob.Character.Name,
			TargetHpCurrent: currentHP,
			TargetHpMax:     int(mob.Character.HealthMax.Value),
		})
	} else {
		targetMutex.Unlock()
	}
}

// checkForMobDamage checks all mobs in combat and sends damage updates if HP changed
func checkForMobDamage(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	// Import gmcp package to use SendCombatDamage
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
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
			targetMutex.Lock()
			lastHP, tracked := mobHealthTracking[mobId]
			currentHP := mob.Character.Health

			// If HP changed, send damage update
			if tracked && lastHP != currentHP {
				hpDiff := lastHP - currentHP
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
			} else {
				// Start tracking this mob
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
