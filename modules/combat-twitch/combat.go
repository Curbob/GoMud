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

	// Track last target name per user for persistent targeting
	userTargets     map[int]string // userId -> target name
	userTargetMutex sync.RWMutex
}

// NewTwitchCombat creates a new twitch-based combat system
func NewTwitchCombat(plug *plugins.Plugin) *TwitchCombat {
	tc := &TwitchCombat{
		plug:        plug,
		calculator:  NewTwitchCalculator(),
		active:      false,
		userTargets: make(map[int]string),
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

	// Unregister commands
	tc.unregisterCommands()

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
	// targetHpCurrent := 0  // Not used in current GMCP structure
	// targetHpMax := 0      // Not used in current GMCP structure

	user := users.GetByUserId(userId)
	if user != nil {
		inCombat = user.Character.Aggro != nil

		// If in combat, get target HP (not used in current GMCP structure)
		// if inCombat && user.Character.Aggro.MobInstanceId > 0 {
		// 	if targetMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); targetMob != nil {
		// 		targetHpCurrent = targetMob.Character.Health
		// 		targetHpMax = targetMob.Character.HealthMax.ValueAdj
		// 	}
		// }
	}

	// Get the user's current target (not used in current GMCP structure)
	// target := tc.GetUserTarget(userId)

	// Send GMCP combat status update
	events.AddToQueue(gmcp.GMCPCombatStatusUpdate{
		UserId:      userId,
		InCombat:    inCombat,
		RoundNumber: 0, // Not applicable for twitch combat
	})

	// Send GMCP cooldown update if in combat
	if inCombat {
		events.AddToQueue(gmcp.GMCPCombatCooldownUpdate{
			UserId:          userId,
			CooldownSeconds: remainingSeconds,
			MaxSeconds:      maxSeconds,
			NameActive:      "Unbalanced",
			NameIdle:        "Balanced",
		})
	}
}

// SetUserTarget stores the last target name for a user
func (tc *TwitchCombat) SetUserTarget(userId int, targetName string) {
	tc.userTargetMutex.Lock()
	if targetName == "" {
		delete(tc.userTargets, userId)
	} else {
		tc.userTargets[userId] = targetName
	}
	tc.userTargetMutex.Unlock()

	// Send GMCP update with new target
	// Get current cooldown to include in update
	remaining := tc.timer.GetRemainingCooldown(userId, combat.User).Seconds()
	maxDuration := float64(0)
	if remaining > 0 {
		// If on cooldown, get max duration
		// For simplicity, we'll just send the update with current remaining time
		maxDuration = remaining // This isn't perfect but avoids accessing internal timer state
	}

	tc.SendGMCPBalanceUpdate(userId, remaining, maxDuration)
}

// GetUserTarget retrieves the last target name for a user
func (tc *TwitchCombat) GetUserTarget(userId int) string {
	tc.userTargetMutex.RLock()
	defer tc.userTargetMutex.RUnlock()

	return tc.userTargets[userId]
}

// ClearUserTarget removes the stored target for a user
func (tc *TwitchCombat) ClearUserTarget(userId int) {
	tc.userTargetMutex.Lock()
	delete(tc.userTargets, userId)
	tc.userTargetMutex.Unlock()

	// Send GMCP update with cleared target
	remaining := tc.timer.GetRemainingCooldown(userId, combat.User).Seconds()
	maxDuration := float64(0)
	if remaining > 0 {
		maxDuration = remaining
	}

	tc.SendGMCPBalanceUpdate(userId, remaining, maxDuration)
}

// SendCombatUpdate sends GMCP updates for a player involved in combat
// This should be called after any combat action that might change HP
func (tc *TwitchCombat) SendCombatUpdate(userId int) {
	// Get current cooldown for the user
	remaining := tc.timer.GetRemainingCooldown(userId, combat.User).Seconds()
	maxDuration := float64(0)
	if remaining > 0 {
		maxDuration = remaining
	}

	tc.SendGMCPBalanceUpdate(userId, remaining, maxDuration)
}
