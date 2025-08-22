package copyover

import (
	"time"
)

// State represents the current state of the copyover system
type State int

const (
	StateIdle State = iota
	StatePreparing
	StateRecovering
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StatePreparing:
		return "preparing"
	case StateRecovering:
		return "recovering"
	default:
		return "unknown"
	}
}

// GatherFunc is a function that gathers state from a subsystem
type GatherFunc func() (interface{}, error)

// RestoreFunc is a function that restores state to a subsystem
type RestoreFunc func(data interface{}) error

// VetoFunc is an optional function that can veto a copyover
type VetoFunc func() (canProceed bool, reason string)

// Subsystem represents a registered subsystem that participates in copyover
type Subsystem struct {
	Name    string
	Gather  GatherFunc
	Restore RestoreFunc
	Veto    VetoFunc // Optional
}

// CopyoverData is the main structure that gets serialized to disk
type CopyoverData struct {
	Version     string                 `json:"version"`
	Timestamp   time.Time              `json:"timestamp"`
	Environment map[string]string      `json:"environment"`
	Subsystems  map[string]interface{} `json:"subsystems"`
}

// VetoResult contains information about a subsystem veto
type VetoResult struct {
	Subsystem string
	Reason    string
	Time      time.Time
}

// Options for initiating a copyover
type Options struct {
	Countdown   int    // Seconds to wait before copyover (0 = immediate)
	Reason      string // Why the copyover is happening
	InitiatedBy int    // User ID who initiated (0 = system)
	Build       bool   // Whether to rebuild the executable before restart
}

// Status provides information about the current copyover state
type Status struct {
	State        State
	ScheduledFor time.Time
	Progress     int // Percentage 0-100
	Message      string
}
