package gmcp

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GMCPState represents GMCP subscription state during copyover
type GMCPState struct {
	// Connection settings per connection ID
	ConnectionSettings map[uint64]GMCPSettings `json:"connection_settings"`

	// Track which users are in combat
	CombatUsers map[int]struct{} `json:"combat_users"`

	// Combat module states per user
	CooldownStates map[int]CooldownState `json:"cooldown_states,omitempty"`
	DamageStates   map[int]DamageState   `json:"damage_states,omitempty"`
	EnemiesStates  map[int]EnemiesState  `json:"enemies_states,omitempty"`
	EventsStates   map[int]EventsState   `json:"events_states,omitempty"`
	StatusStates   map[int]StatusState   `json:"status_states,omitempty"`
	TargetStates   map[int]TargetState   `json:"target_states,omitempty"`
}

// Module-specific state structures
type CooldownState struct {
	LastSeconds float64 `json:"last_seconds"`
	MaxSeconds  float64 `json:"max_seconds"`
}

type DamageState struct {
	RecentDamage []DamageEntry `json:"recent_damage"`
}

type DamageEntry struct {
	Amount    int    `json:"amount"`
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
}

type EnemiesState struct {
	Enemies []CopyoverEnemyInfo `json:"enemies"`
}

type CopyoverEnemyInfo struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	HpPercent int    `json:"hp_percent"`
}

type EventsState struct {
	RecentEvents []string `json:"recent_events"`
}

type StatusState struct {
	InCombat    bool `json:"in_combat"`
	RoundNumber int  `json:"round_number"`
}

type TargetState struct {
	TargetId    string `json:"target_id"`
	TargetName  string `json:"target_name"`
	TargetHp    int    `json:"target_hp"`
	TargetMaxHp int    `json:"target_max_hp"`
}

func init() {
	// Register with copyover system
	copyover.Register("gmcp", gatherGMCPState, restoreGMCPState)
}

// gatherGMCPState collects GMCP state before copyover
func gatherGMCPState() (interface{}, error) {
	state := GMCPState{
		ConnectionSettings: make(map[uint64]GMCPSettings),
		CombatUsers:        make(map[int]struct{}),
		CooldownStates:     make(map[int]CooldownState),
		DamageStates:       make(map[int]DamageState),
		EnemiesStates:      make(map[int]EnemiesState),
		EventsStates:       make(map[int]EventsState),
		StatusStates:       make(map[int]StatusState),
		TargetStates:       make(map[int]TargetState),
	}

	// Save all cached connection settings
	if gmcpModule.cache != nil {
		// Get all keys from the cache
		activeConnections := connections.GetActiveConnections()
		for _, conn := range activeConnections {
			connId := conn.ConnectionId()
			if settings, ok := gmcpModule.cache.Get(connId); ok {
				state.ConnectionSettings[connId] = settings
			}
		}
	}

	// Save combat users
	combatUsersMutex.RLock()
	for userId := range combatUsers {
		state.CombatUsers[userId] = struct{}{}
	}
	combatUsersMutex.RUnlock()

	// Gather combat module states for active users
	activeUsers := users.GetAllActiveUsers()
	for _, user := range activeUsers {
		if user == nil {
			continue
		}

		userId := user.UserId

		// Check if user has GMCP enabled
		if !isGMCPEnabled(user.ConnectionId()) {
			continue
		}

		// Gather Cooldown state
		state.CooldownStates[userId] = gatherCooldownState(userId)

		// Gather Damage state
		state.DamageStates[userId] = gatherDamageState(userId)

		// Gather Enemies state
		state.EnemiesStates[userId] = gatherEnemiesState(userId)

		// Gather Events state
		state.EventsStates[userId] = gatherEventsState(userId)

		// Gather Status state
		state.StatusStates[userId] = gatherStatusState(userId)

		// Gather Target state
		state.TargetStates[userId] = gatherTargetState(userId)
	}

	mudlog.Info("Copyover", "subsystem", "gmcp",
		"gathered", len(state.ConnectionSettings), "connections",
		"combat", len(state.CombatUsers), "users")

	return state, nil
}

// restoreGMCPState restores GMCP state after copyover
func restoreGMCPState(data interface{}) error {
	if data == nil {
		mudlog.Info("Copyover", "subsystem", "gmcp", "status", "no state to restore")
		return nil
	}

	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal GMCP data: %w", err)
	}

	var state GMCPState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal GMCP state: %w", err)
	}

	// Restore connection settings to cache
	if gmcpModule.cache != nil {
		for connId, settings := range state.ConnectionSettings {
			gmcpModule.cache.Add(connId, settings)
			mudlog.Info("Copyover", "subsystem", "gmcp",
				"restored", "connection settings",
				"connId", connId,
				"client", settings.Client.Name)
		}
	}

	// Restore combat users
	combatUsersMutex.Lock()
	combatUsers = state.CombatUsers
	combatUsersMutex.Unlock()

	// Restore combat module states
	for userId, cooldown := range state.CooldownStates {
		restoreCooldownState(userId, cooldown)
	}

	for userId, damage := range state.DamageStates {
		restoreDamageState(userId, damage)
	}

	for userId, enemies := range state.EnemiesStates {
		restoreEnemiesState(userId, enemies)
	}

	for userId, events := range state.EventsStates {
		restoreEventsState(userId, events)
	}

	for userId, status := range state.StatusStates {
		restoreStatusState(userId, status)
	}

	for userId, target := range state.TargetStates {
		restoreTargetState(userId, target)
	}

	// Re-send initial GMCP data to reconnected users
	for _, user := range users.GetAllActiveUsers() {
		if user == nil {
			continue
		}

		if !isGMCPEnabled(user.ConnectionId()) {
			continue
		}

		// Send Core.Hello
		sendGMCPOut(user.UserId, "Core.Hello", GMCPHello{
			Client:  "GoMud",
			Version: "1.0",
		})

		// Send character data
		sendCharacterUpdate(user.UserId)

		// Send room data if user is in a room
		if user.Character.RoomId > 0 {
			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
				sendRoomUpdate(user.UserId, room)
			}
		}

		// Send combat status if in combat
		if _, inCombat := state.CombatUsers[user.UserId]; inCombat {
			if status, exists := state.StatusStates[user.UserId]; exists {
				sendGMCPOut(user.UserId, "Char.Combat.Status", status)
			}
		}
	}

	mudlog.Info("Copyover", "subsystem", "gmcp",
		"restored", len(state.ConnectionSettings), "connections",
		"combat", len(state.CombatUsers), "users")

	return nil
}

// Helper functions to gather module states
func gatherCooldownState(userId int) CooldownState {
	// This would gather actual cooldown state from the cooldown module
	// For now, return empty state
	return CooldownState{}
}

func gatherDamageState(userId int) DamageState {
	// This would gather actual damage history
	return DamageState{}
}

func gatherEnemiesState(userId int) EnemiesState {
	// This would gather current enemy list
	return EnemiesState{}
}

func gatherEventsState(userId int) EventsState {
	// This would gather recent combat events
	return EventsState{}
}

func gatherStatusState(userId int) StatusState {
	// Check if user is in combat
	combatUsersMutex.RLock()
	_, inCombat := combatUsers[userId]
	combatUsersMutex.RUnlock()

	return StatusState{
		InCombat: inCombat,
	}
}

func gatherTargetState(userId int) TargetState {
	// This would gather current target info
	user := users.GetByUserId(userId)
	if user == nil || user.Character.Aggro == nil {
		return TargetState{}
	}

	state := TargetState{}

	// Get target information
	if user.Character.Aggro.UserId > 0 {
		if target := users.GetByUserId(user.Character.Aggro.UserId); target != nil {
			state.TargetId = fmt.Sprintf("user:%d", target.UserId)
			state.TargetName = target.Character.Name
			state.TargetHp = target.Character.Health
			state.TargetMaxHp = target.Character.HealthMax.ValueAdj
		}
	} else if user.Character.Aggro.MobInstanceId > 0 {
		if target := mobs.GetInstance(user.Character.Aggro.MobInstanceId); target != nil {
			state.TargetId = fmt.Sprintf("mob:%d", target.InstanceId)
			state.TargetName = target.Character.Name
			state.TargetHp = target.Character.Health
			state.TargetMaxHp = target.Character.HealthMax.ValueAdj
		}
	}

	return state
}

// Helper functions to restore module states
func restoreCooldownState(userId int, state CooldownState) {
	// This would restore cooldown state
	// Implementation depends on cooldown module internals
}

func restoreDamageState(userId int, state DamageState) {
	// This would restore damage history
}

func restoreEnemiesState(userId int, state EnemiesState) {
	// This would restore enemy list
}

func restoreEventsState(userId int, state EventsState) {
	// This would restore recent events
}

func restoreStatusState(userId int, state StatusState) {
	// Status is mostly derived, so minimal restoration needed
}

func restoreTargetState(userId int, state TargetState) {
	// Target state would be re-established through combat system
}

// Helper functions for sending GMCP updates
func sendGMCPOut(userId int, module string, data interface{}) {
	// This would use the actual GMCP sending mechanism
	// Simplified for copyover
}

func sendCharacterUpdate(userId int) {
	// Send character info, stats, vitals, etc.
	// This would call the actual character update functions
}

func sendRoomUpdate(userId int, room *rooms.Room) {
	// Send room info
	// This would call the actual room update functions
}
