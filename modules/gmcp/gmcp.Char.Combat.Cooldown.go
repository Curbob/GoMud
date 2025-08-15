// Cooldown module provides high-frequency (5Hz) combat round timer updates.
// Timer only runs when players are actively in combat to minimize CPU usage.
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

type GMCPCombatCooldownUpdate struct {
	UserId          int
	CooldownSeconds float64
	MaxSeconds      float64
	NameActive      string
	NameIdle        string
}

func (g GMCPCombatCooldownUpdate) Type() string { return `GMCPCombatCooldownUpdate` }

// CombatCooldownTimer sends 5Hz updates for smooth UI countdown animations
type CombatCooldownTimer struct {
	ticker       *time.Ticker
	done         chan bool
	roundStarted time.Time
	roundNumber  uint64
	roundMutex   sync.RWMutex
	playerMutex  sync.RWMutex
	players      map[int]bool
	running      bool
	runningMutex sync.Mutex
}

var cooldownTimer *CombatCooldownTimer

func InitCombatCooldownTimer() {
	cooldownTimer = &CombatCooldownTimer{
		players: make(map[int]bool),
		done:    make(chan bool),
	}

	events.RegisterListener(events.NewRound{}, cooldownTimer.handleNewRound)
	events.RegisterListener(GMCPCombatCooldownUpdate{}, handleCombatCooldownUpdate)

}

func (ct *CombatCooldownTimer) handleNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		mudlog.Error("GMCPCombatCooldown", "action", "handleNewRound", "error", "type assertion failed", "expectedType", "events.NewRound", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	ct.roundMutex.Lock()
	ct.roundStarted = evt.TimeNow
	ct.roundNumber = evt.RoundNumber
	ct.roundMutex.Unlock()

	return events.Continue
}

func (ct *CombatCooldownTimer) AddPlayer(userId int) {
	ct.playerMutex.Lock()
	ct.players[userId] = true
	needsStart := len(ct.players) == 1
	ct.playerMutex.Unlock()

	mudlog.Info("CombatCooldownTimer", "action", "AddPlayer", "userId", userId, "needsStart", needsStart)

	if needsStart {
		ct.start()
	}
}

func (ct *CombatCooldownTimer) RemovePlayer(userId int) {
	ct.playerMutex.Lock()
	delete(ct.players, userId)
	shouldStop := len(ct.players) == 0
	ct.playerMutex.Unlock()

	if shouldStop {
		ct.stop()
	}
}

func (ct *CombatCooldownTimer) start() {
	ct.runningMutex.Lock()
	defer ct.runningMutex.Unlock()

	if ct.running {
		return
	}

	ct.running = true
	// 200ms interval provides 5 updates per second for smooth countdown animation
	ct.ticker = time.NewTicker(200 * time.Millisecond)

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

func (ct *CombatCooldownTimer) stop() {
	ct.runningMutex.Lock()
	defer ct.runningMutex.Unlock()

	if !ct.running {
		return
	}

	ct.running = false
	ct.ticker.Stop()

	// Non-blocking send prevents deadlock if channel is full
	select {
	case ct.done <- true:
	default:
	}

	mudlog.Info("CombatCooldownTimer", "action", "stopped")
}

func (ct *CombatCooldownTimer) sendUpdates() {
	ct.roundMutex.RLock()
	roundStarted := ct.roundStarted
	ct.roundMutex.RUnlock()

	if roundStarted.IsZero() {
		roundStarted = time.Now()
	}

	timingConfig := configs.GetTimingConfig()
	roundDuration := time.Duration(timingConfig.RoundSeconds) * time.Second

	elapsed := time.Since(roundStarted)
	remainingMs := roundDuration - elapsed
	if remainingMs < 0 {
		remainingMs = 0
	}

	remainingSeconds := float64(remainingMs) / float64(time.Second)
	maxSeconds := float64(roundDuration) / float64(time.Second)

	ct.playerMutex.RLock()
	playerIds := make([]int, 0, len(ct.players))
	for userId := range ct.players {
		playerIds = append(playerIds, userId)
	}
	ct.playerMutex.RUnlock()

	if len(playerIds) > 0 {
	}

	for _, userId := range playerIds {
		user := users.GetByUserId(userId)
		if user == nil {
			ct.playerMutex.Lock()
			delete(ct.players, userId)
			ct.playerMutex.Unlock()
			mudlog.Warn("CombatCooldownTimer", "action", "sendUpdates", "issue", "user not found, cleaning up stale entry", "userId", userId)
			continue
		}

		// Cooldown only shows when player is attacking (has aggro set)
		if user.Character.Aggro == nil {
			continue
		}

		events.AddToQueue(GMCPCombatCooldownUpdate{
			UserId:          userId,
			CooldownSeconds: remainingSeconds,
			MaxSeconds:      maxSeconds,
			NameActive:      "Combat Round",
			NameIdle:        "Ready",
		})
	}
}

func TrackCombatPlayer(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		mudlog.Warn("CombatCooldownTimer", "action", "TrackCombatPlayer", "issue", "attempted to track non-existent user", "userId", userId)
		return
	}

	if cooldownTimer != nil {
		cooldownTimer.AddPlayer(userId)
	}
}

func UntrackCombatPlayer(userId int) {
	if cooldownTimer != nil {
		user := users.GetByUserId(userId)
		if user != nil {
			// Send final 0.0 update before removing
			timingConfig := configs.GetTimingConfig()
			maxSeconds := float64(timingConfig.RoundSeconds)

			// Send final 0.0 update before removing to signal combat end
			handleCombatCooldownUpdate(GMCPCombatCooldownUpdate{
				UserId:          userId,
				CooldownSeconds: 0.0,
				MaxSeconds:      maxSeconds,
				NameActive:      "Combat Round",
				NameIdle:        "Ready",
			})
		}

		cooldownTimer.RemovePlayer(userId)
	}
}

func handleCombatCooldownUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatCooldownUpdate)
	if !typeOk {
		mudlog.Error("GMCPCombatCooldown", "action", "handleCombatCooldownUpdate", "error", "type assertion failed", "expectedType", "GMCPCombatCooldownUpdate", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	_, valid := validateUserForGMCP(evt.UserId, "GMCPCombatCooldown")
	if !valid {
		return events.Continue
	}

	payload := map[string]interface{}{
		"cooldown":     fmt.Sprintf("%.1f", evt.CooldownSeconds),
		"max_cooldown": fmt.Sprintf("%.1f", evt.MaxSeconds),
		"name_active":  evt.NameActive,
		"name_idle":    evt.NameIdle,
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.Combat.Cooldown",
		Payload: payload,
	})

	return events.Continue
}
