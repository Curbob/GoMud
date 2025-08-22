package parties

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// PartyCopyoverState represents party state during copyover
type PartyCopyoverState struct {
	Parties []PartyState `json:"parties"`
}

// PartyState represents a single party's state
type PartyState struct {
	LeaderUserId  int            `json:"leader_user_id"`
	UserIds       []int          `json:"user_ids"`
	InviteUserIds []int          `json:"invite_user_ids"`
	AutoAttackers []int          `json:"auto_attackers"`
	Position      map[int]string `json:"position"`
}

func init() {
	// Register with copyover system
	copyover.Register("parties", gatherPartyState, restorePartyState)
}

// gatherPartyState collects party state before copyover
func gatherPartyState() (interface{}, error) {
	state := PartyCopyoverState{
		Parties: make([]PartyState, 0),
	}

	// Save all active parties
	for leaderId, party := range partyMap {
		partyState := PartyState{
			LeaderUserId:  party.LeaderUserId,
			UserIds:       make([]int, len(party.UserIds)),
			InviteUserIds: make([]int, len(party.InviteUserIds)),
			AutoAttackers: make([]int, len(party.AutoAttackers)),
			Position:      make(map[int]string),
		}

		// Copy slices to avoid reference issues
		copy(partyState.UserIds, party.UserIds)
		copy(partyState.InviteUserIds, party.InviteUserIds)
		copy(partyState.AutoAttackers, party.AutoAttackers)

		// Copy position map
		for userId, pos := range party.Position {
			partyState.Position[userId] = pos
		}

		state.Parties = append(state.Parties, partyState)

		mudlog.Info("Copyover", "subsystem", "parties",
			"saved", "party",
			"leader", leaderId,
			"members", len(party.UserIds))
	}

	mudlog.Info("Copyover", "subsystem", "parties",
		"gathered", len(state.Parties), "parties")

	return state, nil
}

// restorePartyState restores party state after copyover
func restorePartyState(data interface{}) error {
	if data == nil {
		mudlog.Info("Copyover", "subsystem", "parties", "status", "no state to restore")
		return nil
	}

	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal party data: %w", err)
	}

	var state PartyCopyoverState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal party state: %w", err)
	}

	// Clear existing parties
	partyMap = make(map[int]*Party)

	// Restore all parties
	for _, partyState := range state.Parties {
		party := &Party{
			LeaderUserId:  partyState.LeaderUserId,
			UserIds:       make([]int, len(partyState.UserIds)),
			InviteUserIds: make([]int, len(partyState.InviteUserIds)),
			AutoAttackers: make([]int, len(partyState.AutoAttackers)),
			Position:      make(map[int]string),
		}

		// Copy slices
		copy(party.UserIds, partyState.UserIds)
		copy(party.InviteUserIds, partyState.InviteUserIds)
		copy(party.AutoAttackers, partyState.AutoAttackers)

		// Copy position map
		for userId, pos := range partyState.Position {
			party.Position[userId] = pos
		}

		// Store in map
		partyMap[party.LeaderUserId] = party

		mudlog.Info("Copyover", "subsystem", "parties",
			"restored", "party",
			"leader", party.LeaderUserId,
			"members", len(party.UserIds))
	}

	mudlog.Info("Copyover", "subsystem", "parties",
		"restored", len(state.Parties), "parties")

	return nil
}
