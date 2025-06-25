package combattwitch

import (
	"sync"

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
	timer      *CooldownTimer
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
	mudlog.Info("Combat System", "module", "combat-twitch", "action", "Initialize started")

	// Register combat commands
	mudlog.Info("Combat System", "module", "combat-twitch", "action", "registering commands")
	tc.registerCommands()

	// Start the timer system
	mudlog.Info("Combat System", "module", "combat-twitch", "action", "starting timer")
	if err := tc.timer.Start(); err != nil {
		return err
	}

	tc.active = true
	mudlog.Info("Combat System", "module", "combat-twitch", "status", "initialized")
	return nil
}

// Shutdown cleanly shuts down the combat system
func (tc *TwitchCombat) Shutdown() error {
	// Get timer reference while holding lock
	tc.mutex.Lock()
	timer := tc.timer
	tc.active = false
	tc.mutex.Unlock()

	// Stop the timer system outside of lock to avoid deadlock
	if timer != nil {
		if err := timer.Stop(); err != nil {
			mudlog.Error("Failed to stop combat timer", "error", err)
		}
	}

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
	// Check if user is actually in combat
	inCombat := false
	if user := users.GetByUserId(userId); user != nil {
		inCombat = user.Character.Aggro != nil
	}

	// Send GMCP combat status update
	events.AddToQueue(gmcp.GMCPCombatStatusUpdate{
		UserId:          userId,
		CooldownSeconds: remainingSeconds,
		MaxSeconds:      maxSeconds,
		NameActive:      "Unbalanced",
		NameIdle:        "Balanced",
		InCombat:        inCombat,
		CombatStyle:     combat.GetActiveCombatSystemName(),
		RoundNumber:     0, // Not applicable for twitch combat
	})
}
