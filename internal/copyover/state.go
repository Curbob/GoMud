package copyover

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	// Environment variable that indicates we're recovering from copyover
	EnvCopyover = "GOMUD_COPYOVER"

	// Default file names
	StateFile = "copyover.dat"
	TempDir   = "copyover-temp"
)

// stateMachine manages the copyover state transitions
type stateMachine struct {
	currentState State
	progress     int
	message      string
	scheduledFor time.Time
}

// transition changes the state if the transition is valid
func (sm *stateMachine) transition(newState State) error {
	// Validate state transitions
	switch sm.currentState {
	case StateIdle:
		if newState != StatePreparing {
			return fmt.Errorf("invalid transition from %s to %s", sm.currentState, newState)
		}
	case StatePreparing:
		if newState != StateIdle && newState != StateRecovering {
			return fmt.Errorf("invalid transition from %s to %s", sm.currentState, newState)
		}
	case StateRecovering:
		if newState != StateIdle {
			return fmt.Errorf("invalid transition from %s to %s", sm.currentState, newState)
		}
	}

	sm.currentState = newState
	sm.progress = 0
	mudlog.Info("Copyover", "state", newState.String())
	return nil
}

// saveState writes the copyover data to disk
func saveState(data *CopyoverData) error {
	// Create temp directory if it doesn't exist
	tempPath := filepath.Join(os.TempDir(), TempDir)
	if err := os.MkdirAll(tempPath, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Marshal the data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Compress the data
	compressed := util.Compress(jsonData)

	// Save to file
	stateFile := filepath.Join(tempPath, StateFile)
	if err := util.SafeSave(stateFile, compressed); err != nil {
		return fmt.Errorf("failed to save state file: %w", err)
	}

	mudlog.Info("Copyover", "action", "State saved", "size", len(compressed), "file", stateFile)
	return nil
}

// loadState reads the copyover data from disk
func loadState() (*CopyoverData, error) {
	stateFile := filepath.Join(os.TempDir(), TempDir, StateFile)

	// Check if file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("state file not found: %s", stateFile)
	}

	// Read the file
	compressed, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Decompress
	jsonData := util.Decompress(compressed)
	if len(jsonData) == 0 && len(compressed) > 0 {
		// Try without decompression for backwards compatibility
		jsonData = compressed
	}

	// Unmarshal
	var data CopyoverData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	mudlog.Info("Copyover", "action", "State loaded", "version", data.Version, "timestamp", data.Timestamp)
	return &data, nil
}

// cleanupState removes temporary files
func cleanupState() error {
	tempPath := filepath.Join(os.TempDir(), TempDir)

	// Remove the entire temp directory
	if err := os.RemoveAll(tempPath); err != nil {
		// Don't log during init as logger may not be initialized
		return err
	}

	// Don't log during init as logger may not be initialized
	return nil
}

// saveEnvironment captures important environment variables
func saveEnvironment() map[string]string {
	env := make(map[string]string)

	// List of environment variables to preserve
	varsToSave := []string{
		"CONFIG_PATH",
		"LOG_LEVEL",
		"LOG_PATH",
		"LOG_NOCOLOR",
		"CONSOLE_GMCP_OUTPUT",
	}

	for _, key := range varsToSave {
		if value := os.Getenv(key); value != "" {
			env[key] = value
		}
	}

	return env
}

// restoreEnvironment sets environment variables from saved state
func restoreEnvironment(env map[string]string) {
	for key, value := range env {
		os.Setenv(key, value)
	}
}

// isRecovering checks if we're recovering from a copyover
func isRecovering() bool {
	// Check environment variable
	if os.Getenv(EnvCopyover) == "1" {
		return true
	}

	// Check if state file exists
	stateFile := filepath.Join(os.TempDir(), TempDir, StateFile)
	if _, err := os.Stat(stateFile); err == nil {
		return true
	}

	return false
}
