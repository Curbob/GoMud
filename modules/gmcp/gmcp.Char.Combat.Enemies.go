// Package gmcp handles Combat Enemies list updates for GMCP.
//
// Event-driven enemy tracking that shows all combat participants.
// Enemies are added when: player attacks them OR they attack the player.
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
	// enemiesMutexNew protects the enemies tracking map
	enemiesMutexNew sync.RWMutex

	// userEnemiesNew tracks all enemies for each user
	userEnemiesNew = make(map[int]map[int]*EnemyInfoFull) // userId -> map[enemyId]info
)

type EnemyInfoFull struct {
	Name      string
	Id        int
	Type      string // "mob" or "player"
	IsPrimary bool   // true if this is the user's target
	LastRound uint64 // Last round they were in combat
}

func init() {
	// Register the GMCP output handler
	events.RegisterListener(GMCPCombatEnemiesUpdate{}, handleCombatEnemiesUpdate)

	// Listen for combat events
	events.RegisterListener(events.CombatStarted{}, handleEnemiesCombatStarted)
	events.RegisterListener(events.DamageDealt{}, handleEnemiesDamageDealt)
	events.RegisterListener(events.AttackAvoided{}, handleEnemiesAttackAvoided)
	events.RegisterListener(events.MobDeath{}, handleEnemiesMobDeathNew)
	events.RegisterListener(events.PlayerDeath{}, handleEnemiesPlayerDeath)
	events.RegisterListener(events.RoomChange{}, handleEnemiesRoomChangeNew)
	events.RegisterListener(events.CombatEnded{}, handleEnemiesCombatEnded)
	events.RegisterListener(events.NewRound{}, handleEnemiesCleanup)
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

// handleEnemiesCombatStarted adds enemies when combat starts
func handleEnemiesCombatStarted(e events.Event) events.ListenerReturn {
	mudlog.Info("GMCPCombatEnemies", "event", "CombatStarted received in Enemies module")
	evt, ok := e.(events.CombatStarted)
	if !ok {
		mudlog.Error("GMCPCombatEnemies", "error", "CombatStarted type assertion failed")
		return events.Continue
	}

	mudlog.Info("GMCPCombatEnemies", "attackerType", evt.AttackerType, "attackerId", evt.AttackerId,
		"defenderType", evt.DefenderType, "defenderId", evt.DefenderId)

	// If player attacks mob/player, add defender to attacker's enemy list
	if evt.AttackerType == "player" {
		addEnemy(evt.AttackerId, evt.DefenderId, evt.DefenderType, evt.DefenderName, true)
	}

	// If mob/player attacks player, add attacker to defender's enemy list
	if evt.DefenderType == "player" {
		addEnemy(evt.DefenderId, evt.AttackerId, evt.AttackerType, evt.AttackerName, false)
	}

	return events.Continue
}

// handleEnemiesDamageDealt ensures enemies are tracked when damage occurs
func handleEnemiesDamageDealt(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.DamageDealt)
	if !ok {
		return events.Continue
	}

	// If damage is dealt to a player, ensure attacker is in their enemy list
	if evt.TargetType == "player" {
		addEnemy(evt.TargetId, evt.SourceId, evt.SourceType, evt.SourceName, false)
	}

	// If player deals damage, ensure target is in their enemy list
	if evt.SourceType == "player" {
		addEnemy(evt.SourceId, evt.TargetId, evt.TargetType, evt.TargetName, isUserTarget(evt.SourceId, evt.TargetId, evt.TargetType))
	}

	return events.Continue
}

// handleEnemiesAttackAvoided handles missed attacks (still combat participation)
func handleEnemiesAttackAvoided(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.AttackAvoided)
	if !ok {
		return events.Continue
	}

	// If attack was avoided by player, attacker is still an enemy
	if evt.DefenderType == "player" {
		addEnemy(evt.DefenderId, evt.AttackerId, evt.AttackerType, evt.AttackerName, false)
	}

	// If player's attack was avoided, target is still an enemy
	if evt.AttackerType == "player" {
		addEnemy(evt.AttackerId, evt.DefenderId, evt.DefenderType, evt.DefenderName, isUserTarget(evt.AttackerId, evt.DefenderId, evt.DefenderType))
	}

	return events.Continue
}

// handleEnemiesMobDeathNew removes dead mobs from enemy lists
func handleEnemiesMobDeathNew(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	enemiesMutexNew.Lock()
	usersToUpdate := []int{}
	for userId, enemies := range userEnemiesNew {
		if enemy, exists := enemies[evt.InstanceId]; exists && enemy.Type == "mob" {
			delete(enemies, evt.InstanceId)
			if len(enemies) == 0 {
				delete(userEnemiesNew, userId)
			}
			usersToUpdate = append(usersToUpdate, userId)
		}
	}
	enemiesMutexNew.Unlock()

	for _, userId := range usersToUpdate {
		sendEnemiesUpdateNew(userId)
	}

	return events.Continue
}

// handleEnemiesPlayerDeath removes dead players from enemy lists
func handleEnemiesPlayerDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDeath)
	if !ok {
		return events.Continue
	}

	// Also clear the dead player's enemy list
	enemiesMutexNew.Lock()
	delete(userEnemiesNew, evt.UserId)
	enemiesMutexNew.Unlock()

	// Remove from other players' enemy lists
	enemiesMutexNew.Lock()
	usersToUpdate := []int{}
	for userId, enemies := range userEnemiesNew {
		if enemy, exists := enemies[evt.UserId]; exists && enemy.Type == "player" {
			delete(enemies, evt.UserId)
			if len(enemies) == 0 {
				delete(userEnemiesNew, userId)
			}
			usersToUpdate = append(usersToUpdate, userId)
		}
	}
	enemiesMutexNew.Unlock()

	for _, userId := range usersToUpdate {
		sendEnemiesUpdateNew(userId)
	}

	return events.Continue
}

// handleEnemiesRoomChangeNew handles when enemies move away
func handleEnemiesRoomChangeNew(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok {
		return events.Continue
	}

	// Handle mob movement
	if evt.MobInstanceId != 0 {
		enemiesMutexNew.Lock()
		usersToUpdate := []int{}
		for userId, enemies := range userEnemiesNew {
			if enemy, exists := enemies[evt.MobInstanceId]; exists && enemy.Type == "mob" {
				user := users.GetByUserId(userId)
				if user != nil && user.Character.RoomId != evt.ToRoomId {
					delete(enemies, evt.MobInstanceId)
					if len(enemies) == 0 {
						delete(userEnemiesNew, userId)
					}
					usersToUpdate = append(usersToUpdate, userId)
				}
			}
		}
		enemiesMutexNew.Unlock()

		for _, userId := range usersToUpdate {
			sendEnemiesUpdateNew(userId)
		}
	}

	return events.Continue
}

// handleEnemiesCombatEnded clears enemy list when player's combat ends
func handleEnemiesCombatEnded(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CombatEnded)
	if !ok || evt.EntityType != "player" {
		return events.Continue
	}

	enemiesMutexNew.Lock()
	delete(userEnemiesNew, evt.EntityId)
	enemiesMutexNew.Unlock()

	// Send empty enemy list
	handleCombatEnemiesUpdate(GMCPCombatEnemiesUpdate{
		UserId:  evt.EntityId,
		Enemies: []EnemyInfo{},
	})

	return events.Continue
}

// handleEnemiesCleanup removes stale enemies who haven't acted in a while
func handleEnemiesCleanup(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok {
		return events.Continue
	}

	const staleRounds = uint64(10) // Remove enemies who haven't acted in 10 rounds
	currentRound := evt.RoundNumber

	enemiesMutexNew.Lock()
	usersToUpdate := []int{}
	for userId, enemies := range userEnemiesNew {
		changed := false
		for enemyId, enemy := range enemies {
			if currentRound-enemy.LastRound > staleRounds {
				delete(enemies, enemyId)
				changed = true
			}
		}
		if changed {
			if len(enemies) == 0 {
				delete(userEnemiesNew, userId)
			}
			usersToUpdate = append(usersToUpdate, userId)
		}
	}
	enemiesMutexNew.Unlock()

	for _, userId := range usersToUpdate {
		sendEnemiesUpdateNew(userId)
	}

	return events.Continue
}

// addEnemy adds an enemy to a user's enemy list
func addEnemy(userId int, enemyId int, enemyType string, enemyName string, isPrimary bool) {
	// Validate user has GMCP
	_, valid := validateUserForGMCP(userId, "GMCPCombatEnemies")
	if !valid {
		return
	}

	enemiesMutexNew.Lock()
	if userEnemiesNew[userId] == nil {
		userEnemiesNew[userId] = make(map[int]*EnemyInfoFull)
	}

	// Check if already exists
	if existing, exists := userEnemiesNew[userId][enemyId]; exists {
		existing.LastRound = util.GetRoundCount()
		if isPrimary {
			existing.IsPrimary = true
		}
	} else {
		userEnemiesNew[userId][enemyId] = &EnemyInfoFull{
			Name:      util.StripANSI(enemyName),
			Id:        enemyId,
			Type:      enemyType,
			IsPrimary: isPrimary,
			LastRound: util.GetRoundCount(),
		}
	}
	enemiesMutexNew.Unlock()

	mudlog.Info("GMCPCombatEnemies", "action", "Enemy added", "userId", userId,
		"enemyName", enemyName, "enemyType", enemyType, "isPrimary", isPrimary)

	// Send update
	sendEnemiesUpdateNew(userId)
}

// isUserTarget checks if an enemy is the user's current target
func isUserTarget(userId int, enemyId int, enemyType string) bool {
	targetMutexNew.RLock()
	target := userTargetsNew[userId]
	targetMutexNew.RUnlock()

	if target == nil {
		return false
	}

	return target.Id == enemyId && target.Type == enemyType
}

// sendEnemiesUpdateNew sends current enemy list for a user
func sendEnemiesUpdateNew(userId int) {
	enemiesMutexNew.RLock()
	enemyMap := userEnemiesNew[userId]
	enemiesMutexNew.RUnlock()

	enemies := []EnemyInfo{}

	// Get user's current room to verify enemies are still present
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return
	}

	for _, enemy := range enemyMap {
		// Verify enemy still exists and is in same room
		stillExists := false

		if enemy.Type == "mob" {
			for _, mobId := range room.GetMobs() {
				if mobId == enemy.Id {
					if mob := mobs.GetInstance(mobId); mob != nil {
						stillExists = true
						enemies = append(enemies, EnemyInfo{
							Name:      enemy.Name,
							Id:        enemy.Id,
							IsPrimary: enemy.IsPrimary,
						})
					}
					break
				}
			}
		} else if enemy.Type == "player" {
			for _, playerId := range room.GetPlayers() {
				if playerId == enemy.Id {
					stillExists = true
					enemies = append(enemies, EnemyInfo{
						Name:      enemy.Name,
						Id:        enemy.Id,
						IsPrimary: enemy.IsPrimary,
					})
					break
				}
			}
		}

		// Clean up if enemy no longer exists
		if !stillExists {
			enemiesMutexNew.Lock()
			delete(enemyMap, enemy.Id)
			enemiesMutexNew.Unlock()
		}
	}

	handleCombatEnemiesUpdate(GMCPCombatEnemiesUpdate{
		UserId:  userId,
		Enemies: enemies,
	})
}

// cleanupCombatEnemiesNew removes all enemy tracking for a user
func cleanupCombatEnemiesNew(userId int) {
	enemiesMutexNew.Lock()
	delete(userEnemiesNew, userId)
	enemiesMutexNew.Unlock()
}
