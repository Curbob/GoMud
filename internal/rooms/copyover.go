package rooms

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// RoomCopyoverState represents temporary room state during copyover
type RoomCopyoverState struct {
	// Track spawned rooms that need to be preserved
	SpawnedRooms []int `json:"spawned_rooms"`
	// Track any temporary room modifications
	TempModifications map[int]map[string]interface{} `json:"temp_modifications,omitempty"`
}

func init() {
	// Register the rooms subsystem with copyover
	copyover.Register("rooms", gatherRoomState, restoreRoomState)
}

// gatherRoomState collects state about active rooms
func gatherRoomState() (interface{}, error) {
	state := RoomCopyoverState{
		SpawnedRooms:      make([]int, 0),
		TempModifications: make(map[int]map[string]interface{}),
	}

	// Get all loaded rooms
	loadedRooms := GetAllLoadedRooms()

	for _, roomId := range loadedRooms {
		room := LoadRoom(roomId)
		if room == nil {
			continue
		}

		// Track spawned/temporary rooms
		if room.IsSpawned() {
			state.SpawnedRooms = append(state.SpawnedRooms, roomId)

			// Spawned rooms would need special handling
			mudlog.Info("Copyover", "subsystem", "rooms", "info", "Found spawned room", "roomId", roomId)
		}

		// Track any temporary modifications (locked doors, etc.)
		if room.HasTemporaryState() {
			mods := make(map[string]interface{})

			// Check for locked/unlocked exits
			for exitName, exit := range room.Exits {
				if exit.Lock.IsLocked() {
					if mods["locked_exits"] == nil {
						mods["locked_exits"] = make([]string, 0)
					}
					mods["locked_exits"] = append(mods["locked_exits"].([]string), exitName)
				}
			}

			if len(mods) > 0 {
				state.TempModifications[roomId] = mods
			}
		}
	}

	mudlog.Info("Copyover", "subsystem", "rooms", "gathered", len(loadedRooms), "spawned", len(state.SpawnedRooms))
	return state, nil
}

// restoreRoomState restores room state after copyover
func restoreRoomState(data interface{}) error {
	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal room data: %w", err)
	}

	var state RoomCopyoverState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal room state: %w", err)
	}

	// Reload spawned rooms
	for _, roomId := range state.SpawnedRooms {
		room := LoadRoom(roomId)
		if room == nil {
			mudlog.Warn("Copyover", "subsystem", "rooms", "warning", "Spawned room not found", "roomId", roomId)
			continue
		}

		// Mark as spawned again
		room.MarkSpawned()
		mudlog.Info("Copyover", "subsystem", "rooms", "restored", "spawned room", "roomId", roomId)
	}

	// Restore temporary modifications
	for roomId, mods := range state.TempModifications {
		room := LoadRoom(roomId)
		if room == nil {
			continue
		}

		// Restore locked exits
		if lockedExits, ok := mods["locked_exits"].([]interface{}); ok {
			for _, exitName := range lockedExits {
				if exit, exists := room.Exits[exitName.(string)]; exists {
					exit.Lock.SetLocked()
					mudlog.Info("Copyover", "subsystem", "rooms", "restored", "locked exit", "room", roomId, "exit", exitName)
				}
			}
		}
	}

	mudlog.Info("Copyover", "subsystem", "rooms", "restored", "complete", "spawned", len(state.SpawnedRooms))
	return nil
}

// Helper methods that might not exist yet - these are stubs

// GetAllLoadedRooms returns IDs of all rooms currently in memory
func GetAllLoadedRooms() []int {
	// This would need to be implemented based on how rooms are cached
	// For now, return empty slice
	return []int{}
}

// HasTemporaryState checks if a room has temporary modifications
func (r *Room) HasTemporaryState() bool {
	// Check for any temporary state like locked doors
	for _, exit := range r.Exits {
		if exit.Lock.IsLocked() {
			return true
		}
	}
	return false
}

// IsSpawned checks if this room was dynamically created
func (r *Room) IsSpawned() bool {
	// Rooms with IDs > 100000 are typically spawned/temporary
	return r.RoomId > 100000
}

// MarkSpawned marks a room as spawned/temporary
func (r *Room) MarkSpawned() {
	// This would set a flag on the room
	// Implementation depends on room structure
}
