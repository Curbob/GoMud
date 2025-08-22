package combat

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// ActiveCombatState represents an active combat relationship
type ActiveCombatState struct {
	AttackerId   int    `yaml:"attacker_id"`   // User or mob ID
	AttackerType string `yaml:"attacker_type"` // "user" or "mob"
	DefenderId   int    `yaml:"defender_id"`   // User or mob ID
	DefenderType string `yaml:"defender_type"` // "user" or "mob"
	RoundStarted uint64 `yaml:"round_started"` // Round when combat started
	LastAction   uint64 `yaml:"last_action"`   // Last round action was taken
	CombatType   string `yaml:"combat_type"`   // Type of combat
	RoomId       int    `yaml:"room_id"`       // Room where combat is happening
}

// SpellCastInfo tracks active spell casting
type SpellCastInfo struct {
	CasterId     int      `yaml:"caster_id"`     // User casting the spell
	SpellId      string   `yaml:"spell_id"`      // Spell being cast
	Targets      []string `yaml:"targets"`       // Target names
	RoundsLeft   int      `yaml:"rounds_left"`   // Rounds remaining
	StartedRound uint64   `yaml:"started_round"` // When casting started
	RoomId       int      `yaml:"room_id"`       // Room where casting
}

// CombatStateRegistry tracks all active combat in the world
type CombatStateRegistry struct {
	mu            sync.RWMutex
	ActiveCombats map[string]*ActiveCombatState // key: "attacker:type:id:defender:type:id"
	SpellCasts    map[int]*SpellCastInfo        // key: userId
	RoundNumber   uint64                        // Current round number
}

var (
	stateRegistry *CombatStateRegistry
	stateOnce     sync.Once
)

// GetStateRegistry returns the singleton combat state registry instance
func GetStateRegistry() *CombatStateRegistry {
	stateOnce.Do(func() {
		stateRegistry = &CombatStateRegistry{
			ActiveCombats: make(map[string]*ActiveCombatState),
			SpellCasts:    make(map[int]*SpellCastInfo),
			RoundNumber:   util.GetRoundCount(),
		}
	})
	return stateRegistry
}

// UpdateRound updates the current round number
func (r *CombatStateRegistry) UpdateRound(round uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.RoundNumber = round
}

// RegisterCombat registers a new combat relationship
func (r *CombatStateRegistry) RegisterCombat(attackerId int, attackerType string, defenderId int, defenderType string, combatType string, roomId int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeCombatStateKey(attackerId, attackerType, defenderId, defenderType)

	if existing, exists := r.ActiveCombats[key]; exists {
		// Update existing combat
		existing.LastAction = r.RoundNumber
		existing.CombatType = combatType
		existing.RoomId = roomId
	} else {
		// Create new combat entry
		r.ActiveCombats[key] = &ActiveCombatState{
			AttackerId:   attackerId,
			AttackerType: attackerType,
			DefenderId:   defenderId,
			DefenderType: defenderType,
			RoundStarted: r.RoundNumber,
			LastAction:   r.RoundNumber,
			CombatType:   combatType,
			RoomId:       roomId,
		}
	}
}

// UnregisterCombat removes a combat relationship
func (r *CombatStateRegistry) UnregisterCombat(attackerId int, attackerType string, defenderId int, defenderType string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := makeCombatStateKey(attackerId, attackerType, defenderId, defenderType)
	delete(r.ActiveCombats, key)
}

// RegisterSpellCast registers an active spell being cast
func (r *CombatStateRegistry) RegisterSpellCast(casterId int, spellId string, targets []string, roundsLeft int, roomId int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.SpellCasts[casterId] = &SpellCastInfo{
		CasterId:     casterId,
		SpellId:      spellId,
		Targets:      targets,
		RoundsLeft:   roundsLeft,
		StartedRound: r.RoundNumber,
		RoomId:       roomId,
	}
}

// UnregisterSpellCast removes a spell cast entry
func (r *CombatStateRegistry) UnregisterSpellCast(casterId int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.SpellCasts, casterId)
}

// GetSpellCast returns the spell cast info for a user
func (r *CombatStateRegistry) GetSpellCast(casterId int) *SpellCastInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.SpellCasts[casterId]
}

// IsInCombat checks if an entity is in combat
func (r *CombatStateRegistry) IsInCombat(entityId int, entityType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, combat := range r.ActiveCombats {
		if (combat.AttackerId == entityId && combat.AttackerType == entityType) ||
			(combat.DefenderId == entityId && combat.DefenderType == entityType) {
			// Check if combat is still active (within last 10 rounds)
			if r.RoundNumber-combat.LastAction < 10 {
				return true
			}
		}
	}
	return false
}

// GetCombatsInRoom returns all active combats in a room
func (r *CombatStateRegistry) GetCombatsInRoom(roomId int) []*ActiveCombatState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var combats []*ActiveCombatState
	for _, combat := range r.ActiveCombats {
		if combat.RoomId == roomId && r.RoundNumber-combat.LastAction < 10 {
			combats = append(combats, combat)
		}
	}
	return combats
}

// CleanupStaleCombats removes combats that haven't had action in 30 rounds
func (r *CombatStateRegistry) CleanupStaleCombats() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, combat := range r.ActiveCombats {
		if r.RoundNumber-combat.LastAction > 30 {
			delete(r.ActiveCombats, key)
			mudlog.Debug("Combat state registry cleanup", "removed", key, "lastAction", combat.LastAction, "currentRound", r.RoundNumber)
		}
	}
}

// CombatStateRegistryData represents the serializable state
type CombatStateRegistryData struct {
	ActiveCombats map[string]*ActiveCombatState `yaml:"active_combats"`
	SpellCasts    map[int]*SpellCastInfo        `yaml:"spell_casts"`
	RoundNumber   uint64                        `yaml:"round_number"`
}

// SaveState serializes the combat registry state
func (r *CombatStateRegistry) SaveState() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state := CombatStateRegistryData{
		ActiveCombats: r.ActiveCombats,
		SpellCasts:    r.SpellCasts,
		RoundNumber:   r.RoundNumber,
	}

	return yaml.Marshal(state)
}

// RestoreState deserializes and restores the combat registry state
func (r *CombatStateRegistry) RestoreState(data []byte) error {
	var state CombatStateRegistryData
	if err := yaml.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal combat state registry: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.ActiveCombats = state.ActiveCombats
	r.SpellCasts = state.SpellCasts
	r.RoundNumber = state.RoundNumber

	// Clean up any stale entries immediately
	for key, combat := range r.ActiveCombats {
		if r.RoundNumber-combat.LastAction > 30 {
			delete(r.ActiveCombats, key)
		}
	}

	mudlog.Info("Combat state registry restored", "combats", len(r.ActiveCombats), "spells", len(r.SpellCasts), "round", r.RoundNumber)
	return nil
}

// makeCombatStateKey creates a unique key for a combat relationship
func makeCombatStateKey(attackerId int, attackerType string, defenderId int, defenderType string) string {
	return fmt.Sprintf("%s:%d:%s:%d", attackerType, attackerId, defenderType, defenderId)
}

// GetAllCombats returns all active combats (for debugging/admin)
func (r *CombatStateRegistry) GetAllCombats() map[string]*ActiveCombatState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	combats := make(map[string]*ActiveCombatState)
	for k, v := range r.ActiveCombats {
		combats[k] = v
	}
	return combats
}

// GetAllSpellCasts returns all active spell casts (for debugging/admin)
func (r *CombatStateRegistry) GetAllSpellCasts() map[int]*SpellCastInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	spells := make(map[int]*SpellCastInfo)
	for k, v := range r.SpellCasts {
		spells[k] = v
	}
	return spells
}
