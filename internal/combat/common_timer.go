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
	stopOnce   sync.Once
	updateFunc func()
	name       string // For logging
}

// NewBaseTimer creates a new base timer
func NewBaseTimer(name string, updateFunc func()) *BaseTimer {
	return &BaseTimer{
		name:       name,
		updateFunc: updateFunc,
		stopChan:   make(chan bool),
	}
}

// Start begins the timer loop
func (bt *BaseTimer) Start() error {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	if bt.active {
		return nil // Already running
	}

	// Reset stop once to allow restart
	bt.stopOnce = sync.Once{}

	// Create ticker for 100ms updates
	bt.ticker = time.NewTicker(100 * time.Millisecond)
	bt.stopChan = make(chan bool)
	bt.active = true

	// Start update loop
	go bt.runUpdateLoop()

	if mudlog.IsInitialized() {
		mudlog.Info("BaseTimer started", "name", bt.name)
	}
	return nil
}

// Stop halts the timer loop
func (bt *BaseTimer) Stop() error {
	bt.stopOnce.Do(func() {
		bt.mutex.Lock()
		defer bt.mutex.Unlock()

		if !bt.active {
			return
		}

		bt.active = false

		if bt.ticker != nil {
			bt.ticker.Stop()
		}

		close(bt.stopChan)

		if mudlog.IsInitialized() {
			mudlog.Info("BaseTimer stopped", "name", bt.name)
		}
	})
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
	for {
		select {
		case <-bt.ticker.C:
			if bt.updateFunc != nil {
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
