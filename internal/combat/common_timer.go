package combat

import (
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// BaseTimer provides common timer functionality for combat systems
type BaseTimer struct {
	mutex      sync.RWMutex
	ticker     *time.Ticker
	stopChan   chan bool
	active     bool
	updateFunc func()
	name       string         // For logging
	wg         sync.WaitGroup // Track goroutines
}

// NewBaseTimer creates a new base timer
func NewBaseTimer(name string, updateFunc func()) *BaseTimer {
	return &BaseTimer{
		name:       name,
		updateFunc: updateFunc,
		// stopChan created in Start() as buffered channel
	}
}

// Start begins the timer loop
func (bt *BaseTimer) Start() error {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	if bt.active {
		return nil // Already running
	}

	// Create ticker for 100ms updates
	bt.ticker = time.NewTicker(100 * time.Millisecond)
	bt.stopChan = make(chan bool, 1) // Buffered to prevent blocking
	bt.active = true

	// Start update loop
	bt.wg.Add(1)
	go bt.runUpdateLoop()

	mudlog.Info("BaseTimer started", "name", bt.name)
	return nil
}

// Stop halts the timer loop
func (bt *BaseTimer) Stop() error {
	bt.mutex.Lock()

	if !bt.active {
		bt.mutex.Unlock()
		return nil
	}

	bt.active = false

	if bt.ticker != nil {
		bt.ticker.Stop()
		bt.ticker = nil
	}

	// Send stop signal (non-blocking due to buffer)
	select {
	case bt.stopChan <- true:
	default:
	}

	bt.mutex.Unlock()

	// Wait for the update loop to finish with timeout
	done := make(chan struct{})
	go func() {
		bt.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Timer stopped cleanly
		mudlog.Info("BaseTimer stopped", "name", bt.name)
	case <-time.After(2 * time.Second):
		// Timeout - log warning and continue
		mudlog.Error("BaseTimer stop timeout", "name", bt.name,
			"message", "Timer goroutine did not stop within 2 seconds")
	}

	return nil
}

// IsActive returns whether the timer is running
func (bt *BaseTimer) IsActive() bool {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()
	return bt.active
}

// runUpdateLoop runs the timer's update function on each tick
func (bt *BaseTimer) runUpdateLoop() {
	defer bt.wg.Done()

	for {
		select {
		case <-bt.ticker.C:
			// Check if we're still active
			bt.mutex.RLock()
			active := bt.active
			bt.mutex.RUnlock()

			if active && bt.updateFunc != nil {
				bt.updateFunc()
			}
		case <-bt.stopChan:
			return
		}
	}
}

// PlayerTracker provides common player tracking functionality
type PlayerTracker struct {
	players map[int]bool
	mutex   sync.RWMutex
}

// NewPlayerTracker creates a new player tracker
func NewPlayerTracker() *PlayerTracker {
	return &PlayerTracker{
		players: make(map[int]bool),
	}
}

// AddPlayer adds a player to tracking
func (pt *PlayerTracker) AddPlayer(userId int) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	pt.players[userId] = true
}

// RemovePlayer removes a player from tracking
func (pt *PlayerTracker) RemovePlayer(userId int) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	delete(pt.players, userId)
}

// IsPlayerTracked checks if a player is being tracked
func (pt *PlayerTracker) IsPlayerTracked(userId int) bool {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()
	return pt.players[userId]
}

// GetTrackedPlayers returns a copy of tracked player IDs
func (pt *PlayerTracker) GetTrackedPlayers() []int {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	players := make([]int, 0, len(pt.players))
	for userId := range pt.players {
		players = append(players, userId)
	}
	return players
}

// Clear removes all tracked players
func (pt *PlayerTracker) Clear() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	pt.players = make(map[int]bool)
}
