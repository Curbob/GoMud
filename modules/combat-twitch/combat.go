package combattwitch

import (
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
)

// TwitchCombat implements a cooldown-based combat system
type TwitchCombat struct {
	plug       *plugins.Plugin
	calculator combat.ICombatCalculator
	timer      combat.ICombatTimer
	active     bool
	mutex      sync.RWMutex
}

// NewTwitchCombat creates a new twitch-based combat system
func NewTwitchCombat(plug *plugins.Plugin) *TwitchCombat {
	tc := &TwitchCombat{
		plug:       plug,
		calculator: NewTwitchCalculator(),
		active:     false,
	}
	tc.timer = NewCooldownTimer(tc)
	return tc
}

// Initialize sets up the combat system
func (tc *TwitchCombat) Initialize() error {
	// Register combat commands
	tc.registerCommands()

	// Start the timer system
	if err := tc.timer.Start(); err != nil {
		return err
	}

	tc.active = true
	mudlog.Info("Combat System", "module", "combat-twitch", "status", "initialized")
	return nil
}

// Shutdown cleanly shuts down the combat system
func (tc *TwitchCombat) Shutdown() error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	// Stop the timer system
	if tc.timer != nil {
		tc.timer.Stop()
	}

	tc.active = false
	mudlog.Info("Combat System", "module", "combat-twitch", "status", "shutdown")
	return nil
}

// GetName returns the name of this combat system
func (tc *TwitchCombat) GetName() string {
	return "combat-twitch"
}

// ProcessCombatRound is called each round but twitch combat doesn't use rounds
func (tc *TwitchCombat) ProcessCombatRound() {
	// Twitch combat doesn't process rounds - it's event-driven
	// Actions happen when cooldowns expire
}

// GetCalculator returns the combat calculator
func (tc *TwitchCombat) GetCalculator() combat.ICombatCalculator {
	return tc.calculator
}

// GetTimer returns the combat timer
func (tc *TwitchCombat) GetTimer() combat.ICombatTimer {
	return tc.timer
}

// ExecuteCombatAction executes a combat action if cooldown allows
func (tc *TwitchCombat) ExecuteCombatAction(actorId int, actorType combat.SourceTarget, action *combat.CombatAction) error {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	if !tc.active {
		return nil
	}

	// Check if actor can perform action
	if !tc.timer.CanPerformAction(actorId, actorType) {
		return nil // Silently fail if on cooldown
	}

	// Execute the action based on type
	// This is where the actual combat logic would go
	// For now, this is a placeholder

	// Set cooldown based on action type and weapon speed
	cooldownDuration := tc.calculateCooldown(action)
	tc.timer.SetActorCooldown(actorId, actorType, cooldownDuration)

	return nil
}

// calculateCooldown determines cooldown duration based on action and weapon
func (tc *TwitchCombat) calculateCooldown(action *combat.CombatAction) time.Duration {
	// Base cooldown
	baseCooldown := 2 * time.Second

	// Modify based on action type
	switch action.ActionType {
	case characters.DefaultAttack:
		// Weapon speed would modify this
		return baseCooldown
	case characters.SpellCast:
		// Spell cast time would modify this
		return 3 * time.Second
	case characters.Flee:
		// Flee has shorter cooldown
		return 1 * time.Second
	default:
		return baseCooldown
	}
}

// SendBalanceNotification sends a balance regained message to a user
func (tc *TwitchCombat) SendBalanceNotification(actorId int, actorType combat.SourceTarget) {
	if actorType == combat.User {
		if user := users.GetByUserId(actorId); user != nil {
			user.SendText(`<ansi fg="green">You are balanced.</ansi>`)
		}
	}
}

// SendGMCPBalanceUpdate sends GMCP balance information
func (tc *TwitchCombat) SendGMCPBalanceUpdate(userId int, remainingSeconds float64, maxSeconds float64) {
	// Send GMCP combat cooldown update
	events.AddToQueue(gmcp.GMCPCombatCooldownUpdate{
		UserId:          userId,
		CooldownSeconds: remainingSeconds,
		MaxSeconds:      maxSeconds,
		NameActive:      "Unbalanced",
		NameIdle:        "Balanced",
	})
}
