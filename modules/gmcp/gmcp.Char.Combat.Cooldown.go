package gmcp

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GMCPCombatCooldownUpdate is sent frequently during combat to update cooldown timers
type GMCPCombatCooldownUpdate struct {
	UserId          int
	CooldownSeconds float64
	MaxSeconds      float64
	NameActive      string
	NameIdle        string
}

func (g GMCPCombatCooldownUpdate) Type() string { return `GMCPCombatCooldownUpdate` }

// CombatCooldownTimer manages the fast 1/10 second cooldown updates
type CombatCooldownTimer struct {
	ticker       *time.Ticker
	done         chan bool
	roundStarted time.Time
	roundNumber  uint64
	roundMutex   sync.RWMutex
	playerMutex  sync.RWMutex
	players      map[int]bool // tracking which players are in combat
	running      bool
	runningMutex sync.Mutex
}

var cooldownTimer *CombatCooldownTimer

// InitCombatCooldownTimer initializes the cooldown timer system
func InitCombatCooldownTimer() {
	cooldownTimer = &CombatCooldownTimer{
		players: make(map[int]bool),
		done:    make(chan bool),
	}

	// Register for NewRound events to track round timing
	events.RegisterListener(events.NewRound{}, cooldownTimer.handleNewRound)

	// Register the GMCP event handler
	events.RegisterListener(GMCPCombatCooldownUpdate{}, handleCombatCooldownUpdate)
}

// handleNewRound updates round tracking
func (ct *CombatCooldownTimer) handleNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}

	ct.roundMutex.Lock()
	ct.roundStarted = evt.TimeNow
	ct.roundNumber = evt.RoundNumber
	ct.roundMutex.Unlock()

	return events.Continue
}

// AddPlayer adds a player to cooldown tracking
func (ct *CombatCooldownTimer) AddPlayer(userId int) {
	ct.playerMutex.Lock()
	ct.players[userId] = true
	needsStart := len(ct.players) == 1
	ct.playerMutex.Unlock()

	mudlog.Info("CombatCooldownTimer", "action", "AddPlayer", "userId", userId, "needsStart", needsStart)

	// Start timer if this is the first player
	if needsStart {
		ct.start()
	}
}

// RemovePlayer removes a player from cooldown tracking
func (ct *CombatCooldownTimer) RemovePlayer(userId int) {
	ct.playerMutex.Lock()
	delete(ct.players, userId)
	shouldStop := len(ct.players) == 0
	ct.playerMutex.Unlock()

	// Stop timer if no more players
	if shouldStop {
		ct.stop()
	}
}

// start begins the cooldown timer
func (ct *CombatCooldownTimer) start() {
	ct.runningMutex.Lock()
	defer ct.runningMutex.Unlock()

	if ct.running {
		return
	}

	ct.running = true
	// Reduced update frequency from 100ms to 250ms to reduce network traffic
	// This still provides smooth updates (4 per second) while reducing load
	ct.ticker = time.NewTicker(250 * time.Millisecond)

	go func() {
		for {
			select {
			case <-ct.ticker.C:
				ct.sendUpdates()
			case <-ct.done:
				return
			}
		}
	}()

	mudlog.Info("CombatCooldownTimer", "action", "started")
}

// stop halts the cooldown timer
func (ct *CombatCooldownTimer) stop() {
	ct.runningMutex.Lock()
	defer ct.runningMutex.Unlock()

	if !ct.running {
		return
	}

	ct.running = false
	ct.ticker.Stop()

	// Non-blocking send to avoid deadlock
	select {
	case ct.done <- true:
	default:
	}

	mudlog.Info("CombatCooldownTimer", "action", "stopped")
}

// sendUpdates sends cooldown updates to all tracked players
func (ct *CombatCooldownTimer) sendUpdates() {
	ct.roundMutex.RLock()
	roundStarted := ct.roundStarted
	ct.roundMutex.RUnlock()

	// If we haven't received a NewRound event yet, use current time
	if roundStarted.IsZero() {
		roundStarted = time.Now()
	}

	// Get round duration from config
	timingConfig := configs.GetTimingConfig()
	roundDuration := time.Duration(timingConfig.RoundSeconds) * time.Second

	// Calculate remaining time
	elapsed := time.Since(roundStarted)
	remainingMs := roundDuration - elapsed
	if remainingMs < 0 {
		remainingMs = 0
	}

	remainingSeconds := float64(remainingMs) / float64(time.Second)
	maxSeconds := float64(roundDuration) / float64(time.Second)

	// Get current players to update
	ct.playerMutex.RLock()
	playerIds := make([]int, 0, len(ct.players))
	for userId := range ct.players {
		playerIds = append(playerIds, userId)
	}
	ct.playerMutex.RUnlock()

	if len(playerIds) > 0 {
		mudlog.Debug("CombatCooldownTimer", "action", "sendUpdates", "playerCount", len(playerIds), "remainingSeconds", remainingSeconds)
	}

	// Send updates
	for _, userId := range playerIds {
		if user := users.GetByUserId(userId); user != nil {
			// Check if still in combat
			if user.Character.Aggro == nil {
				// Skip players no longer in combat
				// They will be removed by the CombatStatus module
				continue
			}

			// Queue cooldown update event
			events.AddToQueue(GMCPCombatCooldownUpdate{
				UserId:          userId,
				CooldownSeconds: remainingSeconds,
				MaxSeconds:      maxSeconds,
				NameActive:      "Combat Round",
				NameIdle:        "Ready",
			})
		}
		// Note: We don't remove players here to avoid deadlock
		// The CombatStatus module will handle removing players
	}
}

// TrackCombatPlayer starts tracking cooldown for a player entering combat
func TrackCombatPlayer(userId int) {
	if cooldownTimer != nil {
		cooldownTimer.AddPlayer(userId)
	}
}

// UntrackCombatPlayer stops tracking cooldown for a player leaving combat
func UntrackCombatPlayer(userId int) {
	if cooldownTimer != nil {
		// Send final 0.0 update before removing
		timingConfig := configs.GetTimingConfig()
		maxSeconds := float64(timingConfig.RoundSeconds)

		// Queue final update event
		events.AddToQueue(GMCPCombatCooldownUpdate{
			UserId:          userId,
			CooldownSeconds: 0.0,
			MaxSeconds:      maxSeconds,
			NameActive:      "Combat Round",
			NameIdle:        "Ready",
		})

		cooldownTimer.RemovePlayer(userId)
	}
}

// handleCombatCooldownUpdate sends GMCP cooldown updates
func handleCombatCooldownUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatCooldownUpdate)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	mudlog.Debug("CombatCooldownTimer", "action", "handleCombatCooldownUpdate", "userId", evt.UserId, "cooldown", evt.CooldownSeconds)

	// Make sure they have GMCP enabled
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Continue
	}

	// Build the payload
	payload := map[string]interface{}{
		"cooldown":     fmt.Sprintf("%.1f", evt.CooldownSeconds),
		"max_cooldown": fmt.Sprintf("%.1f", evt.MaxSeconds),
		"name_active":  evt.NameActive,
		"name_idle":    evt.NameIdle,
	}

	// Send the GMCP update
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.Combat.Cooldown",
		Payload: payload,
	})

	return events.Continue
}
