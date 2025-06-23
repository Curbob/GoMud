package gmcp

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GMCPCombatCooldownUpdate is sent when combat cooldown status changes
type GMCPCombatCooldownUpdate struct {
	UserId          int
	CooldownSeconds float64
	MaxSeconds      float64
	NameActive      string
	NameIdle        string
}

func (g GMCPCombatCooldownUpdate) Type() string { return `GMCPCombatCooldownUpdate` }

func init() {
	// Register listener for combat cooldown updates
	events.RegisterListener(GMCPCombatCooldownUpdate{}, handleCombatCooldownUpdate)
}

func handleCombatCooldownUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatCooldownUpdate)
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
		return events.Cancel
	}

	// Round to 1 decimal place
	cooldown := math.Round(evt.CooldownSeconds*10) / 10
	maxCooldown := math.Round(evt.MaxSeconds*10) / 10

	// Build the payload
	payload := map[string]interface{}{
		"cooldown":     cooldown,
		"max_cooldown": maxCooldown,
		"name_active":  evt.NameActive,
		"name_idle":    evt.NameIdle,
	}

	// Send the GMCP update
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.CombatCooldown",
		Payload: payload,
	})

	return events.Continue
}
