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

// ICombatTimer defines timing system for combat
type ICombatTimer interface {
	// Start begins the timer system
	Start() error

	// Stop halts the timer system
	Stop() error

	// RegisterActor adds an actor to the timing system
	RegisterActor(actorId int, actorType SourceTarget, callback func())

	// UnregisterActor removes an actor from the timing system
	UnregisterActor(actorId int, actorType SourceTarget)

	// SetActorCooldown sets cooldown for an actor's next action
	SetActorCooldown(actorId int, actorType SourceTarget, duration time.Duration)

	// GetNextActionTime returns when an actor can act next
	GetNextActionTime(actorId int, actorType SourceTarget) time.Time

	// CanPerformAction checks if an actor can act now
	CanPerformAction(actorId int, actorType SourceTarget) bool
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

// CombatState tracks combat state for an actor
type CombatState struct {
	InCombat      bool
	Opponents     []CombatOpponent
	LastAction    time.Time
	NextAction    time.Time
	CurrentAction *CombatAction
}

// CombatOpponent tracks an opponent in combat
type CombatOpponent struct {
	ActorId      int
	ActorType    SourceTarget
	AggroLevel   int
	LastAttacked time.Time
}
