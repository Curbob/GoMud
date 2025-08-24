package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func init() {
	// Register listener for copyover recovery complete event
	events.RegisterListener(events.CopyoverRecoveryComplete{}, handleCopyoverRecoveryComplete)
}

// handleCopyoverRecoveryComplete applies grace to all users after copyover recovery
func handleCopyoverRecoveryComplete(e events.Event) events.ListenerReturn {
	if _, ok := e.(events.CopyoverRecoveryComplete); !ok {
		return events.Continue
	}

	mudlog.Info("Copyover", "grace", "Applying grace to all users after copyover recovery")

	// Apply grace buff to all online users
	ApplyGraceToAll()

	return events.Continue
}
