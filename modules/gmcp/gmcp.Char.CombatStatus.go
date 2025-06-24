package gmcp

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GMCPCombatStatusUpdate is sent when combat status changes
type GMCPCombatStatusUpdate struct {
	UserId          int
	CooldownSeconds float64
	MaxSeconds      float64
	NameActive      string
	NameIdle        string
	InCombat        bool
	CombatStyle     string
	RoundNumber     uint64 // Only relevant for round-based combat
}

func (g GMCPCombatStatusUpdate) Type() string { return `GMCPCombatStatusUpdate` }

func init() {
	// Register listener for combat status updates
	events.RegisterListener(GMCPCombatStatusUpdate{}, handleCombatStatusUpdate)
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
		"in_combat":    evt.InCombat,
		"combat_style": evt.CombatStyle,
	}

	// Only include round_number if it's set (for round-based combat)
	if evt.RoundNumber > 0 {
		payload["round_number"] = evt.RoundNumber
	}

	// Send the GMCP update
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.CombatStatus",
		Payload: payload,
	})

	return events.Continue
}
