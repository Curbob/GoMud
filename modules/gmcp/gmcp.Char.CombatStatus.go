package gmcp

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/combat"
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
	Target          string // Current combat target name
	TargetHpCurrent int    // Current HP of target (only when in combat)
	TargetHpMax     int    // Max HP of target (only when in combat)
}

func (g GMCPCombatStatusUpdate) Type() string { return `GMCPCombatStatusUpdate` }

func init() {
	// Register listener for combat status updates
	events.RegisterListener(GMCPCombatStatusUpdate{}, handleCombatStatusUpdate)
	// Register listener for player spawn to send initial status
	events.RegisterListener(events.PlayerSpawn{}, handlePlayerSpawn)
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

	// Round to 1 decimal place and format as string to ensure x.y format
	cooldown := math.Round(evt.CooldownSeconds*10) / 10
	maxCooldown := math.Round(evt.MaxSeconds*10) / 10

	// Build the payload
	// Use formatted strings for cooldowns to ensure they always have .x format
	payload := map[string]interface{}{
		"cooldown":     fmt.Sprintf("%.1f", cooldown),
		"max_cooldown": fmt.Sprintf("%.1f", maxCooldown),
		"name_active":  evt.NameActive,
		"name_idle":    evt.NameIdle,
		"in_combat":    evt.InCombat,
		"combat_style": evt.CombatStyle,
	}

	// Only include round_number if it's set (for round-based combat)
	if evt.RoundNumber > 0 {
		payload["round_number"] = evt.RoundNumber
	}

	// Always include target, use "None" if not set
	if evt.Target != "" {
		payload["target"] = evt.Target
	} else {
		payload["target"] = "None"
	}

	// Always include target HP fields as strings
	if evt.InCombat && evt.TargetHpMax > 0 {
		payload["target_hp_current"] = fmt.Sprintf("%d", evt.TargetHpCurrent)
		payload["target_hp_max"] = fmt.Sprintf("%d", evt.TargetHpMax)
	} else {
		payload["target_hp_current"] = ""
		payload["target_hp_max"] = ""
	}

	// Send the GMCP update
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.CombatStatus",
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
	}

	// Get active combat system name and set appropriate names
	combatStyle := combat.GetActiveCombatSystemName()
	if combatStyle == "" {
		combatStyle = "default"
	}

	// Set names based on combat system
	var nameActive, nameIdle string
	switch combatStyle {
	case "combat-twitch":
		nameActive = "Unbalanced"
		nameIdle = "Balanced"
	case "combat-rounds":
		nameActive = "Combat Round"
		nameIdle = "Ready"
	default:
		// For default or unknown systems
		nameActive = "In Combat"
		nameIdle = "Ready"
	}

	// Send initial combat status
	events.AddToQueue(GMCPCombatStatusUpdate{
		UserId:          evt.UserId,
		CooldownSeconds: 0,
		MaxSeconds:      0,
		NameActive:      nameActive,
		NameIdle:        nameIdle,
		InCombat:        inCombat,
		CombatStyle:     combatStyle,
		RoundNumber:     0,
		Target:          "", // No target on spawn initially
		TargetHpCurrent: 0,
		TargetHpMax:     0,
	})

	return events.Continue
}
