package combat

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// ICombatSystem defines the interface for modular combat implementations
type ICombatSystem interface {
	// Initialize sets up the combat system
	Initialize() error

	// Shutdown cleanly shuts down the combat system
	Shutdown() error

	// GetName returns the name of this combat system
	GetName() string

	// ProcessCombatRound handles combat for all actors (called by round system)
	ProcessCombatRound()

	// GetCalculator returns the combat calculator
	GetCalculator() ICombatCalculator

	// GetTimer returns the combat timer (may be nil for round-based combat)
	GetTimer() ICombatTimer
}

// ICombatCalculator defines combat calculation methods
type ICombatCalculator interface {
	// CalculateHitChance determines if an attack hits
	CalculateHitChance(attacker, defender *characters.Character) bool

	// CalculateDamage computes damage for an attack
	CalculateDamage(attacker, defender *characters.Character, weapon *items.Item) int

	// CalculateCriticalChance determines if an attack is critical
	CalculateCriticalChance(attacker, defender *characters.Character) bool

	// CalculateDefense computes defensive value
	CalculateDefense(defender *characters.Character) int

	// CalculateInitiative determines action order
	CalculateInitiative(actor *characters.Character) int

	// CalculateAttackCount determines number of attacks per round
	CalculateAttackCount(attacker, defender *characters.Character) int

	// PowerRanking calculates relative power between actors
	PowerRanking(attacker, defender *characters.Character) float64
}

// ICombatTimer defines minimal timing system interface for combat
type ICombatTimer interface {
	// Start begins the timer system
	Start() error

	// Stop halts the timer system
	Stop() error
}


// CombatAction represents a queued combat action
type CombatAction struct {
	AttackerId   int
	AttackerType SourceTarget
	DefenderId   int
	DefenderType SourceTarget
	ActionType   characters.AggroType
	SpellId      int
	ExitName     string
	Timestamp    time.Time
}

