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
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	g := GMCPRoomModule{
		plug: plugins.New(`gmcp.Room`, `1.0`),
	}

	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(events.PlayerDespawn{}, g.despawnHandler)
	events.RegisterListener(GMCPRoomUpdate{}, g.buildAndSendGMCPPayload)
	events.RegisterListener(events.ItemOwnership{}, g.itemOwnershipHandler)
	events.RegisterListener(events.ExitLockChanged{}, g.exitLockChangedHandler)
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

	if evt.UserId == 0 {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

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

	if evt.FromRoomId != 0 {
		if oldRoom := rooms.LoadRoom(evt.FromRoomId); oldRoom != nil {
			for _, uId := range oldRoom.GetPlayers() {
				if uId == evt.UserId {
					continue
				}

				if evt.MobInstanceId > 0 {
					if mob := mobs.GetInstance(evt.MobInstanceId); mob != nil {
						events.AddToQueue(GMCPOut{
							UserId: uId,
							Module: `Room.Remove.Npc`,
							Payload: map[string]interface{}{
								"id":   mob.ShorthandId(),
								"name": util.StripANSI(mob.Character.Name),
							},
						})
					}
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

				if evt.MobInstanceId > 0 {
					if mob := mobs.GetInstance(evt.MobInstanceId); mob != nil {
						threatLevel := "peaceful"
						targeting := []string{}

						viewer := users.GetByUserId(uId)
						if viewer != nil {
							if mob.Character.Aggro != nil {
								if mob.Character.Aggro.UserId == uId {
									threatLevel = "fighting"
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

						// Build list of players this mob is targeting
						if len(mob.Character.PlayerDamage) > 0 {
							for userId := range mob.Character.PlayerDamage {
								if targetUser := users.GetByUserId(userId); targetUser != nil {
									targeting = append(targeting, util.StripANSI(targetUser.Character.Name))
								}
							}
						}

						events.AddToQueue(GMCPOut{
							UserId: uId,
							Module: `Room.Add.Npc`,
							Payload: map[string]interface{}{
								"id":           mob.ShorthandId(),
								"name":         util.StripANSI(mob.Character.Name),
								"threat_level": threatLevel,
								"targeting":    targeting,
							},
						})
					}
				} else {
					if user := users.GetByUserId(evt.UserId); user != nil {
						events.AddToQueue(GMCPOut{
							UserId: uId,
							Module: `Room.Add.Player`,
							Payload: map[string]string{
								"name": util.StripANSI(user.Character.Name),
							},
						})
					}
				}
			}
		}
	}

	if evt.UserId == 0 {
		return events.Continue
	}

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

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Cancel
	}

	if len(evt.Identifier) >= 4 {
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

func (g *GMCPRoomModule) sendAllRoomNodes(user *users.UserRecord) {
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Basic`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Exits`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Players`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Npcs`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Items`})
	events.AddToQueue(GMCPRoomUpdate{UserId: user.UserId, Identifier: `Room.Info.Contents.Containers`})
}

func (g *GMCPRoomModule) GetRoomNode(user *users.UserRecord, gmcpModule string) (data any, moduleName string) {

	all := gmcpModule == `Room` || gmcpModule == `Room.Info`

	if all {
		g.sendAllRoomNodes(user)
		return nil, ""
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return nil, ""
	}

	payload := GMCPRoomModule_Payload{}

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

	if g.wantsGMCPPayload(`Room.Info.Contents.Items`, gmcpModule) {
		payload.Contents.Items = []GMCPRoomModule_Payload_Contents_Item{}
		for _, itm := range room.Items {
			payload.Contents.Items = append(payload.Contents.Items, GMCPRoomModule_Payload_Contents_Item{
				Id:        itm.ShorthandId(),
				Name:      util.StripANSI(itm.Name()),
				QuestFlag: itm.GetSpec().QuestToken != ``,
			})
		}

		if `Room.Info.Contents.Items` == gmcpModule {
			return payload.Contents.Items, `Room.Info.Contents.Items`
		}
	}

	if g.wantsGMCPPayload(`Room.Info.Contents.Players`, gmcpModule) {
		payload.Contents.Players = []GMCPRoomModule_Payload_Contents_Character{}
		for _, uId := range room.GetPlayers() {

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

			payload.Contents.Players = append(payload.Contents.Players, GMCPRoomModule_Payload_Contents_Character{
				Id:          u.ShorthandId(),
				Name:        util.StripANSI(u.Character.Name),
				Adjectives:  u.Character.GetAdjectives(),
				ThreatLevel: "peaceful",
				Targeting:   []string{}, // Players don't target other players in this context
			})
		}

		if `Room.Info.Contents.Players` == gmcpModule {
			return payload.Contents.Players, `Room.Info.Contents.Players`
		}
	}

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

			threatLevel := "peaceful"
			targeting := []string{}

			// Check mob's current aggro target
			if mob.Character.Aggro != nil {
				if mob.Character.Aggro.UserId == user.UserId {
					threatLevel = "fighting"
				} else {
					threatLevel = "aggressive"
				}
			} else if mob.Hostile ||
				(len(mob.Groups) > 0 && mobs.IsHostile(mob.Groups[0], user.UserId)) ||
				mob.HatesRace(user.Character.Race()) ||
				mob.HatesAlignment(user.Character.Alignment) {
				threatLevel = "hostile"
			}

			// Build list of players this mob is targeting (based on damage tracking)
			if len(mob.Character.PlayerDamage) > 0 {
				for userId := range mob.Character.PlayerDamage {
					if targetUser := users.GetByUserId(userId); targetUser != nil {
						targeting = append(targeting, util.StripANSI(targetUser.Character.Name))
					}
				}
			}

			c := GMCPRoomModule_Payload_Contents_Character{
				Id:          mob.ShorthandId(),
				Name:        util.StripANSI(mob.Character.Name),
				Adjectives:  mob.Character.GetAdjectives(),
				ThreatLevel: threatLevel,
				Targeting:   targeting,
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

	if g.wantsGMCPPayload(`Room.Info.Basic`, gmcpModule) || g.wantsGMCPPayload(`Room.Info.Exits`, gmcpModule) {

		payload.Id = room.RoomId
		payload.Name = util.StripANSI(room.Title)
		payload.Area = room.Zone
		payload.Environment = room.GetBiome().Name
		payload.Details = []string{}

		payload.Coordinates = room.Zone
		m := mapper.GetMapper(room.RoomId)

		// Generate map_id based on the lowest room ID in the connected area
		// This provides a stable identifier regardless of which room triggered map creation
		if m != nil {
			payload.MapId = fmt.Sprintf("map-%d", m.GetLowestRoomId())
		}

		x, y, z, err := m.GetCoordinates(room.RoomId)
		if err != nil {
			payload.Coordinates += `, 999999999999999999, 999999999999999999, 999999999999999999`
		} else {
			payload.Coordinates += `, ` + strconv.Itoa(x) + `, ` + strconv.Itoa(y) + `, ` + strconv.Itoa(z)
		}

		payload.Exits = map[string]GMCPRoomModule_Payload_Contents_ExitInfo{}

		for exitName, exitInfo := range room.Exits {

			if exitInfo.Secret {
				if exitRoom := rooms.LoadRoom(exitInfo.RoomId); exitRoom != nil {
					if !exitRoom.HasVisited(user.UserId, rooms.VisitorUser) {
						continue
					}
				}
			}

			// Include all exits, not just compass directions
			// Custom exits are important for gameplay

			deltaX, deltaY, deltaZ := 0, 0, 0
			if len(exitInfo.MapDirection) > 0 {
				deltaX, deltaY, deltaZ = mapper.GetDelta(exitInfo.MapDirection)
			} else {
				deltaX, deltaY, deltaZ = mapper.GetDelta(exitName)
			}

			exitV2 := GMCPRoomModule_Payload_Contents_ExitInfo{
				RoomId: exitInfo.RoomId,
				DeltaX: deltaX,
				DeltaY: deltaY,
				DeltaZ: deltaZ,
			}

			// Check if this exit leads to a different map area
			if targetMapper := mapper.GetMapperIfExists(exitInfo.RoomId); targetMapper != nil {
				targetMapId := fmt.Sprintf("map-%d", targetMapper.GetLowestRoomId())
				if targetMapId != payload.MapId {
					if exitV2.Details == nil {
						exitV2.Details = make(map[string]interface{})
					}
					exitV2.Details["leads_to_map"] = targetMapId

					// Also include the area name of the destination
					if targetRoom := rooms.LoadRoom(exitInfo.RoomId); targetRoom != nil {
						exitV2.Details["leads_to_area"] = targetRoom.Zone
					}
				}
			}

			if exitInfo.HasLock() {
				if exitV2.Details == nil {
					exitV2.Details = make(map[string]interface{})
				}
				exitV2.Details["type"] = "door"
				exitV2.Details["name"] = exitName

				if exitInfo.Lock.IsLocked() {
					exitV2.Details["state"] = "locked"
				} else {
					exitV2.Details["state"] = "open"
				}

				lockId := fmt.Sprintf(`%d-%s`, room.RoomId, exitName)
				haskey, hascombo := user.Character.HasKey(lockId, int(exitInfo.Lock.Difficulty))

				if haskey {
					exitV2.Details["hasKey"] = true
				}

				if hascombo {
					exitV2.Details["hasPicked"] = true
				}
			}

			payload.Exits[exitName] = exitV2
		}

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

		if rooms.IsEphemeralRoomId(room.RoomId) {
			payload.Details = append(payload.Details, `ephemeral`)
		}

		if gmcpModule == `Room.Info.Basic` {
			basicPayload := map[string]interface{}{
				"id":          payload.Id,
				"name":        payload.Name,
				"area":        payload.Area,
				"map_id":      payload.MapId,
				"environment": payload.Environment,
				"coordinates": payload.Coordinates,
				"details":     payload.Details,
			}
			return basicPayload, `Room.Info.Basic`
		}

		if gmcpModule == `Room.Info.Exits` {
			return payload.Exits, `Room.Info.Exits`
		}
	}

	if gmcpModule == `Room.Wrongdir` {
		return map[string]string{"dir": ""}, `Room.Wrongdir`
	}

	if gmcpModule == `Room.Add.Player` {
		return map[string]string{"name": ""}, `Room.Add.Player`
	}
	if gmcpModule == `Room.Add.Npc` {
		return map[string]interface{}{
			"id":           "",
			"name":         "",
			"threat_level": "",
			"targeting":    []string{},
		}, `Room.Add.Npc`
	}
	if gmcpModule == `Room.Add.Item` {
		return map[string]interface{}{"id": "", "name": "", "quest_flag": false}, `Room.Add.Item`
	}

	if gmcpModule == `Room.Remove.Player` {
		return map[string]string{"name": ""}, `Room.Remove.Player`
	}
	if gmcpModule == `Room.Remove.Npc` {
		return map[string]interface{}{"id": "", "name": ""}, `Room.Remove.Npc`
	}
	if gmcpModule == `Room.Remove.Item` {
		return map[string]interface{}{"id": "", "name": ""}, `Room.Remove.Item`
	}

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
	MapId       string                                              `json:"map_id"`
	Environment string                                              `json:"environment"`
	Coordinates string                                              `json:"coordinates"`
	Exits       map[string]GMCPRoomModule_Payload_Contents_ExitInfo `json:"exits"`
	Details     []string                                            `json:"details"`
	Contents    GMCPRoomModule_Payload_Contents                     `json:"contents"`
}

type GMCPRoomModule_Payload_Contents_ExitInfo struct {
	RoomId  int                    `json:"room_id"`
	DeltaX  int                    `json:"delta_x"`
	DeltaY  int                    `json:"delta_y"`
	DeltaZ  int                    `json:"delta_z"`
	Details map[string]interface{} `json:"details,omitempty"` // Only populated for special exits
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
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Adjectives  []string `json:"adjectives"`
	QuestFlag   bool     `json:"quest_flag"`
	ThreatLevel string   `json:"threat_level"` // "peaceful", "hostile", "aggressive", "fighting"
	Targeting   []string `json:"targeting"`    // array of player names this mob is targeting
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
						"name":       util.StripANSI(evt.Item.Name()),
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
						"name": util.StripANSI(evt.Item.Name()),
					},
				})
			}
		}
	}

	return events.Continue
}

func (g *GMCPRoomModule) exitLockChangedHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.ExitLockChanged)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "ExitLockChanged", "Actual Type", e.Type())
		return events.Cancel
	}

	// Load the room where the exit changed
	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

	// Send exit updates to all players in the room
	for _, userId := range room.GetPlayers() {
		events.AddToQueue(GMCPRoomUpdate{UserId: userId, Identifier: `Room.Info.Exits`})
	}

	return events.Continue
}
