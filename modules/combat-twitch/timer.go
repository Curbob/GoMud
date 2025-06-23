package combattwitch

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// CooldownTimer implements a timer system for twitch-based combat
type CooldownTimer struct {
	combat    *TwitchCombat
	cooldowns map[string]*ActorCooldown // key: "user:123" or "mob:456"
	callbacks map[string]func()
	lastSent  map[string]float64 // Track last sent value to avoid duplicates
	mutex     sync.RWMutex
	ticker    *time.Ticker
	stopChan  chan bool
	running   bool
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
func NewCooldownTimer(combat *TwitchCombat) *CooldownTimer {
	return &CooldownTimer{
		combat:    combat,
		cooldowns: make(map[string]*ActorCooldown),
		callbacks: make(map[string]func()),
		lastSent:  make(map[string]float64),
		stopChan:  make(chan bool),
	}
}

// Start begins the timer system
func (ct *CooldownTimer) Start() error {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	if ct.running {
		return nil
	}

	// Check cooldowns every 100ms for responsive combat
	// TODO: Make this configurable via module config
	ct.ticker = time.NewTicker(100 * time.Millisecond)
	ct.running = true

	go ct.processCooldowns()

	mudlog.Info("CooldownTimer", "status", "started")
	return nil
}

// Stop halts the timer system
func (ct *CooldownTimer) Stop() error {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	if !ct.running {
		return nil
	}

	ct.running = false
	ct.stopChan <- true
	ct.ticker.Stop()

	mudlog.Info("CooldownTimer", "status", "stopped")
	return nil
}

// RegisterActor adds an actor to the timing system
func (ct *CooldownTimer) RegisterActor(actorId int, actorType combat.SourceTarget, callback func()) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	key := ct.makeKey(actorId, actorType)
	// Only create if doesn't exist
	if _, exists := ct.cooldowns[key]; !exists {
		ct.cooldowns[key] = &ActorCooldown{
			NextAction: time.Now(), // Can act immediately when registered
			Callback:   callback,
			OffBalance: false,
			ActorId:    actorId,
			ActorType:  actorType,
		}
	}
}

// UnregisterActor removes an actor from the timing system
func (ct *CooldownTimer) UnregisterActor(actorId int, actorType combat.SourceTarget) {
	ct.mutex.Lock()
	key := ct.makeKey(actorId, actorType)
	_, wasRegistered := ct.cooldowns[key]
	delete(ct.cooldowns, key)
	delete(ct.lastSent, key) // Clear last sent value
	ct.mutex.Unlock()

	// Send a final update showing the actor is balanced (0 cooldown)
	if wasRegistered && actorType == combat.User {
		ct.combat.SendGMCPBalanceUpdate(actorId, 0.0, 0.0)
	}
}

// SetActorCooldown sets cooldown for an actor's next action
func (ct *CooldownTimer) SetActorCooldown(actorId int, actorType combat.SourceTarget, duration time.Duration) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	key := ct.makeKey(actorId, actorType)
	if cd, exists := ct.cooldowns[key]; exists {
		cd.NextAction = time.Now().Add(duration)
		cd.OffBalance = true
		cd.ActorId = actorId
		cd.ActorType = actorType
		cd.MaxDuration = duration
	}
}

// GetNextActionTime returns when an actor can act next
func (ct *CooldownTimer) GetNextActionTime(actorId int, actorType combat.SourceTarget) time.Time {
	ct.mutex.RLock()
	defer ct.mutex.RUnlock()

	key := ct.makeKey(actorId, actorType)
	if cd, exists := ct.cooldowns[key]; exists {
		return cd.NextAction
	}

	return time.Now() // If not registered, can act now
}

// CanPerformAction checks if an actor can act now
func (ct *CooldownTimer) CanPerformAction(actorId int, actorType combat.SourceTarget) bool {
	ct.mutex.RLock()
	defer ct.mutex.RUnlock()

	key := ct.makeKey(actorId, actorType)
	if cd, exists := ct.cooldowns[key]; exists {
		return time.Now().After(cd.NextAction) || time.Now().Equal(cd.NextAction)
	}

	return true // If not registered, can act
}

// processCooldowns runs in a goroutine to check and trigger callbacks
func (ct *CooldownTimer) processCooldowns() {
	for {
		select {
		case <-ct.ticker.C:
			ct.checkCooldowns()
		case <-ct.stopChan:
			return
		}
	}
}

// checkCooldowns checks all cooldowns and triggers callbacks
func (ct *CooldownTimer) checkCooldowns() {
	ct.mutex.Lock()
	now := time.Now()

	// Find actors whose cooldowns have expired and who were off balance
	type balanceNotification struct {
		actorId   int
		actorType combat.SourceTarget
		callback  func()
	}

	notifications := make([]balanceNotification, 0)

	type gmcpBalance struct {
		remaining float64
		max       float64
	}
	gmcpUpdates := make(map[int]gmcpBalance) // userId -> balance info

	for _, cd := range ct.cooldowns {
		// Send GMCP updates for users who are unbalanced
		if cd.ActorType == combat.User && cd.OffBalance {
			remaining := cd.NextAction.Sub(now).Seconds()
			if remaining > 0 {
				gmcpUpdates[cd.ActorId] = gmcpBalance{
					remaining: remaining,
					max:       cd.MaxDuration.Seconds(),
				}
			}
		}

		if cd.OffBalance && (now.After(cd.NextAction) || now.Equal(cd.NextAction)) {
			// Mark as balanced
			cd.OffBalance = false

			mudlog.Debug("Balance Restored", "actorId", cd.ActorId, "actorType", cd.ActorType)

			notifications = append(notifications, balanceNotification{
				actorId:   cd.ActorId,
				actorType: cd.ActorType,
				callback:  cd.Callback,
			})

			// Send GMCP update for balanced state
			if cd.ActorType == combat.User {
				gmcpUpdates[cd.ActorId] = gmcpBalance{
					remaining: 0.0,
					max:       0.0,
				}
				// Clear the last sent value so we send the 0.0
				delete(ct.lastSent, ct.makeKey(cd.ActorId, combat.User))
			}
		}
	}
	// Process lastSent updates while still holding the lock
	toSend := make(map[int]gmcpBalance)
	for userId, balance := range gmcpUpdates {
		key := ct.makeKey(userId, combat.User)
		lastValue, exists := ct.lastSent[key]

		// Only send if value has changed
		if !exists || lastValue != balance.remaining {
			toSend[userId] = balance
			ct.lastSent[key] = balance.remaining
		}
	}
	ct.mutex.Unlock()

	// Send GMCP balance updates only if changed
	for userId, balance := range toSend {
		ct.combat.SendGMCPBalanceUpdate(userId, balance.remaining, balance.max)
	}

	// Send balance notifications
	for _, notif := range notifications {
		// Send "You are balanced" message to users
		if notif.actorType == combat.User {
			ct.combat.SendBalanceNotification(notif.actorId, notif.actorType)
		}

		if notif.callback != nil {
			// Run callback in goroutine to avoid blocking timer
			go notif.callback()
		}
	}
}

// makeKey creates a unique key for an actor
func (ct *CooldownTimer) makeKey(actorId int, actorType combat.SourceTarget) string {
	return fmt.Sprintf("%s:%d", actorType, actorId)
}
