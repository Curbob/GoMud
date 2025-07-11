package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GMCPCombatDamageUpdate is sent when damage or healing occurs
type GMCPCombatDamageUpdate struct {
	UserId     int
	Amount     int    // Positive for damage, negative for healing
	DamageType string // "physical", "magical", "heal", etc.
	Source     string // Name of attacker/healer
	Target     string // Name of target
}

func (g GMCPCombatDamageUpdate) Type() string { return `GMCPCombatDamageUpdate` }

func init() {
	// Register listener for actual damage events from combat
	events.RegisterListener(events.DamageDealt{}, handleDamageDealtForGMCP)
	events.RegisterListener(events.HealingReceived{}, handleHealingReceivedForGMCP)

	// Keep the internal event for backward compatibility
	events.RegisterListener(GMCPCombatDamageUpdate{}, handleCombatDamageUpdate)
}

func handleCombatDamageUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatDamageUpdate)
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
		"amount": evt.Amount,
		"type":   evt.DamageType,
		"source": evt.Source,
		"target": evt.Target,
	}

	// Send the GMCP update immediately
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.Combat.Damage",
		Payload: payload,
	})

	return events.Continue
}

// handleDamageDealtForGMCP processes DamageDealt events and sends GMCP updates
func handleDamageDealtForGMCP(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.DamageDealt)
	if !typeOk {
		return events.Continue
	}

	// Only send to players (not mobs)
	if evt.TargetType == "player" {
		events.AddToQueue(GMCPCombatDamageUpdate{
			UserId:     evt.TargetId,
			Amount:     evt.Amount,
			DamageType: evt.DamageType,
			Source:     evt.SourceName,
			Target:     evt.TargetName,
		})
	}

	// Also send to the attacker if they're a player
	if evt.SourceType == "player" {
		events.AddToQueue(GMCPCombatDamageUpdate{
			UserId:     evt.SourceId,
			Amount:     evt.Amount,
			DamageType: evt.DamageType,
			Source:     evt.SourceName,
			Target:     evt.TargetName,
		})
	}

	return events.Continue
}

// handleHealingReceivedForGMCP processes HealingReceived events and sends GMCP updates
func handleHealingReceivedForGMCP(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.HealingReceived)
	if !typeOk {
		return events.Continue
	}

	// Only send to players
	if evt.TargetType == "player" {
		events.AddToQueue(GMCPCombatDamageUpdate{
			UserId:     evt.TargetId,
			Amount:     -evt.Amount, // Negative for healing
			DamageType: "heal",
			Source:     evt.SourceName,
			Target:     evt.TargetName,
		})
	}

	return events.Continue
}

// SendCombatDamage sends a damage/healing update
// This is exported so it can be called from combat code
func SendCombatDamage(userId int, amount int, damageType string, source string, target string) {
	// Send the update directly for immediate processing
	handleCombatDamageUpdate(GMCPCombatDamageUpdate{
		UserId:     userId,
		Amount:     amount,
		DamageType: damageType,
		Source:     source,
		Target:     target,
	})
}
