// Package gmcp handles Combat Status updates for GMCP.
//
// Tracks combat state changes (entering/leaving combat) and sends updates only when state changes.
// Uses round-based checks with immediate updates on vitals changes for accurate HP snapshots.
package gmcp

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GMCPCombatStatusUpdate is sent when combat status changes (entering/leaving combat)
type GMCPCombatStatusUpdate struct {
	UserId      int
	InCombat    bool
	RoundNumber uint64 // Current round number
}

func (g GMCPCombatStatusUpdate) Type() string { return `GMCPCombatStatusUpdate` }

var (
	// stateMutex protects all the state maps
	stateMutex sync.RWMutex

	// userCombatState tracks whether user was in combat last update
	userCombatState = make(map[int]bool) // userId -> wasInCombat

	// lastRoundNumber tracks the last round number sent for each user
	lastRoundNumber = make(map[int]uint64) // userId -> roundNumber
)

func init() {
	events.RegisterListener(GMCPCombatStatusUpdate{}, handleCombatStatusUpdate)
	events.RegisterListener(events.PlayerSpawn{}, handleStatusPlayerSpawn)
	events.RegisterListener(events.NewRound{}, handleStatusNewRound)
	events.RegisterListener(events.CharacterVitalsChanged{}, handleStatusVitalsChanged)
}

func handleCombatStatusUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatStatusUpdate)
	if !typeOk {
		mudlog.Error("GMCPCombatStatus", "action", "handleCombatStatusUpdate", "error", "type assertion failed", "expectedType", "GMCPCombatStatusUpdate", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	_, valid := validateUserForGMCP(evt.UserId, "GMCPCombatStatus")
	if !valid {
		return events.Continue
	}

	// Build the payload - simplified to just combat state
	payload := map[string]interface{}{
		"in_combat": evt.InCombat,
	}

	// Only include round_number if it's set (for round-based combat)
	if evt.RoundNumber > 0 {
		payload["round_number"] = evt.RoundNumber
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.Combat.Status",
		Payload: payload,
	})

	return events.Continue
}

// handlePlayerSpawn sends initial combat status on login
func handleStatusPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		mudlog.Error("GMCPCombatStatus", "action", "handlePlayerSpawn", "error", "type assertion failed", "expectedType", "events.PlayerSpawn", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Check if user has aggro
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		mudlog.Warn("GMCPCombatStatus", "action", "handlePlayerSpawn", "issue", "user not found on spawn", "userId", evt.UserId)
		return events.Continue
	}

	inCombat := user.Character.Aggro != nil
	stateMutex.Lock()
	userCombatState[evt.UserId] = inCombat
	stateMutex.Unlock()

	// Send initial combat status
	sendCombatStatusUpdate(evt.UserId, inCombat, 0)

	return events.Continue
}

// cleanupCombatStatus removes all status tracking for a user
func cleanupCombatStatus(userId int) {
	stateMutex.Lock()
	delete(userCombatState, userId)
	delete(lastRoundNumber, userId)
	stateMutex.Unlock()
}

// handleNewRound checks for combat state changes each round
func handleStatusNewRound(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.NewRound)
	if !typeOk {
		mudlog.Error("GMCPCombatStatus", "action", "handleNewRound", "error", "type assertion failed", "expectedType", "events.NewRound", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Only check users who are in combat or were recently in combat
	stateMutex.RLock()
	trackedUsers := make([]int, 0, len(userCombatState))
	for userId := range userCombatState {
		trackedUsers = append(trackedUsers, userId)
	}
	stateMutex.RUnlock()

	// Also add any users currently in combat (in case they're not tracked yet)
	for _, userId := range GetUsersInCombat() {
		found := false
		for _, tracked := range trackedUsers {
			if tracked == userId {
				found = true
				break
			}
		}
		if !found {
			trackedUsers = append(trackedUsers, userId)
		}
	}

	for _, userId := range trackedUsers {
		user := users.GetByUserId(userId)
		if user == nil {
			stateMutex.Lock()
			if _, exists := userCombatState[userId]; exists {
				delete(userCombatState, userId)
				delete(lastRoundNumber, userId)
				mudlog.Warn("GMCPCombatStatus", "action", "handleNewRound", "issue", "user not found, cleaning up stale state", "userId", userId)
			}
			stateMutex.Unlock()
			continue
		}

		currentlyInCombat := user.Character.Aggro != nil

		stateMutex.RLock()
		wasInCombat := userCombatState[userId]
		stateMutex.RUnlock()

		// Update state and round number if needed
		if currentlyInCombat != wasInCombat || currentlyInCombat {
			stateMutex.Lock()
			if currentlyInCombat != wasInCombat {
				userCombatState[userId] = currentlyInCombat
			}
			if currentlyInCombat {
				lastRoundNumber[userId] = evt.RoundNumber
			}
			stateMutex.Unlock()
		}

		// Only send updates when combat state changes (entering/leaving combat)
		// HP updates will come from CharacterVitalsChanged events after damage
		needsUpdate := currentlyInCombat != wasInCombat

		if needsUpdate {
			// Send immediate update only for state changes
			sendCombatStatusUpdate(userId, currentlyInCombat, evt.RoundNumber)
		}
	}

	return events.Continue
}

// handleVitalsChanged sends immediate updates when character vitals change
func handleStatusVitalsChanged(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.CharacterVitalsChanged)
	if !typeOk {
		mudlog.Error("GMCPCombatStatus", "action", "handleVitalsChanged", "error", "type assertion failed", "expectedType", "events.CharacterVitalsChanged", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Only care about user changes that could affect combat
	if evt.UserId < 1 {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		stateMutex.Lock()
		if _, exists := userCombatState[evt.UserId]; exists {
			delete(userCombatState, evt.UserId)
			delete(lastRoundNumber, evt.UserId)
			mudlog.Warn("GMCPCombatStatus", "action", "handleVitalsChanged", "issue", "user not found, cleaning up stale state", "userId", evt.UserId)
		}
		stateMutex.Unlock()
		return events.Continue
	}

	currentlyInCombat := user.Character.Aggro != nil

	stateMutex.RLock()
	wasInCombat := userCombatState[evt.UserId]
	stateChanged := currentlyInCombat != wasInCombat
	roundNum := lastRoundNumber[evt.UserId]
	stateMutex.RUnlock()

	// Update state if changed
	if stateChanged {
		stateMutex.Lock()
		userCombatState[evt.UserId] = currentlyInCombat
		stateMutex.Unlock()
	}

	// Send updates on state changes AND during combat (for pre-round HP snapshot)
	if stateChanged || currentlyInCombat {

		// Send immediate update - this captures HP at start of round
		sendCombatStatusUpdate(evt.UserId, currentlyInCombat, roundNum)

		// Update cooldown tracking only on state changes
		if stateChanged {
			if currentlyInCombat {
				TrackCombatPlayer(evt.UserId)
			} else {
				UntrackCombatPlayer(evt.UserId)
			}
		}
	}

	return events.Continue
}

// sendCombatStatusUpdate sends a combat status update for a user
func sendCombatStatusUpdate(userId int, inCombat bool, roundNumber uint64) {
	user := users.GetByUserId(userId)
	if user == nil {
		mudlog.Warn("GMCPCombatStatus", "action", "sendCombatStatusUpdate", "issue", "attempted to send update for non-existent user", "userId", userId)
		return
	}

	update := GMCPCombatStatusUpdate{
		UserId:      userId,
		InCombat:    inCombat,
		RoundNumber: roundNumber,
	}

	handleCombatStatusUpdate(update)
}
