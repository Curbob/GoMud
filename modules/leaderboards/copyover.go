package leaderboards

import (
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func init() {
	// Register with copyover system
	// Leaderboards are recalculated periodically, so we don't need to preserve state
	copyover.Register("leaderboards", gatherLeaderboardState, restoreLeaderboardState)
}

// gatherLeaderboardState collects leaderboard state before copyover
func gatherLeaderboardState() (interface{}, error) {
	// Leaderboards are recalculated from user data periodically
	// No need to preserve state - they'll be regenerated after copyover
	mudlog.Info("Copyover", "subsystem", "leaderboards", "status", "no state to preserve")
	return nil, nil
}

// restoreLeaderboardState restores leaderboard state after copyover
func restoreLeaderboardState(data interface{}) error {
	// Nothing to restore - leaderboards will be recalculated on next round
	mudlog.Info("Copyover", "subsystem", "leaderboards", "status", "will recalculate on next round")
	return nil
}
