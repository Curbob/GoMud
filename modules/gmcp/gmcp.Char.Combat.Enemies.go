// Package gmcp handles Combat Enemies list updates for GMCP.
//
// Tracks all mobs targeting the player and sends updates when the enemy list changes.
// Includes metadata like which enemy is the primary target.
// Updates on round checks and immediately when enemies die or flee.
package gmcp

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GMCPCombatEnemiesUpdate is sent when the list of enemies changes
type GMCPCombatEnemiesUpdate struct {
	UserId  int
	Enemies []EnemyInfo
}

type EnemyInfo struct {
	Name      string `json:"name"`
	Id        int    `json:"id"`
	IsPrimary bool   `json:"is_primary"`
}

func (g GMCPCombatEnemiesUpdate) Type() string { return `GMCPCombatEnemiesUpdate` }

var (
	// enemiesMutex protects the enemies tracking map
	enemiesMutex sync.RWMutex

	// userEnemies tracks all enemies for each user
	userEnemies = make(map[int]map[int]bool) // userId -> map[mobInstanceId]bool
)

// NOTE: Race condition mitigated by defensive cleanup in all handlers.
// If user disconnects between validateUserForGMCP and map operations,
// the cleanup functions handle it gracefully without data corruption.

func init() {
	events.RegisterListener(GMCPCombatEnemiesUpdate{}, handleCombatEnemiesUpdate)
	events.RegisterListener(events.NewRound{}, handleEnemiesNewRound)
	events.RegisterListener(events.MobDeath{}, handleEnemiesMobDeath)
	events.RegisterListener(events.RoomChange{}, handleEnemiesRoomChange)
}

func handleCombatEnemiesUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatEnemiesUpdate)
	if !typeOk {
		mudlog.Error("GMCPCombatEnemies", "action", "handleCombatEnemiesUpdate", "error", "type assertion failed", "expectedType", "GMCPCombatEnemiesUpdate", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	_, valid := validateUserForGMCP(evt.UserId, "GMCPCombatEnemies")
	if !valid {
		return events.Continue
	}

	enemies := make([]map[string]interface{}, len(evt.Enemies))
	for i, enemy := range evt.Enemies {
		enemies[i] = map[string]interface{}{
			"name":       enemy.Name,
			"id":         enemy.Id,
			"is_primary": enemy.IsPrimary,
		}
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  "Char.Combat.Enemies",
		Payload: enemies,
	})

	return events.Continue
}

// handleEnemiesNewRound updates enemy lists each round
func handleEnemiesNewRound(e events.Event) events.ListenerReturn {
	_, typeOk := e.(events.NewRound)
	if !typeOk {
		mudlog.Error("GMCPCombatEnemies", "action", "handleEnemiesNewRound", "error", "type assertion failed", "expectedType", "events.NewRound", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	// Check all users currently in combat
	trackedUsers := GetUsersInCombat()

	for _, userId := range trackedUsers {
		user := users.GetByUserId(userId)
		if user == nil {
			enemiesMutex.Lock()
			if _, exists := userEnemies[userId]; exists {
				delete(userEnemies, userId)
				mudlog.Warn("GMCPCombatEnemies", "action", "handleEnemiesNewRound", "issue", "user not found, cleaning up stale enemy tracking", "userId", userId)
			}
			enemiesMutex.Unlock()
			continue
		}

		currentEnemies := make(map[int]bool)

		room := rooms.LoadRoom(user.Character.RoomId)
		if room == nil {
			mudlog.Error("GMCPCombatEnemies", "action", "handleEnemiesNewRound", "error", "room lookup failed", "roomId", user.Character.RoomId, "userId", userId)
			continue
		}
		for _, mobId := range room.GetMobs() {
			if mob := mobs.GetInstance(mobId); mob != nil {
				// Check if mob is targeting this user
				if mob.Character.Aggro != nil && mob.Character.Aggro.UserId == userId {
					currentEnemies[mobId] = true
				}
			}
		}

		enemiesMutex.RLock()
		oldEnemies := userEnemies[userId]
		changed := false

		if len(oldEnemies) != len(currentEnemies) {
			changed = true
		} else {
			for mobId := range currentEnemies {
				if !oldEnemies[mobId] {
					changed = true
					break
				}
			}
		}
		enemiesMutex.RUnlock()

		if changed {
			enemiesMutex.Lock()
			if len(currentEnemies) > 0 {
				userEnemies[userId] = currentEnemies
			} else {
				delete(userEnemies, userId)
			}
			enemiesMutex.Unlock()
		}

		if changed {
			sendEnemiesUpdate(userId)
		}
	}

	return events.Continue
}

// handleEnemiesMobDeath removes dead mobs from enemy lists
func handleEnemiesMobDeath(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MobDeath)
	if !typeOk {
		mudlog.Error("GMCPCombatEnemies", "action", "handleEnemiesMobDeath", "error", "type assertion failed", "expectedType", "events.MobDeath", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	enemiesMutex.RLock()
	usersToCheck := make(map[int]bool)
	for userId, enemies := range userEnemies {
		if enemies[evt.InstanceId] {
			usersToCheck[userId] = true
		}
	}
	enemiesMutex.RUnlock()

	usersToUpdate := []int{}
	if len(usersToCheck) > 0 {
		enemiesMutex.Lock()
		for userId := range usersToCheck {
			if enemies, ok := userEnemies[userId]; ok {
				if enemies[evt.InstanceId] {
					delete(enemies, evt.InstanceId)
					if len(enemies) == 0 {
						delete(userEnemies, userId)
					}
					usersToUpdate = append(usersToUpdate, userId)
				}
			}
		}
		enemiesMutex.Unlock()
	}

	for _, userId := range usersToUpdate {
		sendEnemiesUpdate(userId)
	}

	return events.Continue
}

// handleEnemiesRoomChange handles when enemies move away
func handleEnemiesRoomChange(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		mudlog.Error("GMCPCombatEnemies", "action", "handleEnemiesRoomChange", "error", "type assertion failed", "expectedType", "events.RoomChange", "actualType", fmt.Sprintf("%T", e))
		return events.Continue
	}

	if evt.MobInstanceId == 0 {
		return events.Continue
	}

	// Check all users to see if this mob was in their enemy list
	enemiesMutex.RLock()
	usersToCheck := make(map[int]bool)
	for userId, enemies := range userEnemies {
		if enemies[evt.MobInstanceId] {
			usersToCheck[userId] = true
		}
	}
	enemiesMutex.RUnlock()

	// Now update the affected users
	usersToUpdate := []int{}
	if len(usersToCheck) > 0 {
		enemiesMutex.Lock()
		for userId := range usersToCheck {
			if enemies, ok := userEnemies[userId]; ok {
				if enemies[evt.MobInstanceId] {
					user := users.GetByUserId(userId)
					if user == nil {
						// Clean up stale enemy tracking
						delete(enemies, evt.MobInstanceId)
						if len(enemies) == 0 {
							delete(userEnemies, userId)
						}
						mudlog.Warn("GMCPCombatEnemies", "action", "handleEnemiesRoomChange", "issue", "user not found, cleaning up stale enemy tracking", "userId", userId)
						continue
					}
					if evt.ToRoomId != user.Character.RoomId {
						// Enemy moved to different room, remove from list
						delete(enemies, evt.MobInstanceId)
						if len(enemies) == 0 {
							delete(userEnemies, userId)
						}
						usersToUpdate = append(usersToUpdate, userId)
					}
				}
			}
		}
		enemiesMutex.Unlock()
	}

	for _, userId := range usersToUpdate {
		sendEnemiesUpdate(userId)
	}

	return events.Continue
}

// cleanupCombatEnemies removes all enemy tracking for a user
func cleanupCombatEnemies(userId int) {
	enemiesMutex.Lock()
	delete(userEnemies, userId)
	enemiesMutex.Unlock()
}

// sendEnemiesUpdate sends current enemy list for a user
func sendEnemiesUpdate(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		enemiesMutex.Lock()
		delete(userEnemies, userId)
		enemiesMutex.Unlock()
		mudlog.Warn("GMCPCombatEnemies", "action", "sendEnemiesUpdate", "issue", "user not found, cleaning up stale enemy tracking", "userId", userId)
		return
	}

	enemies := []EnemyInfo{}

	enemiesMutex.RLock()
	enemyMap := userEnemies[userId]
	enemiesMutex.RUnlock()

	for mobId := range enemyMap {
		if mob := mobs.GetInstance(mobId); mob != nil {
			isPrimary := false
			if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId == mobId {
				isPrimary = true
			}

			enemies = append(enemies, EnemyInfo{
				Name:      util.StripANSI(mob.Character.Name),
				Id:        mobId,
				IsPrimary: isPrimary,
			})
		}
	}

	handleCombatEnemiesUpdate(GMCPCombatEnemiesUpdate{
		UserId:  userId,
		Enemies: enemies,
	})
}
