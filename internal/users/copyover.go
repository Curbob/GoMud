package users

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// UserCopyoverState represents the state of logged-in users during copyover
type UserCopyoverState struct {
	// Map of connectionId to userId for reconnection
	ConnectionMap map[string]int `json:"connection_map"`
	// Active user states (for users that were online)
	ActiveUsers []int `json:"active_users"`
	// Map of userId to username for loading
	UserIdToUsername map[int]string `json:"user_id_to_username"`
}

func init() {
	// Register the users subsystem with copyover
	copyover.Register("users", gatherUserState, restoreUserState)
}

// gatherUserState collects state about active users
func gatherUserState() (interface{}, error) {
	state := UserCopyoverState{
		ConnectionMap:    make(map[string]int),
		ActiveUsers:      make([]int, 0),
		UserIdToUsername: make(map[int]string),
	}

	// Get all active users
	activeUsers := GetAllActiveUsers()

	for _, user := range activeUsers {
		if user == nil {
			continue
		}

		// Clear combat state - combat does not persist through copyover
		if user.Character.Aggro != nil {
			user.Character.Aggro = nil
			user.SendText("<ansi fg=\"yellow\">*** Combat interrupted by copyover ***</ansi>")
		}

		// Save the user data to disk
		if err := SaveUser(*user); err != nil {
			mudlog.Error("Copyover", "subsystem", "users", "error", "Failed to save user", "userId", user.UserId, "err", err)
			// Continue anyway - don't fail the whole copyover for one user
		}

		// Map connection to user
		if user.connectionId > 0 {
			state.ConnectionMap[fmt.Sprintf("%d", user.connectionId)] = user.UserId
		}

		// Track active user
		state.ActiveUsers = append(state.ActiveUsers, user.UserId)

		// Store username for loading later
		state.UserIdToUsername[user.UserId] = user.Username

		// Send copyover notification to user
		user.SendText("\n<ansi fg=\"yellow\">*** COPYOVER INITIATED - Your game state is being saved... ***</ansi>\n")
	}

	mudlog.Info("Copyover", "subsystem", "users", "gathered", len(state.ActiveUsers), "active", len(activeUsers))
	return state, nil
}

// restoreUserState restores user connections after copyover
func restoreUserState(data interface{}) error {
	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	var state UserCopyoverState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal user state: %w", err)
	}

	// Get preserved connections from copyover
	preservedConns := copyover.GetPreservedConnections()

	// Reconnect users to their connections
	reconnected := 0
	for connIdStr, userId := range state.ConnectionMap {
		// Parse connection ID
		connIdInt := 0
		fmt.Sscanf(connIdStr, "%d", &connIdInt)
		connId := connections.ConnectionId(connIdInt)

		// Check if we have this connection preserved
		if _, exists := preservedConns[connId]; exists {
			// Get username for this user ID from our saved state
			username, hasUsername := state.UserIdToUsername[userId]
			if !hasUsername {
				mudlog.Warn("Copyover", "subsystem", "users", "warning", "No username for userId", "userId", userId)
				continue
			}

			// Load the user from disk
			user, err := LoadUser(username) // Don't skip validation - we need stats recalculated
			if err != nil {
				mudlog.Error("Copyover", "subsystem", "users", "error", "Failed to load user", "username", username, "err", err)
				continue
			}

			// Clear combat state - combat does not persist through copyover
			user.Character.Aggro = nil
			user.Character.PlayerDamage = make(map[int]int)

			// Reconnect the user to their connection
			user.connectionId = connId
			userManager.Users[user.UserId] = user
			userManager.Usernames[user.Username] = user.UserId
			userManager.Connections[connId] = user.UserId
			userManager.UserConnections[user.UserId] = connId

			// Set their input round to current
			user.SetLastInputRound(util.GetRoundCount())

			// Input handlers will be set up from main.go after recovery to avoid import cycles

			mudlog.Info("Copyover", "subsystem", "users", "reconnected", user.Character.Name, "userId", userId, "connId", connIdStr)

			// Notify user of successful copyover
			user.SendText("\n<ansi fg=\"green-bold\">*** COPYOVER COMPLETE - Welcome back! ***</ansi>\n")
			user.SendText("<ansi fg=\"cyan\">The server has been successfully restarted.</ansi>\n")
			user.SendText("<ansi fg=\"yellow\">Note: Any active combat was reset during the copyover.</ansi>\n")

			// Add user back to their room's player list
			if user.Character.RoomId > 0 {
				// This is a simplified version - normally MoveUserToRoom handles this
				// but we can't import rooms package here due to circular dependency
				// The fix needs to be done after all users are restored
				user.SendText("")
				user.SendText("<ansi fg=\"yellow\">Type 'look' to see your surroundings.</ansi>\n")
			}

			reconnected++
		} else {
			mudlog.Warn("Copyover", "subsystem", "users", "warning", "Connection not preserved", "connId", connIdStr, "userId", userId)
		}
	}

	mudlog.Info("Copyover", "subsystem", "users", "restored", reconnected, "total", len(state.ConnectionMap))
	return nil
}
