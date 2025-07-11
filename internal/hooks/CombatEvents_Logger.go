package hooks

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// This file contains event listeners that log combat events for debugging and verification purposes.
// Since the freezing issue is NOT related to combat events, these are safe to use.

func init() {
	// TESTING: Defer listener registration to avoid potential init-time lock issues
	// This delays registration by 5 seconds to let the game fully initialize
	go func() {
		time.Sleep(5 * time.Second) // Let game fully initialize

		mudlog.Info("CombatEvents", "action", "Registering combat event listeners")

		// Register listeners after delay
		events.RegisterListener(events.DamageDealt{}, logDamageDealt)
		events.RegisterListener(events.AttackAvoided{}, logAttackAvoided)

		mudlog.Info("CombatEvents", "action", "Combat event listeners registered")
	}()
}

func logDamageDealt(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.DamageDealt)
	if !ok {
		return events.Continue
	}

	mudlog.Debug("DamageDealt",
		"SourceType", evt.SourceType,
		"TargetType", evt.TargetType,
		"Amount", evt.Amount,
		"IsCritical", evt.IsCritical)

	return events.Continue
}

func logAttackAvoided(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.AttackAvoided)
	if !ok {
		return events.Continue
	}

	mudlog.Debug("AttackAvoided",
		"AttackerType", evt.AttackerType,
		"DefenderType", evt.DefenderType,
		"AvoidType", evt.AvoidType)

	return events.Continue
}
