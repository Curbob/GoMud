package combattwitch

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combat"
)

// CooldownTimer implements a timer system for twitch-based combat
type CooldownTimer struct {
	*combat.BaseTimer
	combat    *TwitchCombat
	cooldowns map[string]*ActorCooldown // key: "user:123" or "mob:456"
	callbacks map[string]func()
	lastSent  map[string]float64 // Track last sent value to avoid duplicates

	// Additional mutex for cooldown-specific data
	cooldownMutex sync.RWMutex

	// Track if timer is started
	timerStarted bool
	timerMutex   sync.Mutex
}

// ActorCooldown tracks cooldown state for an actor
type ActorCooldown struct {
	NextAction  time.Time
	Callback    func()
	OffBalance  bool
	ActorId     int
	ActorType   combat.SourceTarget
	MaxDuration time.Duration // Total cooldown duration
}

// NewCooldownTimer creates a new cooldown timer
func NewCooldownTimer(c *TwitchCombat) *CooldownTimer {
	ct := &CooldownTimer{
		combat:    c,
		cooldowns: make(map[string]*ActorCooldown),
		callbacks: make(map[string]func()),
		lastSent:  make(map[string]float64),
	}

	// Create base timer with update function
	ct.BaseTimer = combat.NewBaseTimer("combat-twitch", ct.processCooldowns)

	return ct
}

// Start implements ICombatTimer
func (ct *CooldownTimer) Start() error {
	// Don't start timer until cooldowns are added
	return nil
}

// Stop implements ICombatTimer
func (ct *CooldownTimer) Stop() error {
	ct.timerMutex.Lock()
	defer ct.timerMutex.Unlock()

	if ct.timerStarted {
		ct.timerStarted = false
		return ct.BaseTimer.Stop()
	}
	return nil
}

// processCooldowns checks and executes expired cooldowns
func (ct *CooldownTimer) processCooldowns() {
	now := time.Now()

	// Process cooldowns and determine if we should stop
	shouldStop := false

	ct.cooldownMutex.Lock()
	for key, cooldown := range ct.cooldowns {
		if now.After(cooldown.NextAction) {
			// Cooldown expired
			if cooldown.OffBalance {
				cooldown.OffBalance = false

				// Notify the actor their balance is restored
				ct.combat.SendBalanceNotification(cooldown.ActorId, cooldown.ActorType)

				// Update GMCP
				if cooldown.ActorType == combat.User {
					ct.combat.SendGMCPBalanceUpdate(cooldown.ActorId, 0, 0)
				}
			}

			// Execute callback if any
			if cooldown.Callback != nil {
				go cooldown.Callback()
				cooldown.Callback = nil
			}

			// Remove expired cooldown
			delete(ct.cooldowns, key)
			delete(ct.lastSent, key)
		} else if cooldown.ActorType == combat.User {
			// Still on cooldown, send GMCP updates
			remaining := cooldown.NextAction.Sub(now).Seconds()
			maxSeconds := cooldown.MaxDuration.Seconds()

			// Always send updates during cooldown for smooth countdown
			ct.combat.SendGMCPBalanceUpdate(cooldown.ActorId, remaining, maxSeconds)
		}
	}

	// Check if we should stop the timer (no more cooldowns)
	if len(ct.cooldowns) == 0 {
		shouldStop = true
	}
	ct.cooldownMutex.Unlock()

	// Handle timer stopping outside of cooldown mutex
	if shouldStop {
		ct.timerMutex.Lock()
		if ct.timerStarted {
			ct.timerStarted = false
			ct.timerMutex.Unlock()
			// Schedule async stop to avoid blocking the update loop
			go ct.BaseTimer.Stop()
		} else {
			ct.timerMutex.Unlock()
		}
	}
}

// SetActorCooldown sets a cooldown for an actor
func (ct *CooldownTimer) SetActorCooldown(actorId int, actorType combat.SourceTarget, duration time.Duration) {
	key := ct.makeKey(actorId, actorType)

	ct.cooldownMutex.Lock()
	defer ct.cooldownMutex.Unlock()

	ct.cooldowns[key] = &ActorCooldown{
		NextAction:  time.Now().Add(duration),
		OffBalance:  true,
		ActorId:     actorId,
		ActorType:   actorType,
		MaxDuration: duration,
	}

	// Clear last sent value to force immediate update
	delete(ct.lastSent, key)

	// Start timer if not already running
	ct.timerMutex.Lock()
	if !ct.timerStarted {
		ct.timerStarted = true
		ct.BaseTimer.Start()
	}
	ct.timerMutex.Unlock()
}

// SetActorCallback sets a callback to execute when cooldown expires
func (ct *CooldownTimer) SetActorCallback(actorId int, actorType combat.SourceTarget, duration time.Duration, callback func()) {
	key := ct.makeKey(actorId, actorType)

	ct.cooldownMutex.Lock()
	defer ct.cooldownMutex.Unlock()

	cooldown, exists := ct.cooldowns[key]
	if exists {
		cooldown.Callback = callback
	} else {
		ct.cooldowns[key] = &ActorCooldown{
			NextAction:  time.Now().Add(duration),
			Callback:    callback,
			ActorId:     actorId,
			ActorType:   actorType,
			MaxDuration: duration,
		}
	}

	// Start timer if not already running
	ct.timerMutex.Lock()
	if !ct.timerStarted {
		ct.timerStarted = true
		ct.BaseTimer.Start()
	}
	ct.timerMutex.Unlock()
}

// CanPerformAction checks if an actor can perform an action
func (ct *CooldownTimer) CanPerformAction(actorId int, actorType combat.SourceTarget) bool {
	key := ct.makeKey(actorId, actorType)

	ct.cooldownMutex.RLock()
	defer ct.cooldownMutex.RUnlock()

	cooldown, exists := ct.cooldowns[key]
	if !exists {
		return true
	}

	return time.Now().After(cooldown.NextAction)
}

// GetRemainingCooldown returns remaining cooldown time
func (ct *CooldownTimer) GetRemainingCooldown(actorId int, actorType combat.SourceTarget) time.Duration {
	key := ct.makeKey(actorId, actorType)

	ct.cooldownMutex.RLock()
	defer ct.cooldownMutex.RUnlock()

	cooldown, exists := ct.cooldowns[key]
	if !exists {
		return 0
	}

	remaining := cooldown.NextAction.Sub(time.Now())
	if remaining < 0 {
		return 0
	}

	return remaining
}

// ClearActorCooldown removes an actor's cooldown
func (ct *CooldownTimer) ClearActorCooldown(actorId int, actorType combat.SourceTarget) {
	key := ct.makeKey(actorId, actorType)

	ct.cooldownMutex.Lock()
	defer ct.cooldownMutex.Unlock()

	delete(ct.cooldowns, key)
	delete(ct.lastSent, key)
}

// makeKey creates a unique key for an actor
func (ct *CooldownTimer) makeKey(actorId int, actorType combat.SourceTarget) string {
	return fmt.Sprintf("%s:%d", actorType, actorId)
}

// GetNextActionTime returns when an actor can next act
func (ct *CooldownTimer) GetNextActionTime(actorId int, actorType combat.SourceTarget) time.Time {
	key := ct.makeKey(actorId, actorType)

	ct.cooldownMutex.RLock()
	defer ct.cooldownMutex.RUnlock()

	cooldown, exists := ct.cooldowns[key]
	if !exists {
		return time.Now()
	}

	return cooldown.NextAction
}

// RegisterActor ensures an actor is tracked (no-op for cooldown timer)
func (ct *CooldownTimer) RegisterActor(actorId int, actorType combat.SourceTarget) {
	// Cooldown timer doesn't need to pre-register actors
	// They are tracked when cooldowns are set
}
