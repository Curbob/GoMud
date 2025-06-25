package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// SwitchCombatSystemEvent is triggered when a combat system switch is requested
type SwitchCombatSystemEvent struct {
	NewSystem      string
	OldSystem      string
	RequestingUser int
	SavedStates    []CombatState
	SavedMobStates map[int]*characters.Aggro
}

// CombatState holds saved combat state for a user
type CombatState struct {
	UserId int
	Aggro  *characters.Aggro
}

// Type returns the event type
func (e SwitchCombatSystemEvent) Type() string {
	return "combat.switch_system"
}
