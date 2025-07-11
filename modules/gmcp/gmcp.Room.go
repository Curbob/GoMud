package gmcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	//
	// We can use all functions only, but this demonstrates
	// how to use a struct
	//
	g := GMCPRoomModule{
		plug: plugins.New(`gmcp.Room`, `1.0`),
	}

	// Temporary for testing purposes.
	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(events.PlayerDespawn{}, g.despawnHandler)
	events.RegisterListener(GMCPRoomUpdate{}, g.buildAndSendGMCPPayload)
	events.RegisterListener(events.ItemOwnership{}, g.itemOwnershipHandler)

}

type GMCPRoomModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

// Tell the system a wish to send specific GMCP Update data
type GMCPRoomUpdate struct {
	UserId     int
	Identifier string
}

func (g GMCPRoomUpdate) Type() string { return `GMCPRoomUpdate` }

func (g *GMCPRoomModule) despawnHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerDespawn", "Actual Type", e.Type())
		return events.Cancel
	}

	// If this isn't a user changing rooms, just pass it along.
	if evt.UserId == 0 {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

	//
	// Send GMCP Updates for players leaving
	//
	for _, uid := range room.GetPlayers() {

		if uid == evt.UserId {
			continue
		}

		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}

		events.AddToQueue(GMCPOut{
			UserId:  uid,
			Module:  `Room.Remove.Player`,
			Payload: map[string]string{"name": evt.CharacterName},
		})

	}

	return events.Continue
}

func (g *GMCPRoomModule) roomChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "RoomChange", "Actual Type", e.Type())
		return events.Cancel
	}

	// Send updates to players in old/new rooms for
	// players or npcs (whichever changed)

	if evt.FromRoomId != 0 {
		if oldRoom := rooms.LoadRoom(evt.FromRoomId); oldRoom != nil {
			for _, uId := range oldRoom.GetPlayers() {
				if uId == evt.UserId {
					continue
				}

				// Send Room.Remove message for the departure
				if evt.MobInstanceId > 0 {
					// NPC left the room
					if mob := mobs.GetInstance(evt.MobInstanceId); mob != nil {
						events.AddToQueue(GMCPOut{
							UserId: uId,
							Module: `Room.Remove.Npc`,
							Payload: map[string]interface{}{
								"id":   mob.ShorthandId(),
								"name": mob.Character.Name,
							},
						})
					}
				} else {
					// Player left the room - already handled by despawnHandler
				}
			}
		}
	}

	if evt.ToRoomId != 0 {
		if newRoom := rooms.LoadRoom(evt.ToRoomId); newRoom != nil {
			for _, uId := range newRoom.GetPlayers() {
				if uId == evt.UserId {
					continue
				}

				// Send Room.Add message for the new arrival
				if evt.MobInstanceId > 0 {
					// NPC entered the room
					if mob := mobs.GetInstance(evt.MobInstanceId); mob != nil {
						// Determine threat level for this viewer
						threatLevel := "peaceful"
						targetingYou := false

						viewer := users.GetByUserId(uId)
						if viewer != nil {
							if mob.Character.Aggro != nil {
								if mob.Character.Aggro.UserId == uId {
									threatLevel = "fighting"
									targetingYou = true
								} else {
									threatLevel = "aggressive"
								}
							} else if mob.Hostile ||
								(len(mob.Groups) > 0 && mobs.IsHostile(mob.Groups[0], uId)) ||
								mob.HatesRace(viewer.Character.Race()) ||
								mob.HatesAlignment(viewer.Character.Alignment) {
								threatLevel = "hostile"
							}
						}

						events.AddToQueue(GMCPOut{
							UserId: uId,
							Module: `Room.Add.Npc`,
							Payload: map[string]interface{}{
								"id":            mob.ShorthandId(),
								"name":          mob.Character.Name,
								"threat_level":  threatLevel,
								"targeting_you": targetingYou,
							},
						})
					}
				} else {
					// Player entered the room
					if user := users.GetByUserId(evt.UserId); user != nil {
						events.AddToQueue(GMCPOut{
							UserId: uId,
							Module: `Room.Add.Player`,
							Payload: map[string]string{
								"name": user.Character.Name,
							},
						})
					}
				}
			}
		}
	}

	// If it's a mob changing rooms, don't need to send it its own room info
	if evt.UserId == 0 {
		return events.Continue
	}

	// Send update to the moved player about their new room.
	events.AddToQueue(GMCPRoomUpdate{
		UserId:     evt.UserId,
		Identifier: `Room.Info`,
	})

	return events.Continue
}

func (g *GMCPRoomModule) buildAndSendGMCPPayload(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPRoomUpdate)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPCharUpdate", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Make sure they have this gmcp module enabled.
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Cancel
	}

	if len(evt.Identifier) >= 4 {
		// Normalize the identifier (handle case variations)
		identifierParts := strings.Split(strings.ToLower(evt.Identifier), `.`)
		for i := 0; i < len(identifierParts); i++ {
			identifierParts[i] = strings.Title(identifierParts[i])
		}

		requestedId := strings.Join(identifierParts, `.`)

		payload, moduleName := g.GetRoomNode(user, requestedId)

		if payload != nil && moduleName != "" {
			events.AddToQueue(GMCPOut{
				UserId:  evt.UserId,
				Module:  moduleName,
				Payload: payload,
			})
		}
	}

	return events.Continue
}

// sendAllRoomNodes sends all Room nodes as individual GMCP messages
func (g *GMCPRoomModule) sendAllRoomNodes(user *users.UserRecord) {
	// Send each node individually
	// Note: Room.Info.Basic contains the static room information (id, name, area, etc)
	// Room.Info.Exits is separated since exits can change (locks, secrets discovered)
	// Room.Info.Contents.* are the most dynamic and change frequently
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Basic`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Exits`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Players`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Npcs`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Items`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Containers`})
}

func (g *GMCPRoomModule) GetRoomNode(user *users.UserRecord, gmcpModule string) (data any, moduleName string) {

	// Handle both "Room" and "Room.Info" as requests for all room data
	all := gmcpModule == `Room` || gmcpModule == `Room.Info`

	// If requesting all, we'll handle it differently by queuing individual messages
	if all {
		g.sendAllRoomNodes(user)
		return nil, ""
	}

	// Get the new room data... abort if doesn't exist.
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return nil, ""
	}

	payload := GMCPRoomModule_Payload{}

	////////////////////////////////////////////////
	// Room.Contents
	// Note: Process this first since we might be
	//       sending a subset of data
	////////////////////////////////////////////////

	////////////////////////////////////////////////
	// Room.Contents.Containers
	////////////////////////////////////////////////
	if g.wantsGMCPPayload(`Room.Info.Contents.Containers`, gmcpModule) {

		payload.Contents.Containers = []GMCPRoomModule_Payload_Contents_Container{}
		for name, container := range room.Containers {

			c := GMCPRoomModule_Payload_Contents_Container{
				Name:   name,
				Usable: len(container.Recipes) > 0,
			}

			if container.HasLock() {
				c.Locked = true
				lockId := fmt.Sprintf(`%d-%s`, room.RoomId, name)
				c.HasKey, c.HasPickCombo = user.Character.HasKey(lockId, int(container.Lock.Difficulty))
			}

			payload.Contents.Containers = append(payload.Contents.Containers, c)
		}

		if `Room.Info.Contents.Containers` == gmcpModule {
			return payload.Contents.Containers, `Room.Info.Contents.Containers`
		}
	}

	////////////////////////////////////////////////
	// Room.Contents.Items
	////////////////////////////////////////////////
	if g.wantsGMCPPayload(`Room.Info.Contents.Items`, gmcpModule) {
		payload.Contents.Items = []GMCPRoomModule_Payload_Contents_Item{}
		for _, itm := range room.Items {
			payload.Contents.Items = append(payload.Contents.Items, GMCPRoomModule_Payload_Contents_Item{
				Id:        itm.ShorthandId(),
				Name:      itm.Name(),
				QuestFlag: itm.GetSpec().QuestToken != ``,
			})
		}

		if `Room.Info.Contents.Items` == gmcpModule {
			return payload.Contents.Items, `Room.Info.Contents.Items`
		}
	}

	////////////////////////////////////////////////
	// Room.Contents.Players
	////////////////////////////////////////////////
	if g.wantsGMCPPayload(`Room.Info.Contents.Players`, gmcpModule) {
		payload.Contents.Players = []GMCPRoomModule_Payload_Contents_Character{}
		for _, uId := range room.GetPlayers() {

			// Exclude viewing player
			if uId == user.UserId {
				continue
			}

			u := users.GetByUserId(uId)
			if u == nil {
				continue
			}

			if u.Character.HasBuffFlag(buffs.Hidden) {
				continue
			}

			// Players are always shown as peaceful to other players
			payload.Contents.Players = append(payload.Contents.Players, GMCPRoomModule_Payload_Contents_Character{
				Id:           u.ShorthandId(),
				Name:         u.Character.Name,
				Adjectives:   u.Character.GetAdjectives(),
				ThreatLevel:  "peaceful",
				TargetingYou: false,
			})
		}

		if `Room.Info.Contents.Players` == gmcpModule {
			return payload.Contents.Players, `Room.Info.Contents.Players`
		}
	}

	////////////////////////////////////////////////
	// Room.Contents.Npcs
	////////////////////////////////////////////////
	if g.wantsGMCPPayload(`Room.Info.Contents.Npcs`, gmcpModule) {
		payload.Contents.Npcs = []GMCPRoomModule_Payload_Contents_Character{}
		for _, mIId := range room.GetMobs() {
			mob := mobs.GetInstance(mIId)
			if mob == nil {
				continue
			}

			if mob.Character.HasBuffFlag(buffs.Hidden) {
				continue
			}

			// Determine threat level and targeting status
			threatLevel := "peaceful"
			targetingYou := false

			// Check if mob is targeting this user
			if mob.Character.Aggro != nil {
				if mob.Character.Aggro.UserId == user.UserId {
					threatLevel = "fighting"
					targetingYou = true
				} else {
					threatLevel = "aggressive"
				}
			} else if mob.Hostile ||
				(len(mob.Groups) > 0 && mobs.IsHostile(mob.Groups[0], user.UserId)) ||
				mob.HatesRace(user.Character.Race()) ||
				mob.HatesAlignment(user.Character.Alignment) {
				threatLevel = "hostile"
			}

			c := GMCPRoomModule_Payload_Contents_Character{
				Id:           mob.ShorthandId(),
				Name:         mob.Character.Name,
				Adjectives:   mob.Character.GetAdjectives(),
				ThreatLevel:  threatLevel,
				TargetingYou: targetingYou,
			}

			if len(mob.QuestFlags) > 0 {
				for _, qFlag := range mob.QuestFlags {
					if user.Character.HasQuest(qFlag) || (len(qFlag) >= 5 && qFlag[len(qFlag)-5:] == `start`) {
						c.QuestFlag = true
						break
					}
				}
			}

			payload.Contents.Npcs = append(payload.Contents.Npcs, c)
		}

		if `Room.Info.Contents.Npcs` == gmcpModule {
			return payload.Contents.Npcs, `Room.Info.Contents.Npcs`
		}

	}

	if !all && `Room.Info.Contents` == gmcpModule {
		return payload.Contents, `Room.Info.Contents`
	}

	////////////////////////////////////////////////
	// Room.Info
	// Note: This populates the root Room.Info data
	////////////////////////////////////////////////
	if g.wantsGMCPPayload(`Room.Info.Basic`, gmcpModule) || g.wantsGMCPPayload(`Room.Info.Exits`, gmcpModule) {

		// Basic details
		payload.Id = room.RoomId
		payload.Name = room.Title
		payload.Area = room.Zone
		payload.Environment = room.GetBiome().Name
		payload.Details = []string{}

		// Coordinates
		payload.Coordinates = room.Zone
		m := mapper.GetMapper(room.RoomId)
		x, y, z, err := m.GetCoordinates(room.RoomId)
		if err != nil {
			payload.Coordinates += `, 999999999999999999, 999999999999999999, 999999999999999999`
		} else {
			payload.Coordinates += `, ` + strconv.Itoa(x) + `, ` + strconv.Itoa(y) + `, ` + strconv.Itoa(z)
		}

		// set exits
		payload.Exits = map[string]GMCPRoomModule_Payload_Contents_ExitInfo{}

		for exitName, exitInfo := range room.Exits {

			if exitInfo.Secret {
				if exitRoom := rooms.LoadRoom(exitInfo.RoomId); exitRoom != nil {
					if !exitRoom.HasVisited(user.UserId, rooms.VisitorUser) {
						continue
					}
				}
			}

			// Skip non compass directions?
			if !mapper.IsCompassDirection(exitName) {
				//continue
			}

			// Build the exit info
			deltaX, deltaY, deltaZ := 0, 0, 0
			if len(exitInfo.MapDirection) > 0 {
				deltaX, deltaY, deltaZ = mapper.GetDelta(exitInfo.MapDirection)
			} else {
				deltaX, deltaY, deltaZ = mapper.GetDelta(exitName)
			}

			exitV2 := GMCPRoomModule_Payload_Contents_ExitInfo{
				RoomId:  exitInfo.RoomId,
				DeltaX:  deltaX,
				DeltaY:  deltaY,
				DeltaZ:  deltaZ,
				Status:  "open",
				Details: []string{},
			}

			if exitInfo.Secret {
				exitV2.Details = append(exitV2.Details, `secret`)
			}

			if exitInfo.HasLock() {
				// Check if the lock is currently locked
				if exitInfo.Lock.IsLocked() {
					exitV2.Status = "locked"
					exitV2.Details = append(exitV2.Details, `locked`)
				}

				lockId := fmt.Sprintf(`%d-%s`, room.RoomId, exitName)
				haskey, hascombo := user.Character.HasKey(lockId, int(exitInfo.Lock.Difficulty))

				if haskey {
					exitV2.Details = append(exitV2.Details, `player_has_key`)
				}

				if hascombo {
					exitV2.Details = append(exitV2.Details, `player_has_pick_combo`)
				}
			}

			payload.Exits[exitName] = exitV2
		}
		// end exits

		// Set room details
		if len(room.SkillTraining) > 0 {
			payload.Details = append(payload.Details, `trainer`)
		}
		if room.IsBank {
			payload.Details = append(payload.Details, `bank`)
		}
		if room.IsStorage {
			payload.Details = append(payload.Details, `storage`)
		}
		if room.IsCharacterRoom {
			payload.Details = append(payload.Details, `character`)
		}

		// Indicate if this is an ephemeral room
		if rooms.IsEphemeralRoomId(room.RoomId) {
			payload.Details = append(payload.Details, `ephemeral`)
		}
		// end room details

		// Return specific nodes if requested
		if gmcpModule == `Room.Info.Basic` {
			// Basic room info without exits (static data that rarely changes)
			basicPayload := map[string]interface{}{
				"id":          payload.Id,
				"name":        payload.Name,
				"area":        payload.Area,
				"environment": payload.Environment,
				"coordinates": payload.Coordinates,
				"details":     payload.Details,
			}
			return basicPayload, `Room.Info.Basic`
		}

		if gmcpModule == `Room.Info.Exits` {
			// Exits can change when locks are opened or secrets discovered
			exitsPayload := map[string]interface{}{
				"exits": payload.Exits,
			}
			return exitsPayload, `Room.Info.Exits`
		}
	}

	// Handle Room.Wrongdir request
	if gmcpModule == `Room.Wrongdir` {
		// Return empty structure to match what we send during gameplay
		return map[string]string{"dir": ""}, `Room.Wrongdir`
	}

	// Handle Room.Add requests
	if gmcpModule == `Room.Add.Player` {
		return map[string]string{"name": ""}, `Room.Add.Player`
	}
	if gmcpModule == `Room.Add.Npc` {
		return map[string]interface{}{
			"id":            "",
			"name":          "",
			"threat_level":  "",
			"targeting_you": false,
		}, `Room.Add.Npc`
	}
	if gmcpModule == `Room.Add.Item` {
		return map[string]interface{}{"id": "", "name": "", "quest_flag": false}, `Room.Add.Item`
	}

	// Handle Room.Remove requests
	if gmcpModule == `Room.Remove.Player` {
		return map[string]string{"name": ""}, `Room.Remove.Player`
	}
	if gmcpModule == `Room.Remove.Npc` {
		return map[string]interface{}{"id": "", "name": ""}, `Room.Remove.Npc`
	}
	if gmcpModule == `Room.Remove.Item` {
		return map[string]interface{}{"id": "", "name": ""}, `Room.Remove.Item`
	}

	// If we reached this point, we have a problem.
	mudlog.Error(`gmcp.Room`, `error`, `Bad module requested`, `module`, gmcpModule)
	return nil, ""
}

// wantsGMCPPayload(`Room.Info.Contents`, `Room.Info`)
func (g *GMCPRoomModule) wantsGMCPPayload(packageToConsider string, packageRequested string) bool {

	if packageToConsider == packageRequested {
		return true
	}

	if len(packageToConsider) < len(packageRequested) {
		return false
	}

	if packageToConsider[0:len(packageRequested)] == packageRequested {
		return true
	}

	return false
}

type GMCPRoomModule_Payload struct {
	Id          int                                                 `json:"id"`
	Name        string                                              `json:"name"`
	Area        string                                              `json:"area"`
	Environment string                                              `json:"environment"`
	Coordinates string                                              `json:"coordinates"`
	Exits       map[string]GMCPRoomModule_Payload_Contents_ExitInfo `json:"exits"`
	Details     []string                                            `json:"details"`
	Contents    GMCPRoomModule_Payload_Contents                     `json:"contents"`
}

type GMCPRoomModule_Payload_Contents_ExitInfo struct {
	RoomId  int      `json:"room_id"`
	DeltaX  int      `json:"delta_x"`
	DeltaY  int      `json:"delta_y"`
	DeltaZ  int      `json:"delta_z"`
	Status  string   `json:"status"` // "open", "locked", "blocked"
	Details []string `json:"details"`
}

// /////////////////
// Room.Contents
// /////////////////
type GMCPRoomModule_Payload_Contents struct {
	Players    []GMCPRoomModule_Payload_Contents_Character `json:"players"`
	Npcs       []GMCPRoomModule_Payload_Contents_Character `json:"npcs"`
	Items      []GMCPRoomModule_Payload_Contents_Item      `json:"items"`
	Containers []GMCPRoomModule_Payload_Contents_Container `json:"containers"`
}

type GMCPRoomModule_Payload_Contents_Character struct {
	Id           string   `json:"id"`
	Name         string   `json:"name"`
	Adjectives   []string `json:"adjectives"`
	QuestFlag    bool     `json:"quest_flag"`
	ThreatLevel  string   `json:"threat_level"`  // "peaceful", "hostile", "aggressive", "fighting"
	TargetingYou bool     `json:"targeting_you"` // true if this mob has you as aggro target
}

type GMCPRoomModule_Payload_Contents_Item struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	QuestFlag bool   `json:"quest_flag"`
}

type GMCPRoomModule_Payload_Contents_Container struct {
	Name         string `json:"name"`
	Locked       bool   `json:"locked"`
	HasKey       bool   `json:"has_key"`
	HasPickCombo bool   `json:"has_pick_combo"`
	Usable       bool   `json:"usable"`
}

func (g *GMCPRoomModule) itemOwnershipHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.ItemOwnership)
	if !typeOk {
		return events.Continue
	}

	// We need to determine if this is a room-related item change
	// When a user drops an item: UserId > 0 and Gained = false
	// When a user picks up an item: UserId > 0 and Gained = true

	if evt.UserId > 0 {
		user := users.GetByUserId(evt.UserId)
		if user == nil {
			return events.Continue
		}

		room := rooms.LoadRoom(user.Character.RoomId)
		if room == nil {
			return events.Continue
		}

		// Send updates to all players in the room
		for _, uid := range room.GetPlayers() {
			if !evt.Gained {
				// User dropped item (item added to room)
				events.AddToQueue(GMCPOut{
					UserId: uid,
					Module: `Room.Add.Item`,
					Payload: map[string]interface{}{
						"id":         evt.Item.ShorthandId(),
						"name":       evt.Item.Name(),
						"quest_flag": evt.Item.GetSpec().QuestToken != ``,
					},
				})
			} else {
				// User picked up item (item removed from room)
				events.AddToQueue(GMCPOut{
					UserId: uid,
					Module: `Room.Remove.Item`,
					Payload: map[string]interface{}{
						"id":   evt.Item.ShorthandId(),
						"name": evt.Item.Name(),
					},
				})
			}
		}
	}

	return events.Continue
}
