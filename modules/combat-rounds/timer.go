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

	// Track if we've started the timer
	timerStarted bool
	timerMutex   sync.Mutex
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
	// ARCHITECTURAL NOTE: Event registration is deferred to prevent deadlocks
	// The combat system switching uses events, and registering listeners during
	// event processing (when combat system initializes) would cause a deadlock
	// in the event system. This pattern is unique to combat modules due to their
	// dynamic initialization/shutdown during runtime.
	go func() {
		// Small delay to ensure we're out of the event handler
		time.Sleep(1 * time.Millisecond)

		// Register for round events
		rt.newRoundListenerId = events.RegisterListener(events.NewRound{}, rt.handleNewRound)
		rt.newTurnListenerId = events.RegisterListener(events.NewTurn{}, rt.handleNewTurn)

		mudlog.Info("RoundBasedTimer", "action", "event listeners registered")
	}()

	// Don't start the actual timer until players are added
	return nil
}

// Stop implements ICombatTimer
func (rt *RoundBasedTimer) Stop() error {
	mudlog.Info("RoundBasedTimer", "action", "Stop called")

	// Save listener IDs for deferred unregistration
	roundListenerId := rt.newRoundListenerId
	turnListenerId := rt.newTurnListenerId
	rt.newRoundListenerId = 0
	rt.newTurnListenerId = 0

	// Defer unregistration to avoid deadlock when called from event handler
	if roundListenerId != 0 || turnListenerId != 0 {
		go func() {
			// Small delay to ensure we're out of the event handler
			time.Sleep(1 * time.Millisecond)

			if roundListenerId != 0 {
				mudlog.Info("RoundBasedTimer", "action", "unregistering NewRound listener")
				events.UnregisterListener(events.NewRound{}, roundListenerId)
			}
			if turnListenerId != 0 {
				mudlog.Info("RoundBasedTimer", "action", "unregistering NewTurn listener")
				events.UnregisterListener(events.NewTurn{}, turnListenerId)
			}
		}()
	}

	// Stop base timer if it's running
	rt.timerMutex.Lock()
	defer rt.timerMutex.Unlock()

	if rt.timerStarted {
		mudlog.Info("RoundBasedTimer", "action", "stopping base timer")
		rt.timerStarted = false
		// Also defer the base timer stop to avoid any potential issues
		go rt.BaseTimer.Stop()
	}

	mudlog.Info("RoundBasedTimer", "action", "Stop completed")
	return nil
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
	// Combat end detection is now handled in updatePlayers() for immediate response
	// This function can be used for other turn-based logic if needed
	return events.Continue
}

// updatePlayers sends GMCP updates to all tracked players
func (rt *RoundBasedTimer) updatePlayers() {
	rt.timerMutex.Lock()
	if !rt.timerStarted {
		rt.timerMutex.Unlock()
		return
	}
	rt.timerMutex.Unlock()

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
			// Check if player still has aggro
			if user.Character.Aggro == nil {
				// Player no longer in combat, remove and send final update
				rt.RemovePlayer(userId)

				// Send final GMCP update showing combat has ended
				events.AddToQueue(gmcp.GMCPCombatStatusUpdate{
					UserId:          userId,
					CooldownSeconds: 0,
					MaxSeconds:      maxSeconds,
					NameActive:      "Combat Round",
					NameIdle:        "Ready",
					InCombat:        false,
					CombatStyle:     combat.GetActiveCombatSystemName(),
					RoundNumber:     roundNumber,
				})
			} else {
				// Still in combat, send normal update
				events.AddToQueue(gmcp.GMCPCombatStatusUpdate{
					UserId:          userId,
					CooldownSeconds: remainingSeconds,
					MaxSeconds:      maxSeconds,
					NameActive:      "Combat Round",
					NameIdle:        "Ready",
					InCombat:        true,
					CombatStyle:     combat.GetActiveCombatSystemName(),
					RoundNumber:     roundNumber,
				})
			}
		} else {
			// User no longer exists, remove from tracking
			rt.RemovePlayer(userId)
		}
	}
}

// AddPlayer adds a player to combat tracking and starts timer if needed
func (rt *RoundBasedTimer) AddPlayer(userId int) {
	// Add to tracker
	rt.PlayerTracker.AddPlayer(userId)

	// Start timer if this is the first player
	rt.timerMutex.Lock()
	defer rt.timerMutex.Unlock()

	if !rt.timerStarted && len(rt.GetTrackedPlayers()) > 0 {
		rt.timerStarted = true
		rt.BaseTimer.Start()
	}
}

// RemovePlayer removes a player from combat tracking and stops timer if needed
func (rt *RoundBasedTimer) RemovePlayer(userId int) {
	// Remove from tracker
	rt.PlayerTracker.RemovePlayer(userId)

	// Check if we should stop timer
	shouldStop := false
	rt.timerMutex.Lock()
	if rt.timerStarted && len(rt.GetTrackedPlayers()) == 0 {
		rt.timerStarted = false
		shouldStop = true
	}
	rt.timerMutex.Unlock()

	// Stop timer outside of mutex to avoid deadlock
	if shouldStop {
		go rt.BaseTimer.Stop()
	}
}
