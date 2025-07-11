package gmcp

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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

func init() {
	// Register listener for combat enemies updates
	events.RegisterListener(GMCPCombatEnemiesUpdate{}, handleCombatEnemiesUpdate)

	// Listen for events that affect enemy lists
	events.RegisterListener(events.NewRound{}, handleEnemiesNewRound)
	events.RegisterListener(events.MobDeath{}, handleEnemiesMobDeath)
	events.RegisterListener(events.RoomChange{}, handleEnemiesRoomChange)
	events.RegisterListener(events.PlayerDespawn{}, handleEnemiesPlayerDespawn)
}

func handleCombatEnemiesUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPCombatEnemiesUpdate)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Make sure they have GMCP enabled
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Continue
	}

	// Build the payload as an array directly
	enemies := make([]map[string]interface{}, len(evt.Enemies))
	for i, enemy := range evt.Enemies {
		enemies[i] = map[string]interface{}{
			"name":       enemy.Name,
			"id":         enemy.Id,
			"is_primary": enemy.IsPrimary,
		}
	}

	// Send the GMCP update with the array as the payload
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
		return events.Continue
	}

	// Check all online users
	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil {
			continue
		}

		// Get current enemies (mobs that have this user as a target)
		currentEnemies := make(map[int]bool)

		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			for _, mobId := range room.GetMobs() {
				if mob := mobs.GetInstance(mobId); mob != nil {
					// Check if mob is targeting this user
					if mob.Character.Aggro != nil && mob.Character.Aggro.UserId == userId {
						currentEnemies[mobId] = true
					}
				}
			}
		}

		// Check if enemies changed
		enemiesMutex.Lock()
		oldEnemies := userEnemies[userId]
		changed := false

		// Check if any enemies were added or removed
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

		if changed {
			if len(currentEnemies) > 0 {
				userEnemies[userId] = currentEnemies
			} else {
				delete(userEnemies, userId)
			}
		}
		enemiesMutex.Unlock()

		// Send update if changed
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
		return events.Continue
	}

	// Check all users to see if this mob was in their enemy list
	enemiesMutex.Lock()
	usersToUpdate := []int{}

	for userId, enemies := range userEnemies {
		if enemies[evt.InstanceId] {
			delete(enemies, evt.InstanceId)
			if len(enemies) == 0 {
				delete(userEnemies, userId)
			}
			usersToUpdate = append(usersToUpdate, userId)
		}
	}
	enemiesMutex.Unlock()

	// Send updates
	for _, userId := range usersToUpdate {
		sendEnemiesUpdate(userId)
	}

	return events.Continue
}

// handleEnemiesRoomChange handles when enemies move away
func handleEnemiesRoomChange(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		return events.Continue
	}

	// Only care about mob movements
	if evt.MobInstanceId == 0 {
		return events.Continue
	}

	// Check all users to see if this mob was in their enemy list
	enemiesMutex.Lock()
	usersToUpdate := []int{}

	for userId, enemies := range userEnemies {
		if enemies[evt.MobInstanceId] {
			user := users.GetByUserId(userId)
			if user != nil && evt.ToRoomId != user.Character.RoomId {
				// Enemy moved to different room, remove from list
				delete(enemies, evt.MobInstanceId)
				if len(enemies) == 0 {
					delete(userEnemies, userId)
				}
				usersToUpdate = append(usersToUpdate, userId)
			}
		}
	}
	enemiesMutex.Unlock()

	// Send updates
	for _, userId := range usersToUpdate {
		sendEnemiesUpdate(userId)
	}

	return events.Continue
}

// handleEnemiesPlayerDespawn cleans up when player leaves
func handleEnemiesPlayerDespawn(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		return events.Continue
	}

	enemiesMutex.Lock()
	delete(userEnemies, evt.UserId)
	enemiesMutex.Unlock()

	return events.Continue
}

// sendEnemiesUpdate sends current enemy list for a user
func sendEnemiesUpdate(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	enemies := []EnemyInfo{}

	enemiesMutex.RLock()
	enemyMap := userEnemies[userId]
	enemiesMutex.RUnlock()

	// Build enemy info list
	for mobId := range enemyMap {
		if mob := mobs.GetInstance(mobId); mob != nil {
			isPrimary := false
			if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId == mobId {
				isPrimary = true
			}

			enemies = append(enemies, EnemyInfo{
				Name:      mob.Character.Name,
				Id:        mobId,
				IsPrimary: isPrimary,
			})
		}
	}

	// Send update
	handleCombatEnemiesUpdate(GMCPCombatEnemiesUpdate{
		UserId:  userId,
		Enemies: enemies,
	})
}
