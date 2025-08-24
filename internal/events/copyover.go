package events

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// EventsCopyoverState represents event queue state during copyover
type EventsCopyoverState struct {
	// We don't preserve the event queue itself - events are transient
	// But we track some statistics
	QueueSize    int  `json:"queue_size"`
	EventsPaused bool `json:"events_paused"`
	RoundNumber  int  `json:"round_number"`
	TurnNumber   int  `json:"turn_number"`
}

func init() {
	// Register the events subsystem with copyover
	copyover.RegisterWithVeto("events", gatherEventState, restoreEventState, vetoEventCopyover)

	// Set up the broadcast callback for copyover announcements
	manager := copyover.GetManager()
	manager.SetBroadcastFunc(func(message string) {
		// Queue a broadcast event
		AddToQueue(Broadcast{
			Text: message,
		})
	})

	// Register pre-copyover hook to save all rooms
	// Note: The actual room save function needs to be registered from a higher level
	// to avoid import cycles. This is typically done from world.go or main.go
}

// gatherEventState prepares the event system for copyover
func gatherEventState() (interface{}, error) {
	// Pause event processing
	PauseEventProcessing()

	state := EventsCopyoverState{
		QueueSize:    GetQueueSize(),
		EventsPaused: true,
		RoundNumber:  GetCurrentRound(),
		TurnNumber:   GetCurrentTurn(),
	}

	// Clear the event queue - events are transient and shouldn't survive copyover
	// Important events should be regenerated after copyover
	ClearEventQueue()

	mudlog.Info("Copyover", "subsystem", "events", "gathered", "queueSize", state.QueueSize, "round", state.RoundNumber)
	return state, nil
}

// restoreEventState restores event processing after copyover
func restoreEventState(data interface{}) error {
	// The actual state doesn't matter much - we just need to restart processing

	// Resume event processing
	ResumeEventProcessing()

	// Trigger a post-copyover event so systems can reinitialize
	AddToQueue(PostCopyover{
		Timestamp: time.Now(),
	})

	mudlog.Info("Copyover", "subsystem", "events", "restored", "processing resumed")
	return nil
}

// vetoEventCopyover checks if it's safe to copyover from events perspective
func vetoEventCopyover() (bool, string) {
	queueSize := GetQueueSize()

	// If event queue is extremely large, might want to wait
	if queueSize > 1000 {
		return false, fmt.Sprintf("event queue too large (%d events)", queueSize)
	}

	// Check if we're in the middle of combat rounds
	if IsProcessingCombat() {
		// Soft veto - copyover can proceed but it's not ideal
		return true, "combat events in progress"
	}

	return true, ""
}

// PostCopyover event fired after copyover completes
type PostCopyover struct {
	Timestamp time.Time
}

func (e PostCopyover) Type() string {
	return "PostCopyover"
}

// CopyoverRecoveryComplete event fired when all copyover recovery tasks are done
// This includes: workers started, users restored to rooms, connections re-established
type CopyoverRecoveryComplete struct {
	Timestamp      time.Time // When recovery completed
	UsersRestored  int       // Number of users restored from copyover
	RoomsWithUsers int       // Number of rooms containing active users
}

func (e CopyoverRecoveryComplete) Type() string {
	return "CopyoverRecoveryComplete"
}

// Helper functions that would need to be implemented

// PauseEventProcessing stops processing events
func PauseEventProcessing() {
	// This would pause the event worker
}

// ResumeEventProcessing resumes processing events
func ResumeEventProcessing() {
	// This would resume the event worker
}

// GetQueueSize returns the number of events in queue
func GetQueueSize() int {
	// Return actual queue size
	return 0
}

// ClearEventQueue removes all pending events
func ClearEventQueue() {
	// Clear the queue
}

// GetCurrentRound returns the current game round number
func GetCurrentRound() int {
	// Return current round
	return 0
}

// GetCurrentTurn returns the current game turn number
func GetCurrentTurn() int {
	// Return current turn
	return 0
}

// IsProcessingCombat checks if combat events are being processed
func IsProcessingCombat() bool {
	// Check if combat is active
	return false
}
