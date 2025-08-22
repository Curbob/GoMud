package usercommands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Copyover handles the copyover admin command
func Copyover(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Get the copyover manager
	manager := copyover.GetManager()

	// Parse arguments
	args := strings.Fields(strings.ToLower(rest))

	// No arguments - show help
	if len(args) == 0 {
		user.SendText(`<ansi fg="yellow">Copyover Command Usage:</ansi>`)
		user.SendText(`  <ansi fg="command">copyover <seconds></ansi> - Schedule copyover with countdown`)
		user.SendText(`  <ansi fg="command">copyover now</ansi> - Immediate copyover (no countdown)`)
		user.SendText(`  <ansi fg="command">copyover cancel</ansi> - Cancel scheduled copyover`)
		user.SendText(`  <ansi fg="command">copyover status</ansi> - Check copyover status`)
		user.SendText(``)
		user.SendText(`<ansi fg="cyan">What is copyover?</ansi>`)
		user.SendText(`Copyover performs a hot-reload of the server, preserving all`)
		user.SendText(`player connections. The server restarts without disconnecting`)
		user.SendText(`anyone, allowing code updates without disrupting gameplay.`)
		return true, nil
	}

	// Handle different subcommands
	switch args[0] {
	case "cancel":
		return copyoverCancel(user, manager)

	case "status":
		return copyoverStatus(user, manager)

	case "now":
		// For immediate copyover, just initiate without confirmation prompt
		// The confirmation system would be too complex to implement properly
		return copyoverImmediate(user, manager)

	default:
		// Try to parse as countdown seconds
		seconds, err := strconv.Atoi(args[0])
		if err != nil {
			user.SendText(fmt.Sprintf(`<ansi fg="red">Invalid option: %s</ansi>`, args[0]))
			user.SendText(`Use <ansi fg="command">copyover</ansi> without arguments for help.`)
			return true, nil
		}

		// Validate countdown range
		if seconds < 10 {
			user.SendText(`<ansi fg="red">Countdown must be at least 10 seconds.</ansi>`)
			return true, nil
		}
		if seconds > 300 {
			user.SendText(`<ansi fg="red">Countdown cannot exceed 300 seconds (5 minutes).</ansi>`)
			return true, nil
		}

		return copyoverSchedule(user, manager, seconds)
	}
}

func copyoverCancel(user *users.UserRecord, manager *copyover.Manager) (bool, error) {
	if err := manager.Cancel(); err != nil {
		user.SendText(fmt.Sprintf(`<ansi fg="red">Cannot cancel: %s</ansi>`, err.Error()))
		return true, nil
	}

	user.SendText(`<ansi fg="green">Copyover cancelled successfully.</ansi>`)

	// Log the cancellation
	logAdminCommand(user.UserId, user.Character.Name, "copyover cancel")

	return true, nil
}

func copyoverStatus(user *users.UserRecord, manager *copyover.Manager) (bool, error) {
	status := manager.GetStatus()

	user.SendText(`<ansi fg="yellow">═══════════════════════════════════</ansi>`)
	user.SendText(`<ansi fg="yellow-bold">       COPYOVER STATUS</ansi>`)
	user.SendText(`<ansi fg="yellow">═══════════════════════════════════</ansi>`)

	// Show current state
	stateColor := "cyan"
	switch status.State {
	case copyover.StatePreparing:
		stateColor = "yellow"
	case copyover.StateRecovering:
		stateColor = "green"
	}
	user.SendText(fmt.Sprintf(`<ansi fg="white">State:</ansi> <ansi fg="%s-bold">%s</ansi>`, stateColor, status.State))

	// Show scheduled time if applicable
	if !status.ScheduledFor.IsZero() {
		remaining := status.ScheduledFor.Sub(time.Now())
		if remaining > 0 {
			user.SendText(fmt.Sprintf(`<ansi fg="white">Scheduled:</ansi> <ansi fg="yellow">%s</ansi> <ansi fg="white">(in %d seconds)</ansi>`,
				status.ScheduledFor.Format("15:04:05"),
				int(remaining.Seconds())))
		}
	}

	// Show progress if in progress
	if status.Progress > 0 {
		// Create progress bar
		barWidth := 20
		filled := (status.Progress * barWidth) / 100
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		user.SendText(fmt.Sprintf(`<ansi fg="white">Progress:</ansi> <ansi fg="green">%s</ansi> <ansi fg="white">%d%%</ansi>`, bar, status.Progress))

		if status.Message != "" {
			user.SendText(fmt.Sprintf(`<ansi fg="white">Status:</ansi> <ansi fg="cyan">%s</ansi>`, status.Message))
		}
	}

	// Show registered subsystems
	subsystems := copyover.GetRegisteredSubsystems()
	if len(subsystems) > 0 {
		user.SendText(fmt.Sprintf(`<ansi fg="white">Subsystems:</ansi> <ansi fg="magenta">%d registered</ansi>`, len(subsystems)))
		for _, name := range subsystems {
			user.SendText(fmt.Sprintf(`  <ansi fg="white">•</ansi> <ansi fg="cyan">%s</ansi>`, name))
		}
	}

	user.SendText(`<ansi fg="yellow">═══════════════════════════════════</ansi>`)

	return true, nil
}

func copyoverImmediate(user *users.UserRecord, manager *copyover.Manager) (bool, error) {
	// Immediate copyover with warning
	user.SendText(`<ansi fg="red-bold">*** COPYOVER INITIATED ***</ansi>`)
	user.SendText(`<ansi fg="yellow">Server will restart immediately!</ansi>`)

	options := copyover.Options{
		Countdown:   0, // Immediate
		Reason:      fmt.Sprintf("Initiated by %s", user.Character.Name),
		InitiatedBy: user.UserId,
	}

	if err := manager.Initiate(options); err != nil {
		user.SendText(fmt.Sprintf(`<ansi fg="red">Copyover failed: %s</ansi>`, err.Error()))
		return true, nil
	}

	// Log the action
	logAdminCommand(user.UserId, user.Character.Name, "copyover now")

	return true, nil
}

func copyoverSchedule(user *users.UserRecord, manager *copyover.Manager, seconds int) (bool, error) {
	options := copyover.Options{
		Countdown:   seconds,
		Reason:      fmt.Sprintf("Scheduled by %s", user.Character.Name),
		InitiatedBy: user.UserId,
	}

	if err := manager.Initiate(options); err != nil {
		user.SendText(fmt.Sprintf(`<ansi fg="red">Failed to schedule copyover: %s</ansi>`, err.Error()))
		return true, nil
	}

	user.SendText(fmt.Sprintf(`<ansi fg="green">Copyover scheduled in %d seconds.</ansi>`, seconds))
	user.SendText(`Use <ansi fg="command">copyover cancel</ansi> to abort if needed.`)

	// Log the action
	logAdminCommand(user.UserId, user.Character.Name, fmt.Sprintf("copyover %d", seconds))

	return true, nil
}

// Helper function to log admin commands
func logAdminCommand(userId int, userName string, command string) {
	// Log to mudlog for now
	// In the future, could add proper admin command logging
	mudlog.Info("AdminCommand", "userId", userId, "user", userName, "command", command)
}
