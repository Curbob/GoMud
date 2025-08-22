package mobs

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// MobState represents a single mob's state for copyover
type MobState struct {
	InstanceId     int                    `json:"instance_id"`
	MobId          MobId                  `json:"mob_id"`
	RoomId         int                    `json:"room_id"`
	Health         int                    `json:"health"`
	Mana           int                    `json:"mana"`
	CurrentCommand string                 `json:"current_command,omitempty"`
	LastCommand    string                 `json:"last_command,omitempty"`
	Charmed        bool                   `json:"charmed,omitempty"`
	CharmedUserId  int                    `json:"charmed_user_id,omitempty"`
	Aggro          map[int]int            `json:"aggro,omitempty"` // userId -> damage
	CustomData     map[string]interface{} `json:"custom_data,omitempty"`
}

// MobCopyoverState represents all mob state during copyover
type MobCopyoverState struct {
	ActiveMobs []MobState `json:"active_mobs"`
	NextId     int        `json:"next_id"` // Next instance ID to use
}

func init() {
	// Register the mobs subsystem with copyover
	copyover.Register("mobs", gatherMobState, restoreMobState)
}

// gatherMobState collects state about active mobs
func gatherMobState() (interface{}, error) {
	state := MobCopyoverState{
		ActiveMobs: make([]MobState, 0),
		NextId:     GetNextInstanceId(),
	}

	// Get all active mobs
	allMobs := GetAllMobs()

	for _, mob := range allMobs {
		if mob == nil {
			continue
		}

		// Create mob state
		mobState := MobState{
			InstanceId:     mob.InstanceId,
			MobId:          mob.MobId,
			RoomId:         mob.Character.RoomId,
			Health:         mob.Character.Health,
			Mana:           mob.Character.Mana,
			CurrentCommand: mob.CurrentCommand(),
			LastCommand:    mob.LastCommand(),
		}

		// Check if charmed
		if mob.Character.IsCharmed() {
			mobState.Charmed = true
			mobState.CharmedUserId = mob.Character.GetCharmedUserId()
		}

		// Save aggro table
		if mob.Character.Aggro != nil && mob.Character.Aggro.UserId > 0 {
			mobState.Aggro = make(map[int]int)
			// Simplified - just track the main aggro target
			mobState.Aggro[mob.Character.Aggro.UserId] = 1
		}

		// Save any custom data (for scripted mobs)
		if mob.HasCustomData() {
			mobState.CustomData = mob.GetCustomData()
		}

		state.ActiveMobs = append(state.ActiveMobs, mobState)
	}

	mudlog.Info("Copyover", "subsystem", "mobs", "gathered", len(state.ActiveMobs))
	return state, nil
}

// restoreMobState restores mob state after copyover
func restoreMobState(data interface{}) error {
	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal mob data: %w", err)
	}

	var state MobCopyoverState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal mob state: %w", err)
	}

	// Restore next instance ID
	SetNextInstanceId(state.NextId)

	// Restore all mobs
	restored := 0
	for _, mobState := range state.ActiveMobs {
		// Load the mob spec
		mobSpec := GetMobSpec(mobState.MobId)
		if mobSpec == nil {
			mudlog.Warn("Copyover", "subsystem", "mobs", "warning", "Mob spec not found", "mobId", mobState.MobId)
			continue
		}

		// Create new mob instance
		mob := NewMobById(mobState.MobId, mobState.RoomId)
		if mob == nil {
			mudlog.Error("Copyover", "subsystem", "mobs", "error", "Failed to create mob", "mobId", mobState.MobId)
			continue
		}

		// Restore instance ID
		mob.InstanceId = mobState.InstanceId

		// Restore health/mana
		mob.Character.Health = mobState.Health
		mob.Character.Mana = mobState.Mana

		// Restore commands
		if mobState.CurrentCommand != "" {
			mob.SetCommand(mobState.CurrentCommand)
		}

		// Restore charmed state
		if mobState.Charmed && mobState.CharmedUserId > 0 {
			mob.Character.Charmed.UserId = mobState.CharmedUserId
		}

		// Restore aggro
		if len(mobState.Aggro) > 0 {
			for userId, damage := range mobState.Aggro {
				mob.Character.TrackPlayerDamage(userId, damage)
			}
		}

		// Restore custom data
		if len(mobState.CustomData) > 0 {
			mob.SetCustomData(mobState.CustomData)
		}

		// Place mob in room
		PlaceMobInRoom(mob, mobState.RoomId)

		restored++
		mudlog.Info("Copyover", "subsystem", "mobs", "restored", mob.Character.Name, "instanceId", mob.InstanceId, "room", mobState.RoomId)
	}

	mudlog.Info("Copyover", "subsystem", "mobs", "restored", restored, "total", len(state.ActiveMobs))
	return nil
}

// Helper methods - these would need to be implemented or already exist

// GetNextInstanceId returns the next available instance ID
func GetNextInstanceId() int {
	// This would return the next ID from the mob manager
	return 0
}

// SetNextInstanceId sets the next instance ID to use
func SetNextInstanceId(id int) {
	// This would set the next ID in the mob manager
}

// GetAllMobs returns all active mob instances
func GetAllMobs() []*Mob {
	// This would return all mobs from the mob manager
	return []*Mob{}
}

// PlaceMobInRoom places a mob in a specific room
func PlaceMobInRoom(mob *Mob, roomId int) {
	// This would add the mob to the room's mob list
}

// CurrentCommand returns the mob's current command
func (m *Mob) CurrentCommand() string {
	// Return current command if any
	return ""
}

// LastCommand returns the mob's last executed command
func (m *Mob) LastCommand() string {
	// Return last command if any
	return ""
}

// SetCommand sets the mob's current command
func (m *Mob) SetCommand(cmd string) {
	// Set the command to execute
}

// HasCustomData checks if mob has custom scripted data
func (m *Mob) HasCustomData() bool {
	return false
}

// GetCustomData returns mob's custom data
func (m *Mob) GetCustomData() map[string]interface{} {
	return nil
}

// SetCustomData sets mob's custom data
func (m *Mob) SetCustomData(data map[string]interface{}) {
	// Set custom data
}
