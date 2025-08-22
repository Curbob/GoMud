package characters

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// AggroState represents a serializable version of combat aggro information
// This is used during copyover to preserve active combat relationships
type AggroState struct {
	UserId        int            `yaml:"user_id,omitempty"`         // If attacking a user
	MobInstanceId int            `yaml:"mob_instance_id,omitempty"` // If attacking a mob
	Type          string         `yaml:"type"`                      // "default", "shooting", "backstab", "spellcast", "flee"
	RoundsWaiting int            `yaml:"rounds_waiting"`            // Rounds until next attack
	ExitName      string         `yaml:"exit_name,omitempty"`       // For ranged attacks
	SpellInfo     SpellAggroInfo `yaml:"spell_info,omitempty"`      // Spell information if casting
}

// SpellCastState preserves spell casting information during copyover
type SpellCastState struct {
	SpellId              string `yaml:"spell_id"`                  // ID of spell being cast
	SpellRest            string `yaml:"spell_rest,omitempty"`      // Rest of spell command
	TargetUserIds        []int  `yaml:"target_user_ids,omitempty"` // Target user IDs
	TargetMobInstanceIds []int  `yaml:"target_mob_ids,omitempty"`  // Target mob instance IDs
	RoundsLeft           int    `yaml:"rounds_left"`               // Rounds remaining to cast
}

// CombatState holds all combat-related state for a character during copyover
type CombatState struct {
	Aggro            []AggroState    `yaml:"aggro,omitempty"`              // All active aggro relationships
	PlayerDamage     map[int]int     `yaml:"player_damage,omitempty"`      // Damage dealt by each player (for kill credit)
	LastPlayerDamage uint64          `yaml:"last_player_damage,omitempty"` // Last round a player damaged this character
	SpellCasting     *SpellCastState `yaml:"spell_casting,omitempty"`      // Active spell being cast
}

// SaveCombatState creates a serializable version of the character's combat state
func (c *Character) SaveCombatState() *CombatState {
	if c.Aggro == nil && len(c.PlayerDamage) == 0 {
		return nil
	}

	state := &CombatState{
		PlayerDamage:     c.PlayerDamage,
		LastPlayerDamage: c.LastPlayerDamage,
	}

	// Convert Aggro to serializable format (single aggro, not a list)
	if c.Aggro != nil {
		state.Aggro = []AggroState{}
		aggroState := AggroState{
			UserId:        c.Aggro.UserId,
			MobInstanceId: c.Aggro.MobInstanceId,
			RoundsWaiting: c.Aggro.RoundsWaiting,
			ExitName:      c.Aggro.ExitName,
			SpellInfo:     c.Aggro.SpellInfo,
		}

		// Convert combat type to string
		switch c.Aggro.Type {
		case DefaultAttack:
			aggroState.Type = "default"
		case Shooting:
			aggroState.Type = "shooting"
		case BackStab:
			aggroState.Type = "backstab"
		case SpellCast:
			aggroState.Type = "spellcast"
			// Save spell casting details
			if c.Aggro.SpellInfo.SpellId != "" && state.SpellCasting == nil {
				state.SpellCasting = &SpellCastState{
					SpellId:              c.Aggro.SpellInfo.SpellId,
					SpellRest:            c.Aggro.SpellInfo.SpellRest,
					TargetUserIds:        c.Aggro.SpellInfo.TargetUserIds,
					TargetMobInstanceIds: c.Aggro.SpellInfo.TargetMobInstanceIds,
					RoundsLeft:           c.Aggro.RoundsWaiting,
				}
			}
		case Flee:
			aggroState.Type = "flee"
		default:
			aggroState.Type = "default"
		}

		state.Aggro = append(state.Aggro, aggroState)
	}

	return state
}

// RestoreCombatState restores the character's combat state after copyover
func (c *Character) RestoreCombatState(state *CombatState) error {
	if state == nil {
		return nil
	}

	// Restore player damage tracking
	c.PlayerDamage = state.PlayerDamage
	c.LastPlayerDamage = state.LastPlayerDamage

	// Restore aggro (single aggro, not a list)
	if len(state.Aggro) > 0 {
		// Only restore the first aggro since Character only has one Aggro
		aggroState := state.Aggro[0]
		aggro := &Aggro{
			UserId:        aggroState.UserId,
			MobInstanceId: aggroState.MobInstanceId,
			RoundsWaiting: aggroState.RoundsWaiting,
			ExitName:      aggroState.ExitName,
			SpellInfo:     aggroState.SpellInfo,
		}

		// Convert string type back to enum
		switch aggroState.Type {
		case "shooting":
			aggro.Type = Shooting
		case "backstab":
			aggro.Type = BackStab
		case "spellcast":
			aggro.Type = SpellCast
		case "flee":
			aggro.Type = Flee
		default:
			aggro.Type = DefaultAttack
		}

		c.Aggro = aggro
	}

	// Restore spell casting state
	if state.SpellCasting != nil && c.Aggro != nil && c.Aggro.Type == SpellCast {
		// Restore the spell info from saved state
		c.Aggro.SpellInfo = SpellAggroInfo{
			SpellId:              state.SpellCasting.SpellId,
			SpellRest:            state.SpellCasting.SpellRest,
			TargetUserIds:        state.SpellCasting.TargetUserIds,
			TargetMobInstanceIds: state.SpellCasting.TargetMobInstanceIds,
		}
		c.Aggro.RoundsWaiting = state.SpellCasting.RoundsLeft
	}

	return nil
}

// ValidateCombatState checks if the combat state is valid after restoration
func (c *Character) ValidateCombatState() error {
	// Note: Validation of user/mob existence must be done by the caller
	// to avoid import cycles. The users/mobs packages should call
	// ValidateAggroTargets() after restore to clean up invalid references.
	if c.Aggro != nil {
		// Log what we're validating
		if c.Aggro.UserId > 0 {
			mudlog.Debug("Combat validation", "checking", "user aggro", "userId", c.Aggro.UserId)
		}
		if c.Aggro.MobInstanceId > 0 {
			mudlog.Debug("Combat validation", "checking", "mob aggro", "mobId", c.Aggro.MobInstanceId)
		}
	}

	// Clean up old player damage entries (older than 5 minutes = 300 rounds)
	// This prevents memory leaks from old combat data
	if len(c.PlayerDamage) > 0 {
		currentRound := util.GetRoundCount()
		// If last damage was more than 300 rounds ago, clear the damage table
		if currentRound > c.LastPlayerDamage+300 {
			mudlog.Debug("Combat validation", "action", "clearing old damage", "rounds_old", currentRound-c.LastPlayerDamage)
			c.PlayerDamage = make(map[int]int)
			c.LastPlayerDamage = 0
		}
	}

	return nil
}

// PrepareCombatForCopyover prepares combat state for serialization
func (c *Character) PrepareCombatForCopyover() {
	// Save current combat state to CombatStateData field
	c.CombatStateData = c.SaveCombatState()
}

// RestoreCombatAfterCopyover restores combat state after copyover
func (c *Character) RestoreCombatAfterCopyover() error {
	if c.CombatStateData != nil {
		err := c.RestoreCombatState(c.CombatStateData)
		if err != nil {
			return err
		}
		// Clear the temporary storage
		c.CombatStateData = nil
	}
	return c.ValidateCombatState()
}

// ValidateAggroTargets checks if aggro targets still exist
// This should be called by the users package after restore
// where both users and mobs packages are available
func (c *Character) ValidateAggroTargets(userExists func(int) bool, mobExists func(int) bool) {
	if c.Aggro == nil {
		return
	}

	// Check user target
	if c.Aggro.UserId > 0 && userExists != nil {
		if !userExists(c.Aggro.UserId) {
			mudlog.Info("Combat validation", "action", "clearing invalid user aggro", "userId", c.Aggro.UserId)
			c.Aggro = nil
			return
		}
	}

	// Check mob target
	if c.Aggro.MobInstanceId > 0 && mobExists != nil {
		if !mobExists(c.Aggro.MobInstanceId) {
			mudlog.Info("Combat validation", "action", "clearing invalid mob aggro", "mobId", c.Aggro.MobInstanceId)
			c.Aggro = nil
			return
		}
	}
}
