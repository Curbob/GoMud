package combatrounds

import (
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
)

// RoundBasedTimer tracks combat rounds and sends GMCP updates
type RoundBasedTimer struct {
	combat       *RoundBasedCombat
	roundStarted time.Time
	roundNumber  uint64

	// Track which players are in combat
	playersInCombat map[int]bool
	mutex           sync.RWMutex

	// Configuration
	roundDuration time.Duration

	// Ticker for sending updates
	ticker     *time.Ticker
	stopTicker chan bool
	active     bool
}

// NewRoundBasedTimer creates a new round-based timer
func NewRoundBasedTimer(combat *RoundBasedCombat) *RoundBasedTimer {
	// Get the actual round duration from the game config
	timingConfig := configs.GetTimingConfig()
	roundDuration := time.Duration(timingConfig.RoundSeconds) * time.Second

	return &RoundBasedTimer{
		combat:          combat,
		playersInCombat: make(map[int]bool),
		roundDuration:   roundDuration,
		stopTicker:      make(chan bool),
	}
}

// startTimer begins the timer
func (rt *RoundBasedTimer) startTimer() {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	if rt.active {
		return
	}

	rt.active = true
	rt.ticker = time.NewTicker(100 * time.Millisecond) // Update every 100ms

	// Start update goroutine
	go rt.updateLoop()

	// Register for round events
	events.RegisterListener(events.NewRound{}, rt.handleNewRound)
	events.RegisterListener(events.NewTurn{}, rt.handleNewTurn)

	mudlog.Info("Round Timer", "status", "started")
}

// stopTimer halts the timer
func (rt *RoundBasedTimer) stopTimer() {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	if !rt.active {
		return
	}

	rt.active = false
	if rt.ticker != nil {
		rt.ticker.Stop()
	}
	close(rt.stopTicker)

	mudlog.Info("Round Timer", "status", "stopped")
}

// handleNewRound updates round tracking
func (rt *RoundBasedTimer) handleNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}

	rt.mutex.Lock()
	rt.roundStarted = evt.TimeNow
	rt.roundNumber = evt.RoundNumber
	rt.mutex.Unlock()

	return events.Continue
}

// handleNewTurn checks combat state for all tracked players
func (rt *RoundBasedTimer) handleNewTurn(e events.Event) events.ListenerReturn {
	rt.mutex.Lock()
	playersToCheck := make([]int, 0, len(rt.playersInCombat))
	for userId := range rt.playersInCombat {
		playersToCheck = append(playersToCheck, userId)
	}
	rt.mutex.Unlock()

	// Check each player's combat state
	for _, userId := range playersToCheck {
		if user := users.GetByUserId(userId); user != nil {
			// If player no longer has aggro, remove from combat tracking
			if user.Character.Aggro == nil {
				rt.RemovePlayer(userId)
			}
		} else {
			// User no longer exists, remove from tracking
			rt.RemovePlayer(userId)
		}
	}

	return events.Continue
}

// AddPlayer marks a player as being in combat
func (rt *RoundBasedTimer) AddPlayer(userId int) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	rt.playersInCombat[userId] = true
}

// RemovePlayer marks a player as no longer in combat
func (rt *RoundBasedTimer) RemovePlayer(userId int) {
	rt.mutex.Lock()
	wasInCombat := rt.playersInCombat[userId]
	delete(rt.playersInCombat, userId)
	rt.mutex.Unlock()

	// Send a final update showing the player is ready (0 cooldown)
	if wasInCombat {
		events.AddToQueue(gmcp.GMCPCombatCooldownUpdate{
			UserId:          userId,
			CooldownSeconds: 0,
			MaxSeconds:      rt.roundDuration.Seconds(),
			NameActive:      "Round Countdown",
			NameIdle:        "Ready",
		})
	}
}

// IsPlayerInCombat checks if a player is in combat
func (rt *RoundBasedTimer) IsPlayerInCombat(userId int) bool {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	return rt.playersInCombat[userId]
}

// updateLoop sends periodic GMCP updates
func (rt *RoundBasedTimer) updateLoop() {
	for {
		select {
		case <-rt.ticker.C:
			rt.sendUpdates()
		case <-rt.stopTicker:
			return
		}
	}
}

// sendUpdates sends GMCP updates to all players in combat
func (rt *RoundBasedTimer) sendUpdates() {
	rt.mutex.RLock()
	players := make([]int, 0, len(rt.playersInCombat))
	for userId := range rt.playersInCombat {
		players = append(players, userId)
	}
	roundStart := rt.roundStarted
	roundDuration := rt.roundDuration
	rt.mutex.RUnlock()

	// If we don't have a valid round start time yet, don't send updates
	if roundStart.IsZero() {
		return
	}

	// Calculate time remaining in current round
	elapsed := time.Since(roundStart)
	remaining := roundDuration - elapsed

	// If we've exceeded the round duration, we're waiting for next round
	// Show a small amount of time remaining to indicate "almost ready"
	if remaining < 0 {
		remaining = 100 * time.Millisecond
	}

	// Convert to seconds
	remainingSeconds := remaining.Seconds()
	maxSeconds := roundDuration.Seconds()

	// Send updates to all players in combat
	for _, userId := range players {
		events.AddToQueue(gmcp.GMCPCombatCooldownUpdate{
			UserId:          userId,
			CooldownSeconds: remainingSeconds,
			MaxSeconds:      maxSeconds,
			NameActive:      "Round Countdown",
			NameIdle:        "Ready",
		})
	}
}

// ICombatTimer implementation methods

// Start implements ICombatTimer.Start()
func (rt *RoundBasedTimer) Start() error {
	rt.startTimer()
	return nil
}

// Stop implements ICombatTimer.Stop()
func (rt *RoundBasedTimer) Stop() error {
	rt.stopTimer()
	return nil
}

func (rt *RoundBasedTimer) CanPerformAction(actorId int, actorType combat.SourceTarget) bool {
	// In round-based combat, actions are always allowed
	// The round system itself gates when combat happens
	return true
}

func (rt *RoundBasedTimer) SetActorCooldown(actorId int, actorType combat.SourceTarget, duration time.Duration) {
	// Round-based combat doesn't use individual cooldowns
	// But we track who's in combat
	if actorType == combat.User {
		rt.AddPlayer(actorId)
	}
}

func (rt *RoundBasedTimer) GetNextActionTime(actorId int, actorType combat.SourceTarget) time.Time {
	// Return time of next round
	rt.mutex.RLock()
	roundStart := rt.roundStarted
	roundDuration := rt.roundDuration
	rt.mutex.RUnlock()

	// If we don't have a round start time yet, return current time
	if roundStart.IsZero() {
		return time.Now()
	}

	elapsed := time.Since(roundStart)
	remaining := roundDuration - elapsed

	if remaining < 0 {
		// We're past the round duration, next round should start soon
		return time.Now()
	}

	return time.Now().Add(remaining)
}

func (rt *RoundBasedTimer) RegisterActor(actorId int, actorType combat.SourceTarget, callback func()) {
	// Round-based combat doesn't use callbacks
	if actorType == combat.User {
		rt.AddPlayer(actorId)
	}
}

func (rt *RoundBasedTimer) UnregisterActor(actorId int, actorType combat.SourceTarget) {
	if actorType == combat.User {
		rt.RemovePlayer(actorId)
	}
}
