// Package gmcp handles Combat Target updates for GMCP.
//
// Event-driven target tracking that updates immediately when combat starts.
// Tracks the player's current combat target with HP updates.
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
	// targetMutexNew protects the target tracking maps
	targetMutexNew sync.RWMutex

	// userTargetsNew tracks the current target for each user
	userTargetsNew = make(map[int]*TargetInfo) // userId -> target info
)

type TargetInfo struct {
	Id        int
	Name      string
	Type      string // "mob" or "player"
	LastHP    int
	LastMaxHP int
}

func init() {
	// Register the GMCP output handler
	events.RegisterListener(GMCPCombatTargetUpdate{}, handleCombatTargetUpdate)

	// Listen for combat events
	events.RegisterListener(events.CombatStarted{}, handleTargetCombatStarted)
	events.RegisterListener(events.MobVitalsChanged{}, handleTargetVitalsChanged)
	events.RegisterListener(events.MobDeath{}, handleTargetDeath)
	events.RegisterListener(events.PlayerDeath{}, handleTargetPlayerDeath)
	events.RegisterListener(events.RoomChange{}, handleTargetRoomChangeNew)
	events.RegisterListener(events.CombatEnded{}, handleTargetCombatEnded)
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

// handleTargetCombatStarted sets target when player attacks
func handleTargetCombatStarted(e events.Event) events.ListenerReturn {
	mudlog.Info("GMCPCombatTarget", "event", "CombatStarted received in Target module")
	evt, ok := e.(events.CombatStarted)
	if !ok {
		mudlog.Error("GMCPCombatTarget", "error", "CombatStarted type assertion failed")
		return events.Continue
	}

	mudlog.Info("GMCPCombatTarget", "attackerType", evt.AttackerType, "attackerId", evt.AttackerId,
		"defenderType", evt.DefenderType, "defenderId", evt.DefenderId)

	// Only care about player attackers
	if evt.AttackerType != "player" {
		return events.Continue
	}

	// Validate user has GMCP
	_, valid := validateUserForGMCP(evt.AttackerId, "GMCPCombatTarget")
	if !valid {
		return events.Continue
	}

	// Create target info
	targetInfo := &TargetInfo{
		Id:   evt.DefenderId,
		Name: util.StripANSI(evt.DefenderName),
		Type: evt.DefenderType,
	}

	// Get initial HP if possible
	if evt.DefenderType == "mob" {
		if mob := mobs.GetInstance(evt.DefenderId); mob != nil {
			targetInfo.LastHP = mob.Character.Health
			targetInfo.LastMaxHP = int(mob.Character.HealthMax.Value)
		}
	} else if evt.DefenderType == "player" {
		if player := users.GetByUserId(evt.DefenderId); player != nil {
			targetInfo.LastHP = player.Character.Health
			targetInfo.LastMaxHP = int(player.Character.HealthMax.Value)
		}
	}

	// Update tracking
	targetMutexNew.Lock()
	userTargetsNew[evt.AttackerId] = targetInfo
	targetMutexNew.Unlock()

	// Send immediate GMCP update
	sendTargetUpdateNew(evt.AttackerId)

	mudlog.Info("GMCPCombatTarget", "action", "Target set", "userId", evt.AttackerId,
		"targetName", targetInfo.Name, "targetType", targetInfo.Type)

	return events.Continue
}

// handleTargetVitalsChanged updates target HP
func handleTargetVitalsChanged(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobVitalsChanged)
	if !ok {
		return events.Continue
	}

	targetMutexNew.RLock()
	// Check all users to see who has this mob as target
	usersToUpdate := []int{}
	for userId, target := range userTargetsNew {
		if target.Type == "mob" && target.Id == evt.MobId {
			usersToUpdate = append(usersToUpdate, userId)
		}
	}
	targetMutexNew.RUnlock()

	// Send updates
	for _, userId := range usersToUpdate {
		sendTargetUpdateNew(userId)
	}

	return events.Continue
}

// handleTargetDeath clears target when it dies
func handleTargetDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	targetMutexNew.Lock()
	usersToUpdate := []int{}
	for userId, target := range userTargetsNew {
		if target.Type == "mob" && target.Id == evt.InstanceId {
			delete(userTargetsNew, userId)
			usersToUpdate = append(usersToUpdate, userId)
		}
	}
	targetMutexNew.Unlock()

	// Send clear target updates
	for _, userId := range usersToUpdate {
		handleCombatTargetUpdate(GMCPCombatTargetUpdate{
			UserId:     userId,
			TargetName: "",
		})
	}

	return events.Continue
}

// handleTargetPlayerDeath handles player target death
func handleTargetPlayerDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDeath)
	if !ok {
		return events.Continue
	}

	targetMutexNew.Lock()
	usersToUpdate := []int{}
	for userId, target := range userTargetsNew {
		if target.Type == "player" && target.Id == evt.UserId {
			delete(userTargetsNew, userId)
			usersToUpdate = append(usersToUpdate, userId)
		}
	}
	targetMutexNew.Unlock()

	// Send clear target updates
	for _, userId := range usersToUpdate {
		handleCombatTargetUpdate(GMCPCombatTargetUpdate{
			UserId:     userId,
			TargetName: "",
		})
	}

	return events.Continue
}

// handleTargetRoomChangeNew handles when target moves away
func handleTargetRoomChangeNew(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok {
		return events.Continue
	}

	// Handle mob targets moving
	if evt.MobInstanceId != 0 {
		targetMutexNew.Lock()
		usersToUpdate := []int{}
		for userId, target := range userTargetsNew {
			if target.Type == "mob" && target.Id == evt.MobInstanceId {
				user := users.GetByUserId(userId)
				if user != nil && user.Character.RoomId != evt.ToRoomId {
					delete(userTargetsNew, userId)
					usersToUpdate = append(usersToUpdate, userId)
				}
			}
		}
		targetMutexNew.Unlock()

		for _, userId := range usersToUpdate {
			handleCombatTargetUpdate(GMCPCombatTargetUpdate{
				UserId:     userId,
				TargetName: "",
			})
		}
	}

	// Handle player targets moving
	if evt.UserId != 0 {
		targetMutexNew.Lock()
		usersToUpdate := []int{}
		for userId, target := range userTargetsNew {
			if target.Type == "player" && target.Id == evt.UserId {
				user := users.GetByUserId(userId)
				if user != nil && user.Character.RoomId != evt.ToRoomId {
					delete(userTargetsNew, userId)
					usersToUpdate = append(usersToUpdate, userId)
				}
			}
		}
		targetMutexNew.Unlock()

		for _, userId := range usersToUpdate {
			handleCombatTargetUpdate(GMCPCombatTargetUpdate{
				UserId:     userId,
				TargetName: "",
			})
		}
	}

	return events.Continue
}

// handleTargetCombatEnded clears target when combat ends
func handleTargetCombatEnded(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CombatEnded)
	if !ok || evt.EntityType != "player" {
		return events.Continue
	}

	targetMutexNew.Lock()
	delete(userTargetsNew, evt.EntityId)
	targetMutexNew.Unlock()

	// Clear target
	handleCombatTargetUpdate(GMCPCombatTargetUpdate{
		UserId:     evt.EntityId,
		TargetName: "",
	})

	return events.Continue
}

// sendTargetUpdateNew sends current target info for a user
func sendTargetUpdateNew(userId int) {
	targetMutexNew.RLock()
	target := userTargetsNew[userId]
	targetMutexNew.RUnlock()

	if target == nil {
		return
	}

	// Get current HP
	currentHP := 0
	maxHP := 0

	if target.Type == "mob" {
		if mob := mobs.GetInstance(target.Id); mob != nil {
			currentHP = mob.Character.Health
			maxHP = int(mob.Character.HealthMax.Value)
		}
	} else if target.Type == "player" {
		if player := users.GetByUserId(target.Id); player != nil {
			currentHP = player.Character.Health
			maxHP = int(player.Character.HealthMax.Value)
		}
	}

	// Only send if HP changed or initial send
	if currentHP != target.LastHP || target.LastHP == 0 {
		target.LastHP = currentHP
		target.LastMaxHP = maxHP

		handleCombatTargetUpdate(GMCPCombatTargetUpdate{
			UserId:          userId,
			TargetName:      target.Name,
			TargetHpCurrent: currentHP,
			TargetHpMax:     maxHP,
		})
	}
}

// cleanupCombatTargetNew removes all target tracking for a user
func cleanupCombatTargetNew(userId int) {
	targetMutexNew.Lock()
	delete(userTargetsNew, userId)
	targetMutexNew.Unlock()
}
