package combatrounds

import (
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// RoundBasedCombat implements the traditional round-based combat system
type RoundBasedCombat struct {
	plug       *plugins.Plugin
	calculator combat.ICombatCalculator
	timer      *RoundBasedTimer
	active     bool
}

// NewRoundBasedCombat creates a new round-based combat system
func NewRoundBasedCombat(plug *plugins.Plugin) *RoundBasedCombat {
	rbc := &RoundBasedCombat{
		plug:       plug,
		calculator: NewRoundBasedCalculator(),
		active:     false,
	}
	rbc.timer = NewRoundBasedTimer(rbc)
	return rbc
}

// Initialize sets up the combat system
func (rbc *RoundBasedCombat) Initialize() error {
	// Register combat commands
	rbc.registerCommands()

	// Start the timer
	if err := rbc.timer.Start(); err != nil {
		return err
	}

	rbc.active = true
	if mudlog.IsInitialized() {
		mudlog.Info("Combat System", "module", "combat-rounds", "status", "initialized")
	}
	return nil
}

// Shutdown cleanly shuts down the combat system
func (rbc *RoundBasedCombat) Shutdown() error {
	// Stop the timer
	if err := rbc.timer.Stop(); err != nil {
		return err
	}

	rbc.active = false
	if mudlog.IsInitialized() {
		mudlog.Info("Combat System", "module", "combat-rounds", "status", "shutdown")
	}
	return nil
}

// GetName returns the name of this combat system
func (rbc *RoundBasedCombat) GetName() string {
	return "combat-rounds"
}

// ProcessCombatRound handles combat for all actors
func (rbc *RoundBasedCombat) ProcessCombatRound() {
	// Get the current round number
	roundNumber := util.GetRoundCount()

	// Process combat for this round
	rbc.processCombatRound(roundNumber)
}

// GetCalculator returns the combat calculator
func (rbc *RoundBasedCombat) GetCalculator() combat.ICombatCalculator {
	return rbc.calculator
}

// GetTimer returns the combat timer
func (rbc *RoundBasedCombat) GetTimer() combat.ICombatTimer {
	return rbc.timer
}
