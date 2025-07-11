package gmcp

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
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
	// Register listener for combat status updates
	events.RegisterListener(GMCPCombatStatusUpdate{}, handleCombatStatusUpdate)

	// Register listeners for events that should trigger immediate status updates
	events.RegisterListener(events.PlayerSpawn{}, handlePlayerSpawn)
	events.RegisterListener(events.PlayerDespawn{}, handlePlayerDespawn)
	events.RegisterListener(events.NewRound{}, handleNewRound)
	events.RegisterListener(events.CharacterVitalsChanged{}, handleVitalsChanged)
}

func handleCombatStatusUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatStatusUpdate)
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
		// Don't cancel, just skip - the event might be useful for other listeners
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

	// Send the GMCP update
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.Combat.Status",
		Payload: payload,
	})

	return events.Continue
}

// handlePlayerSpawn sends initial combat status on login
func handlePlayerSpawn(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Check if user has aggro
	inCombat := false
	if user := users.GetByUserId(evt.UserId); user != nil {
		inCombat = user.Character.Aggro != nil
		stateMutex.Lock()
		userCombatState[evt.UserId] = inCombat
		stateMutex.Unlock()
	}

	// Send initial combat status
	sendCombatStatusUpdate(evt.UserId, inCombat, 0)

	return events.Continue
}

// handlePlayerDespawn cleans up tracking when player leaves
func handlePlayerDespawn(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		return events.Continue
	}

	// Clean up tracking maps
	stateMutex.Lock()
	delete(userCombatState, evt.UserId)
	delete(lastRoundNumber, evt.UserId)
	stateMutex.Unlock()

	// Stop cooldown tracking
	UntrackCombatPlayer(evt.UserId)

	return events.Continue
}

// handleNewRound checks for combat state changes each round
func handleNewRound(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.NewRound)
	if !typeOk {
		return events.Continue
	}

	// Check all online users for combat state changes
	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil {
			continue
		}

		currentlyInCombat := user.Character.Aggro != nil

		stateMutex.Lock()
		wasInCombat := userCombatState[userId]

		// Update state and round number
		if currentlyInCombat != wasInCombat {
			userCombatState[userId] = currentlyInCombat
		}
		if currentlyInCombat {
			lastRoundNumber[userId] = evt.RoundNumber
		}

		// Only send updates when combat state changes (entering/leaving combat)
		// HP updates will come from CharacterVitalsChanged events after damage
		needsUpdate := currentlyInCombat != wasInCombat
		stateMutex.Unlock()

		if needsUpdate {
			// Send immediate update only for state changes
			sendCombatStatusUpdate(userId, currentlyInCombat, evt.RoundNumber)

			// Update cooldown tracking
			if currentlyInCombat {
				TrackCombatPlayer(userId)
			} else {
				UntrackCombatPlayer(userId)
			}
		}
	}

	return events.Continue
}

// handleVitalsChanged sends immediate updates when character vitals change
func handleVitalsChanged(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.CharacterVitalsChanged)
	if !typeOk {
		return events.Continue
	}

	// Only care about user changes that could affect combat
	if evt.UserId < 1 {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	currentlyInCombat := user.Character.Aggro != nil

	stateMutex.Lock()
	wasInCombat := userCombatState[evt.UserId]
	stateChanged := currentlyInCombat != wasInCombat
	if stateChanged {
		userCombatState[evt.UserId] = currentlyInCombat
	}
	roundNum := lastRoundNumber[evt.UserId]
	stateMutex.Unlock()

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
	// Send directly instead of queuing to ensure consistent ordering
	update := GMCPCombatStatusUpdate{
		UserId:      userId,
		InCombat:    inCombat,
		RoundNumber: roundNumber,
	}

	// Process the update immediately
	handleCombatStatusUpdate(update)
}
