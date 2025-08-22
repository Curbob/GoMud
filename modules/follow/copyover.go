package follow

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// FollowCopyoverState represents follow relationships during copyover
type FollowCopyoverState struct {
	// Map of who's following whom
	Followers map[string]string `json:"followers"` // follower_id -> target_id
	// Map of follow time limits
	FollowLimits map[string]uint64 `json:"follow_limits"` // follower_id -> round cutoff
}

var followModuleInstance *FollowModule

func init() {
	// Register with copyover system
	copyover.Register("follow", gatherFollowState, restoreFollowState)
}

// SetFollowModuleInstance stores the module instance for copyover
func SetFollowModuleInstance(fm *FollowModule) {
	followModuleInstance = fm
}

// gatherFollowState collects follow relationships before copyover
func gatherFollowState() (interface{}, error) {
	if followModuleInstance == nil {
		mudlog.Info("Copyover", "subsystem", "follow", "status", "no active instance")
		return nil, nil
	}

	state := FollowCopyoverState{
		Followers:    make(map[string]string),
		FollowLimits: make(map[string]uint64),
	}

	// Convert follow relationships to strings for serialization
	for follower, target := range followModuleInstance.followers {
		followerKey := followIdToString(follower)
		targetKey := followIdToString(target)
		state.Followers[followerKey] = targetKey
	}

	// Save follow limits
	for follower, limit := range followModuleInstance.followLimits {
		followerKey := followIdToString(follower)
		state.FollowLimits[followerKey] = limit
	}

	mudlog.Info("Copyover", "subsystem", "follow",
		"gathered", len(state.Followers), "relationships",
		"limits", len(state.FollowLimits))

	return state, nil
}

// restoreFollowState restores follow relationships after copyover
func restoreFollowState(data interface{}) error {
	if data == nil {
		mudlog.Info("Copyover", "subsystem", "follow", "status", "no state to restore")
		return nil
	}

	if followModuleInstance == nil {
		mudlog.Warn("Copyover", "subsystem", "follow", "warning", "no module instance to restore to")
		return nil
	}

	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal follow data: %w", err)
	}

	var state FollowCopyoverState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal follow state: %w", err)
	}

	// Clear existing relationships
	followModuleInstance.followed = make(map[followId][]followId)
	followModuleInstance.followers = make(map[followId]followId)
	followModuleInstance.followLimits = make(map[followId]uint64)

	// Restore follow relationships
	for followerStr, targetStr := range state.Followers {
		follower := stringToFollowId(followerStr)
		target := stringToFollowId(targetStr)

		// Restore the follower -> target relationship
		followModuleInstance.followers[follower] = target

		// Rebuild the followed map (target -> list of followers)
		if _, ok := followModuleInstance.followed[target]; !ok {
			followModuleInstance.followed[target] = []followId{}
		}
		followModuleInstance.followed[target] = append(followModuleInstance.followed[target], follower)
	}

	// Restore follow limits
	for followerStr, limit := range state.FollowLimits {
		follower := stringToFollowId(followerStr)
		followModuleInstance.followLimits[follower] = limit
	}

	mudlog.Info("Copyover", "subsystem", "follow",
		"restored", len(state.Followers), "relationships",
		"limits", len(state.FollowLimits))

	return nil
}

// Helper functions to convert followId to/from string for serialization
func followIdToString(id followId) string {
	if id.userId > 0 {
		return fmt.Sprintf("u:%d", id.userId)
	}
	return fmt.Sprintf("m:%d", id.mobInstanceId)
}

func stringToFollowId(s string) followId {
	var id followId
	var val int

	if n, err := fmt.Sscanf(s, "u:%d", &val); n == 1 && err == nil {
		id.userId = val
	} else if n, err := fmt.Sscanf(s, "m:%d", &val); n == 1 && err == nil {
		id.mobInstanceId = val
	}

	return id
}
