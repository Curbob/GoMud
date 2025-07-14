// Package gmcp handles Combat Event notifications for GMCP.
//
// Stateless event transformer that converts internal combat events into GMCP messages.
// Each event (CombatStarted, AttackMissed, etc.) is meaningful and sent immediately.
package gmcp

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GMCPCombatEvent is a generic GMCP event for combat notifications
type GMCPCombatEvent struct {
	UserId    int
	EventType string
	Data      map[string]interface{}
}

func (g GMCPCombatEvent) Type() string { return `GMCPCombatEvent` }

func init() {
	// Register listener for GMCP combat events
	events.RegisterListener(GMCPCombatEvent{}, handleCombatEvent)

	// Register listeners for all combat events
	events.RegisterListener(events.CombatStarted{}, handleCombatStarted)
	events.RegisterListener(events.CombatEnded{}, handleCombatEnded)
	events.RegisterListener(events.DamageDealt{}, handleDamageDealt)
	events.RegisterListener(events.AttackAvoided{}, handleAttackAvoided)
	events.RegisterListener(events.CombatantFled{}, handleCombatantFled)

}

// handleCombatEvent sends GMCP combat events
func handleCombatEvent(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatEvent)
	if !typeOk {
		mudlog.Error("GMCPCombatEvents", "action", "handleCombatEvent", "error", "type assertion failed", "expectedType", "GMCPCombatEvent", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	_, valid := validateUserForGMCP(evt.UserId, "GMCPCombatEvents")
	if !valid {
		return events.Continue
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  fmt.Sprintf("Char.Combat.%s", evt.EventType),
		Payload: evt.Data,
	})

	return events.Continue
}

// handleCombatStarted sends GMCP notification when combat begins
func handleCombatStarted(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.CombatStarted)
	if !typeOk {
		mudlog.Error("GMCPCombatEvents", "action", "handleCombatStarted", "error", "type assertion failed", "expectedType", "events.CombatStarted", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Send to attacker if they're a player
	if evt.AttackerType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.AttackerId,
			EventType: "Started",
			Data: map[string]interface{}{
				"role":        "attacker",
				"targetId":    evt.DefenderId,
				"targetType":  evt.DefenderType,
				"targetName":  util.StripANSI(evt.DefenderName),
				"initiatedBy": evt.InitiatedBy,
			},
		})
	}

	// Send to defender if they're a player
	if evt.DefenderType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.DefenderId,
			EventType: "Started",
			Data: map[string]interface{}{
				"role":         "defender",
				"attackerId":   evt.AttackerId,
				"attackerType": evt.AttackerType,
				"attackerName": util.StripANSI(evt.AttackerName),
				"initiatedBy":  evt.InitiatedBy,
			},
		})
	}

	return events.Continue
}

// handleCombatEnded sends GMCP notification when combat ends
func handleCombatEnded(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.CombatEnded)
	if !typeOk {
		mudlog.Error("GMCPCombatEvents", "action", "handleCombatEnded", "error", "type assertion failed", "expectedType", "events.CombatEnded", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Only send to players
	if evt.EntityType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.EntityId,
			EventType: "Ended",
			Data: map[string]interface{}{
				"reason":   evt.Reason,
				"duration": evt.Duration,
			},
		})
	}

	return events.Continue
}

// handleDamageDealt sends GMCP notification for damage events
func handleDamageDealt(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.DamageDealt)
	if !typeOk {
		mudlog.Error("GMCPCombatEvents", "action", "handleDamageDealt", "error", "type assertion failed", "expectedType", "events.DamageDealt", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Send to source if they're a player
	if evt.SourceType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.SourceId,
			EventType: "DamageDealt",
			Data: map[string]interface{}{
				"targetId":      evt.TargetId,
				"targetType":    evt.TargetType,
				"targetName":    util.StripANSI(evt.TargetName),
				"amount":        evt.Amount,
				"damageType":    evt.DamageType,
				"weaponName":    util.StripANSI(evt.WeaponName),
				"spellName":     util.StripANSI(evt.SpellName),
				"isCritical":    evt.IsCritical,
				"isKillingBlow": evt.IsKillingBlow,
			},
		})
	}

	// Send to target if they're a player
	if evt.TargetType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.TargetId,
			EventType: "DamageReceived",
			Data: map[string]interface{}{
				"sourceId":      evt.SourceId,
				"sourceType":    evt.SourceType,
				"sourceName":    util.StripANSI(evt.SourceName),
				"amount":        evt.Amount,
				"damageType":    evt.DamageType,
				"weaponName":    util.StripANSI(evt.WeaponName),
				"spellName":     util.StripANSI(evt.SpellName),
				"isCritical":    evt.IsCritical,
				"isKillingBlow": evt.IsKillingBlow,
			},
		})
	}

	return events.Continue
}

// handleAttackAvoided sends GMCP notification for avoided attacks
func handleAttackAvoided(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.AttackAvoided)
	if !typeOk {
		mudlog.Error("GMCPCombatEvents", "action", "handleAttackAvoided", "error", "type assertion failed", "expectedType", "events.AttackAvoided", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Send to attacker if they're a player
	if evt.AttackerType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.AttackerId,
			EventType: "AttackMissed",
			Data: map[string]interface{}{
				"defenderId":   evt.DefenderId,
				"defenderType": evt.DefenderType,
				"defenderName": util.StripANSI(evt.DefenderName),
				"avoidType":    evt.AvoidType,
				"weaponName":   util.StripANSI(evt.WeaponName),
			},
		})
	}

	// Send to defender if they're a player
	if evt.DefenderType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.DefenderId,
			EventType: "AttackAvoided",
			Data: map[string]interface{}{
				"attackerId":   evt.AttackerId,
				"attackerType": evt.AttackerType,
				"attackerName": util.StripANSI(evt.AttackerName),
				"avoidType":    evt.AvoidType,
				"weaponName":   util.StripANSI(evt.WeaponName),
			},
		})
	}

	return events.Continue
}

// handleCombatantFled sends GMCP notification when someone flees
func handleCombatantFled(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.CombatantFled)
	if !typeOk {
		mudlog.Error("GMCPCombatEvents", "action", "handleCombatantFled", "error", "type assertion failed", "expectedType", "events.CombatantFled", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Only send to players
	if evt.EntityType == "player" {
		events.AddToQueue(GMCPCombatEvent{
			UserId:    evt.EntityId,
			EventType: "Fled",
			Data: map[string]interface{}{
				"direction":   evt.Direction,
				"success":     evt.Success,
				"preventedBy": evt.PreventedBy,
			},
		})
	}

	return events.Continue
}
