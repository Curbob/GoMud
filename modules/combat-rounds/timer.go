package combatrounds

import (
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
)

// RoundBasedTimer tracks combat rounds and sends GMCP updates
type RoundBasedTimer struct {
	*combat.BaseTimer
	*combat.PlayerTracker
	combat       *RoundBasedCombat
	roundStarted time.Time
	roundNumber  uint64

	// Configuration
	roundDuration time.Duration

	// Event listener IDs for cleanup
	newRoundListenerId events.ListenerId
	newTurnListenerId  events.ListenerId

	// Additional mutex for round-specific data
	roundMutex sync.RWMutex
}

// NewRoundBasedTimer creates a new round-based timer
func NewRoundBasedTimer(c *RoundBasedCombat) *RoundBasedTimer {
	// Get the actual round duration from the game config
	timingConfig := configs.GetTimingConfig()
	roundDuration := time.Duration(timingConfig.RoundSeconds) * time.Second

	rt := &RoundBasedTimer{
		combat:        c,
		PlayerTracker: combat.NewPlayerTracker(),
		roundDuration: roundDuration,
	}

	// Create base timer with update function
	rt.BaseTimer = combat.NewBaseTimer("combat-rounds", rt.updatePlayers)

	return rt
}

// Start implements ICombatTimer
func (rt *RoundBasedTimer) Start() error {
	// Start base timer
	if err := rt.BaseTimer.Start(); err != nil {
		return err
	}

	// Register for round events
	rt.newRoundListenerId = events.RegisterListener(events.NewRound{}, rt.handleNewRound)
	rt.newTurnListenerId = events.RegisterListener(events.NewTurn{}, rt.handleNewTurn)

	return nil
}

// Stop implements ICombatTimer
func (rt *RoundBasedTimer) Stop() error {
	// Unregister event listeners
	if rt.newRoundListenerId != 0 {
		events.UnregisterListener(events.NewRound{}, rt.newRoundListenerId)
		rt.newRoundListenerId = 0
	}
	if rt.newTurnListenerId != 0 {
		events.UnregisterListener(events.NewTurn{}, rt.newTurnListenerId)
		rt.newTurnListenerId = 0
	}

	// Stop base timer
	return rt.BaseTimer.Stop()
}

// handleNewRound updates round tracking
func (rt *RoundBasedTimer) handleNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}

	rt.roundMutex.Lock()
	rt.roundStarted = evt.TimeNow
	rt.roundNumber = evt.RoundNumber
	rt.roundMutex.Unlock()

	return events.Continue
}

// handleNewTurn checks combat state for all tracked players
func (rt *RoundBasedTimer) handleNewTurn(e events.Event) events.ListenerReturn {
	playersToCheck := rt.GetTrackedPlayers()

	// Check each player's combat state
	for _, userId := range playersToCheck {
		if user := users.GetByUserId(userId); user != nil {
			// If player no longer has aggro, remove from combat tracking
			if user.Character.Aggro == nil {
				rt.RemovePlayer(userId)
				
				// Send final GMCP update showing combat has ended
				events.AddToQueue(gmcp.GMCPCombatStatusUpdate{
					UserId:          userId,
					CooldownSeconds: 0,
					MaxSeconds:      float64(rt.roundDuration) / float64(time.Second),
					NameActive:      "Combat Round",
					NameIdle:        "Ready",
					InCombat:        false,
					CombatStyle:     combat.GetActiveCombatSystemName(),
					RoundNumber:     rt.roundNumber,
				})
			}
		} else {
			// User no longer exists, remove from tracking
			rt.RemovePlayer(userId)
		}
	}

	return events.Continue
}

// updatePlayers sends GMCP updates to all tracked players
func (rt *RoundBasedTimer) updatePlayers() {
	if !rt.IsActive() {
		return
	}

	rt.roundMutex.RLock()
	roundStarted := rt.roundStarted
	roundNumber := rt.roundNumber
	rt.roundMutex.RUnlock()

	// Calculate progress through current round
	elapsed := time.Since(roundStarted)
	remainingMs := rt.roundDuration - elapsed
	if remainingMs < 0 {
		remainingMs = 0
	}

	remainingSeconds := float64(remainingMs) / float64(time.Second)
	maxSeconds := float64(rt.roundDuration) / float64(time.Second)

	// Send updates to all tracked players
	players := rt.GetTrackedPlayers()
	for _, userId := range players {
		if user := users.GetByUserId(userId); user != nil {
			// Send GMCP combat status update
			events.AddToQueue(gmcp.GMCPCombatStatusUpdate{
				UserId:          userId,
				CooldownSeconds: remainingSeconds,
				MaxSeconds:      maxSeconds,
				NameActive:      "Combat Round",
				NameIdle:        "Ready",
				InCombat:        user.Character.Aggro != nil,
				CombatStyle:     combat.GetActiveCombatSystemName(),
				RoundNumber:     roundNumber,
			})
		}
	}
}
