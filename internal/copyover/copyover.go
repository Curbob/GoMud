package copyover

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// BroadcastFunc is a callback for sending messages to all users
type BroadcastFunc func(message string)

// PreCopyoverFunc is a callback called before copyover execution
type PreCopyoverFunc func() error

// Manager handles the copyover process
type Manager struct {
	mu               sync.RWMutex
	state            *stateMachine
	timer            *time.Timer
	cancelChan       chan struct{}
	initiatedBy      int
	reason           string
	broadcastFunc    BroadcastFunc
	preCopyoverHooks []PreCopyoverFunc
	buildOnCopyover  bool // Whether to rebuild during this copyover
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// GetManager returns the singleton copyover manager
func GetManager() *Manager {
	managerOnce.Do(func() {
		initialState := StateIdle
		if isRecovering() {
			initialState = StateRecovering
		}

		manager = &Manager{
			state: &stateMachine{
				currentState: initialState,
			},
		}

		// Clean up stale state file if not recovering
		if !isRecovering() {
			cleanupState()
		}
	})
	return manager
}

// SetBroadcastFunc sets the callback for broadcasting messages
func (m *Manager) SetBroadcastFunc(f BroadcastFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcastFunc = f
}

// RegisterPreCopyoverHook registers a function to be called before copyover
func (m *Manager) RegisterPreCopyoverHook(f PreCopyoverFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preCopyoverHooks = append(m.preCopyoverHooks, f)
}

// RegisterPreCopyoverHook is a convenience function to register a pre-copyover hook on the global manager
func RegisterPreCopyoverHook(f PreCopyoverFunc) {
	GetManager().RegisterPreCopyoverHook(f)
}

// Initiate starts the copyover process
func (m *Manager) Initiate(options Options) error {
	m.mu.Lock()

	// Check current state
	if m.state.currentState != StateIdle {
		m.mu.Unlock()
		return fmt.Errorf("copyover already in progress (state: %s)", m.state.currentState)
	}

	// Check vetoes
	if canProceed, vetoes := checkVetoes(); !canProceed {
		reasons := ""
		for _, v := range vetoes {
			reasons += fmt.Sprintf("\n  - %s: %s", v.Subsystem, v.Reason)
		}
		m.mu.Unlock()
		return fmt.Errorf("copyover vetoed by subsystems:%s", reasons)
	}

	// Store options
	m.initiatedBy = options.InitiatedBy
	m.reason = options.Reason
	m.buildOnCopyover = options.Build

	// Handle countdown
	if options.Countdown > 0 {
		m.state.scheduledFor = time.Now().Add(time.Duration(options.Countdown) * time.Second)
		m.cancelChan = make(chan struct{})
		m.timer = time.NewTimer(time.Duration(options.Countdown) * time.Second)

		// Announce scheduling
		m.announce("copyover-scheduled", map[string]interface{}{
			"Seconds": options.Countdown,
			"Time":    m.state.scheduledFor.Format("15:04:05"),
			"Reason":  options.Reason,
		})

		// Start countdown in background
		go m.countdown(options)
		m.mu.Unlock()
		return nil
	}

	// Immediate execution - use the same countdown function but with a timer that fires immediately
	m.timer = time.NewTimer(1 * time.Millisecond) // Fire almost immediately
	m.cancelChan = make(chan struct{})

	// Use the same countdown path for consistency
	go m.countdown(options)
	m.mu.Unlock()
	return nil
}

// Cancel cancels a scheduled copyover
func (m *Manager) Cancel() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.currentState != StateIdle || m.timer == nil {
		return fmt.Errorf("no scheduled copyover to cancel")
	}

	// Stop timer
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}

	// Signal cancellation
	if m.cancelChan != nil {
		close(m.cancelChan)
		m.cancelChan = nil
	}

	// Reset state
	m.state.scheduledFor = time.Time{}

	// Announce cancellation
	m.announce("copyover-cancelled", nil)

	mudlog.Info("Copyover", "action", "Cancelled")
	return nil
}

// GetStatus returns the current copyover status
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Status{
		State:        m.state.currentState,
		ScheduledFor: m.state.scheduledFor,
		Progress:     m.state.progress,
		Message:      m.state.message,
	}
}

// Recover handles recovery after a copyover restart
func (m *Manager) Recover() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.currentState != StateRecovering {
		return fmt.Errorf("not in recovery state")
	}

	mudlog.Info("Copyover", "status", "Starting recovery")

	// Note: We don't lock the MUD during recovery as the server
	// is just starting up and there's no concurrent activity yet

	// Load saved state
	data, err := loadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Restore environment
	restoreEnvironment(data.Environment)

	// Restore subsystem states
	if err := restoreStates(data.Subsystems); err != nil {
		mudlog.Error("Copyover", "error", "Failed to restore some subsystems", "err", err)
		// Continue anyway - partial recovery is better than none
	}

	// Clean up state file
	cleanupState()

	// Transition to idle
	m.state.transition(StateIdle)

	// Announce recovery complete
	m.announce("copyover-complete", nil)

	mudlog.Info("Copyover", "status", "Recovery complete")
	return nil
}

// IsRecovering returns true if the system is recovering from copyover
func IsRecovering() bool {
	return isRecovering()
}

// Recover performs copyover recovery - called from main.go
func Recover() error {
	return GetManager().Recover()
}

// Private methods

func (m *Manager) countdown(options Options) {
	// Determine ticker interval based on countdown duration
	interval := 10 * time.Second
	if options.Countdown <= 30 {
		interval = 5 * time.Second
	}
	if options.Countdown <= 15 {
		interval = 3 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Keep track of last announcement to avoid spam
	lastAnnounced := options.Countdown

	for {
		select {
		case <-m.timer.C:
			// Timer expired, execute copyover
			m.mu.Lock()
			m.timer = nil
			m.cancelChan = nil
			m.mu.Unlock()

			if err := m.execute(); err != nil {
				mudlog.Error("Copyover", "error", "Execution failed", "err", err)
				m.announce("copyover-failed", map[string]interface{}{
					"Error": err.Error(),
				})

				m.mu.Lock()
				m.state.transition(StateIdle)
				m.mu.Unlock()
			}
			return

		case <-ticker.C:
			// Countdown announcement
			remaining := int(time.Until(m.state.scheduledFor).Seconds())
			if remaining > 0 && remaining < lastAnnounced {
				// Announce at specific intervals: 60, 50, 40, 30, 20, 10, 5, 3, 2, 1
				shouldAnnounce := false
				if remaining <= 10 || // Every second for last 10
					remaining == 20 || remaining == 30 ||
					remaining == 40 || remaining == 50 || remaining == 60 ||
					remaining == 120 || remaining == 180 || remaining == 240 || remaining == 300 {
					shouldAnnounce = true
				}

				if shouldAnnounce {
					m.announce("copyover-countdown", map[string]interface{}{
						"Seconds": remaining,
					})
					lastAnnounced = remaining
				}
			}

		case <-m.cancelChan:
			// Cancelled
			return
		}
	}
}

func (m *Manager) execute() error {
	// Transition to preparing state
	if err := m.state.transition(StatePreparing); err != nil {
		return err
	}

	m.updateProgress(10, "Preparing for copyover")

	// Note: We don't use util.LockMud() here because it would block
	// all game processing including our own execution.
	// The subsystems should handle their own synchronization.

	// Execute pre-copyover hooks
	m.updateProgress(20, "Running pre-copyover hooks")
	for i, hook := range m.preCopyoverHooks {
		if hook != nil {
			if err := hook(); err != nil {
				mudlog.Error("Copyover", "error", "Pre-copyover hook failed", "hookIndex", i, "err", err)
				// Continue anyway - don't fail the whole copyover for a hook failure
			}
		}
	}

	// Save all users - done by user subsystem gatherer
	m.updateProgress(30, "Saving state")

	// Save all rooms to ensure persistence
	// Note: SaveAllRooms must be called via pre-copyover hooks from a higher level
	// to avoid import cycles. The rooms subsystem will handle its own state gathering.
	mudlog.Info("Copyover", "action", "Preparing to save game state")

	// Gather subsystem states
	m.updateProgress(50, "Gathering state")
	states, err := gatherStates()
	if err != nil {
		return fmt.Errorf("failed to gather states: %w", err)
	}

	// Create copyover data
	data := &CopyoverData{
		Version:     "1.0",
		Timestamp:   time.Now(),
		Environment: saveEnvironment(),
		Subsystems:  states,
	}

	// Save state to disk
	m.updateProgress(70, "Saving state")
	if err := saveState(data); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Build new executable (optional)
	if m.shouldBuild() {
		m.updateProgress(80, "Building executable")
		if err := m.buildExecutable(); err != nil {
			return fmt.Errorf("failed to build: %w", err)
		}
	}

	// Send the restarting message BEFORE we prepare file descriptors
	m.announce("copyover-restarting", nil)

	// Give time for the message to be sent
	time.Sleep(100 * time.Millisecond)

	// Execute new process
	m.updateProgress(95, "Restarting server")

	// Notify systems to prepare for shutdown
	// This would be handled by event subsystem's gather function

	if err := m.executeNewProcess(); err != nil {
		m.state.transition(StateIdle)
		return fmt.Errorf("failed to execute: %w", err)
	}

	// Should never reach here
	return fmt.Errorf("exec returned unexpectedly")
}

func (m *Manager) updateProgress(percent int, message string) {
	m.mu.Lock()
	m.state.progress = percent
	m.state.message = message
	m.mu.Unlock()

	mudlog.Info("Copyover", "progress", percent, "message", message)
}

func (m *Manager) buildExecutable() error {
	// Get the current executable name
	currentExe, err := os.Executable()
	if err != nil {
		// Fallback to default name if we can't determine current executable
		currentExe = "./go-mud-server"
		mudlog.Warn("Copyover", "warning", "Could not determine current executable, using default", "err", err)
	}

	mudlog.Info("Copyover", "status", "Building executable", "target", currentExe)

	// Try to use make build first, but specify the output file
	// Extract just the filename from the path
	exeName := filepath.Base(currentExe)

	// First try make with the specific output name using BIN parameter
	// Don't output build commands to stdout/stderr - only log errors
	cmd := exec.Command("make", "build", fmt.Sprintf("BIN=%s", exeName))

	// Capture output instead of displaying it
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If make doesn't support OUTPUT param or fails, try direct go build
		mudlog.Warn("Copyover", "warning", "make build failed, trying direct go build", "err", err, "output", string(output))

		cmd = exec.Command("go", "build", "-trimpath", "-o", currentExe)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		// Capture output for this as well
		output, err := cmd.CombinedOutput()
		if err != nil {
			mudlog.Error("Copyover", "error", "Build failed", "output", string(output))
			return fmt.Errorf("failed to build executable %s: %w", currentExe, err)
		}
	}

	return nil
}

func (m *Manager) executeNewProcess() error {
	// Get the current executable - this is what we should replace
	executable, err := os.Executable()
	if err != nil {
		// If we can't determine current executable, try the default
		executable = "./go-mud-server"
		mudlog.Warn("Copyover", "warning", "Could not determine current executable, trying default", "err", err)

		// Check if the default exists
		if _, err := os.Stat(executable); err != nil {
			return fmt.Errorf("cannot find executable to exec: %w", err)
		}
	}

	// Prepare file descriptors for passing
	extraFiles, err := PrepareFileDescriptors()
	if err != nil {
		return fmt.Errorf("failed to prepare file descriptors: %w", err)
	}

	mudlog.Info("Copyover", "status", "Executing new process", "exe", executable, "extraFiles", len(extraFiles))

	// Prepare environment with copyover flag and FD count
	env := append(os.Environ(),
		fmt.Sprintf("%s=1", EnvCopyover),
		fmt.Sprintf("GOMUD_FD_COUNT=%d", len(extraFiles)))

	// Prepare file descriptors for exec
	// We need to convert our extraFiles to raw FDs
	files := make([]*os.File, 3+len(extraFiles)) // stdin, stdout, stderr + extras
	files[0] = os.Stdin
	files[1] = os.Stdout
	files[2] = os.Stderr

	for i, f := range extraFiles {
		files[3+i] = f
	}

	// Get the process attributes
	procAttr := &os.ProcAttr{
		Files: files,
		Env:   env,
	}

	// Use StartProcess which allows us to exec
	process, err := os.StartProcess(executable, append([]string{executable}, os.Args[1:]...), procAttr)
	if err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	mudlog.Info("Copyover", "status", "New process started", "pid", process.Pid)

	// The old process should exit, letting the new one take over
	// Close our copies of the file descriptors
	for _, f := range extraFiles {
		if f != nil {
			f.Close()
		}
	}

	// Exit the current process
	os.Exit(0)
	return nil
}

func (m *Manager) announce(template string, data interface{}) {
	// Simple text announcements without template processing
	tplText := ""
	switch template {
	case "copyover-scheduled":
		if data != nil {
			if d, ok := data.(map[string]interface{}); ok {
				tplText = fmt.Sprintf("<ansi fg=\"red-bold\">*** COPYOVER SCHEDULED IN %d SECONDS ***</ansi>", d["Seconds"])
			}
		}
	case "copyover-countdown":
		if data != nil {
			if d, ok := data.(map[string]interface{}); ok {
				seconds := d["Seconds"].(int)
				color := "yellow"
				if seconds <= 10 {
					color = "red"
				} else if seconds <= 30 {
					color = "yellow-bold"
				}
				tplText = fmt.Sprintf("<ansi fg=\"%s\">*** Copyover in %d seconds ***</ansi>", color, seconds)
			}
		}
	case "copyover-cancelled":
		tplText = "<ansi fg=\"green\">*** COPYOVER CANCELLED ***</ansi>"
	case "copyover-restarting":
		tplText = "<ansi fg=\"red-bold\">*** RESTARTING NOW ***</ansi>"
	case "copyover-complete":
		tplText = "<ansi fg=\"green-bold\">*** COPYOVER COMPLETE ***</ansi>"
	case "copyover-failed":
		tplText = "<ansi fg=\"red\">*** COPYOVER FAILED ***</ansi>"
	default:
		tplText = fmt.Sprintf("<ansi fg=\"yellow\">*** %s ***</ansi>", template)
	}

	// Log the announcement
	mudlog.Info("Copyover", "announce", template)

	// Broadcast to all users using callback if available
	if tplText != "" && m.broadcastFunc != nil {
		m.broadcastFunc(tplText + "\n")
	}
}

func (m *Manager) shouldBuild() bool {
	// Check if this specific copyover should build
	// Can be overridden by environment variable
	if os.Getenv("BUILD_ON_COPYOVER") == "0" {
		return false // Environment variable disables all builds
	}
	return m.buildOnCopyover // Use the flag from options
}
