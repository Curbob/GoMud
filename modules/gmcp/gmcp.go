// Package gmcp implements the Generic MUD Communication Protocol for real-time
// client-server data exchange over Telnet IAC subnegotiation or WebSocket frames.
//
// GMCP provides structured JSON data updates to clients for UI state management,
// including character stats, room information, combat state, and party data.
// The protocol is transport-agnostic, supporting both traditional Telnet and modern WebSocket connections.
package gmcp

import (
	"embed"
	"encoding/json"
	"strconv"
	"strings"

	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	TELNET_GMCP term.IACByte = 201 // https://tintin.mudhalla.net/protocols/gmcp/
)

var (
	//go:embed files/*
	files embed.FS

	// Telnet IAC negotiation sequences for GMCP protocol
	GmcpEnable  = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_WILL, TELNET_GMCP}, EndChars: []byte{}}
	GmcpDisable = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_WONT, TELNET_GMCP}, EndChars: []byte{}}

	GmcpAccept = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_DO, TELNET_GMCP}, EndChars: []byte{}}
	GmcpRefuse = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_DONT, TELNET_GMCP}, EndChars: []byte{}}

	// GMCP payload wrappers for different transport types
	GmcpPayload    = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_SB, TELNET_GMCP}, EndChars: []byte{term.TELNET_IAC, term.TELNET_SE}}
	GmcpWebPayload = term.TerminalCommand{Chars: []byte("!!GMCP("), EndChars: []byte{')'}}
	gmcpModule     GMCPModule = GMCPModule{}

	// Central combat state tracking - shared by all combat modules
	combatUsersMutex sync.RWMutex
	combatUsers      = make(map[int]struct{})
)

func init() {
	gmcpModule = GMCPModule{
		plug: plugins.New(`gmcp`, `1.0`),
	}

	// LRU cache limits memory usage for connection settings to 128 entries
	gmcpModule.cache, _ = lru.New[uint64, GMCPSettings](128)

	// Embedded filesystem provides config overlays without disk access
	if err := gmcpModule.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	gmcpModule.plug.Callbacks.SetOnLoad(gmcpModule.load)
	gmcpModule.plug.Callbacks.SetOnSave(gmcpModule.save)

	gmcpModule.plug.ExportFunction(`SendGMCPEvent`, gmcpModule.sendGMCPEvent)
	gmcpModule.plug.ExportFunction(`IsMudlet`, gmcpModule.IsMudletExportedFunction)
	gmcpModule.plug.ExportFunction(`TriggerRoomUpdate`, gmcpModule.triggerRoomUpdate)

	gmcpModule.plug.Callbacks.SetIACHandler(gmcpModule.HandleIAC)
	gmcpModule.plug.Callbacks.SetOnNetConnect(gmcpModule.onNetConnect)

	events.RegisterListener(events.CombatStarted{}, handleCombatStartedTracking)
	events.RegisterListener(events.CombatEnded{}, handleCombatEndedTracking)
	events.RegisterListener(events.PlayerDespawn{}, handlePlayerDespawnTracking)

	events.RegisterListener(GMCPOut{}, gmcpModule.dispatchGMCP)
	events.RegisterListener(events.PlayerSpawn{}, gmcpModule.handlePlayerJoin)

	InitCombatCooldownTimer()

	initMudlet()
}

func isGMCPEnabled(connectionId uint64) bool {
	if gmcpData, ok := gmcpModule.cache.Get(connectionId); ok {
		return gmcpData.GMCPAccepted
	}

	return false
}

// validateUserForGMCP provides centralized validation for all GMCP operations.
// Prevents operations on disconnected users and non-GMCP clients.
func validateUserForGMCP(userId int, module string) (*users.UserRecord, bool) {
	if userId < 1 {
		return nil, false
	}

	user := users.GetByUserId(userId)
	if user == nil {
		mudlog.Warn(module, "action", "validateUserForGMCP", "issue", "user not found", "userId", userId)
		return nil, false
	}

	gmcpEnabled := isGMCPEnabled(user.ConnectionId())
	if !gmcpEnabled {
		return nil, false
	}

	return user, true
}

type GMCPOut struct {
	UserId  int
	Module  string
	Payload any
}

func (g GMCPOut) Type() string { return `GMCPOut` }

type GMCPModule struct {
	plug  *plugins.Plugin
	cache *lru.Cache[uint64, GMCPSettings]
}

type GMCPHello struct {
	Client  string
	Version string
}

type GMCPSupportsSet []string

type GMCPSupportsRemove = []string

type GMCPLogin struct {
	Name     string
	Password string
}

type GMCPSettings struct {
	Client struct {
		Name     string
		Version  string
		IsMudlet bool // Mudlet requires special handling for ANSI codes and mapper integration
	}
	GMCPAccepted bool
}

func (gs *GMCPSettings) IsMudlet() bool {
	return gs.Client.IsMudlet
}

func handleCombatStartedTracking(e events.Event) events.ListenerReturn {
	mudlog.Info("GMCP Combat Tracking", "event", "CombatStarted received")
	evt, ok := e.(events.CombatStarted)
	if !ok {
		mudlog.Error("GMCP Combat Tracking", "error", "CombatStarted type assertion failed")
		return events.Continue
	}

	mudlog.Info("GMCP Combat Tracking", "event", "CombatStarted",
		"attackerType", evt.AttackerType, "attackerId", evt.AttackerId,
		"defenderType", evt.DefenderType, "defenderId", evt.DefenderId,
		"initiatedBy", evt.InitiatedBy)

	if evt.AttackerType == "player" {
		combatUsersMutex.Lock()
		combatUsers[evt.AttackerId] = struct{}{}
		combatUsersMutex.Unlock()
		TrackCombatPlayer(evt.AttackerId)
		mudlog.Info("GMCP Combat Tracking", "action", "Added player to combat", "userId", evt.AttackerId)
	}

	if evt.DefenderType == "player" {
		combatUsersMutex.Lock()
		combatUsers[evt.DefenderId] = struct{}{}
		combatUsersMutex.Unlock()
		TrackCombatPlayer(evt.DefenderId)
		mudlog.Info("GMCP Combat Tracking", "action", "Added player to combat", "userId", evt.DefenderId)
	}

	return events.Continue
}

func handleCombatEndedTracking(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CombatEnded)
	if !ok || evt.EntityType != "player" {
		return events.Continue
	}

	combatUsersMutex.Lock()
	delete(combatUsers, evt.EntityId)
	combatUsersMutex.Unlock()

	UntrackCombatPlayer(evt.EntityId)

	return events.Continue
}

func handlePlayerDespawnTracking(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDespawn)
	if !ok {
		return events.Continue
	}

	combatUsersMutex.Lock()
	delete(combatUsers, evt.UserId)
	combatUsersMutex.Unlock()

	CleanupUser(evt.UserId)

	return events.Continue
}

func GetUsersInCombat() []int {
	combatUsersMutex.RLock()
	defer combatUsersMutex.RUnlock()

	usersInCombat := make([]int, 0, len(combatUsers))
	for userId := range combatUsers {
		usersInCombat = append(usersInCombat, userId)
	}
	return usersInCombat
}

// IsUserInCombat is the authoritative source for combat state.
// Returns true if user is attacking (has aggro) or being attacked (mob has aggro on them).
func IsUserInCombat(userId int) bool {
	user := users.GetByUserId(userId)
	if user == nil {
		return false
	}

	// Both UserId and MobInstanceId checked since aggro can target either players or mobs
	if user.Character.Aggro != nil && (user.Character.Aggro.UserId > 0 || user.Character.Aggro.MobInstanceId > 0) {
		return true
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return false
	}

	for _, mobId := range room.GetMobs() {
		if mob := mobs.GetInstance(mobId); mob != nil {
			if mob.Character.Aggro != nil && mob.Character.Aggro.UserId == userId {
				return true
			}
		}
	}

	return false
}

func GetUsersInOrTargetedByCombat() []int {
	result := []int{}

	for _, user := range users.GetAllActiveUsers() {
		if IsUserInCombat(user.UserId) {
			result = append(result, user.UserId)
		}
	}

	return result
}

// CleanupUser orchestrates cleanup across all GMCP modules to prevent memory leaks
func CleanupUser(userId int) {
	cleanupCombatStatus(userId)
	cleanupCombatTargetNew(userId)
	cleanupCombatEnemiesNew(userId)
	UntrackCombatPlayer(userId)
}

func (g *GMCPModule) IsMudletExportedFunction(connectionId uint64) bool {
	gmcpData, ok := g.cache.Get(connectionId)
	if !ok {
		return false
	}
	return gmcpData.IsMudlet()
}

func (g *GMCPModule) onNetConnect(n plugins.NetConnection) {

	if n.IsWebSocket() {
		setting := GMCPSettings{}
		setting.Client.Name = `WebClient`
		setting.Client.Version = `1.0.0`
		setting.GMCPAccepted = true
		g.cache.Add(n.ConnectionId(), setting)
		return
	}

	g.cache.Add(n.ConnectionId(), GMCPSettings{})

	g.sendGMCPEnableRequest(n.ConnectionId())
}

func (g *GMCPModule) isGMCPCommand(b []byte) bool {
	return len(b) > 2 && b[0] == term.TELNET_IAC && b[2] == TELNET_GMCP
}

func (g *GMCPModule) load() {
}

func (g *GMCPModule) save() {
}

func (g *GMCPModule) sendGMCPEvent(userId int, moduleName string, payload any) {

	evt := GMCPOut{
		UserId:  userId,
		Module:  moduleName,
		Payload: payload,
	}

	events.AddToQueue(evt)
}

func (g *GMCPModule) triggerRoomUpdate(userId int) {
	events.AddToQueue(GMCPRoomUpdate{
		UserId:     userId,
		Identifier: `Room.Info`,
	})
}

func (g *GMCPModule) handlePlayerJoin(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerSpawn", "Actual Type", e.Type())
		return events.Cancel
	}

	if _, ok := g.cache.Get(evt.ConnectionId); !ok {
		g.cache.Add(evt.ConnectionId, GMCPSettings{})
		g.sendGMCPEnableRequest(evt.ConnectionId)
	}

	if evt.UserId > 0 {
		SendFullGMCPUpdate(evt.UserId)
		mudlog.Info("GMCP", "type", "PlayerSpawn", "action", "Full GMCP sent on login", "userId", evt.UserId)
	}

	return events.Continue
}

func (g *GMCPModule) sendGMCPEnableRequest(connectionId uint64) {
	connections.SendTo(
		GmcpEnable.BytesWithPayload(nil),
		connectionId,
	)
}

func (s GMCPSupportsSet) GetSupportedModules() map[string]int {
	ret := map[string]int{}

	for _, entry := range s {
		parts := strings.Split(entry, ` `)
		if len(parts) == 2 {
			ret[parts[0]], _ = strconv.Atoi(parts[1])
		}
	}

	return ret
}

func (g *GMCPModule) HandleIAC(connectionId uint64, iacCmd []byte) bool {

	if !g.isGMCPCommand(iacCmd) {
		return false
	}

	if ok, _ := term.Matches(iacCmd, GmcpAccept); ok {

		gmcpData, ok := g.cache.Get(connectionId)
		if !ok {
			gmcpData = GMCPSettings{}
		}
		gmcpData.GMCPAccepted = true
		g.cache.Add(connectionId, gmcpData)

		return true
	}

	if ok, _ := term.Matches(iacCmd, GmcpRefuse); ok {

		gmcpData, ok := g.cache.Get(connectionId)
		if !ok {
			gmcpData = GMCPSettings{}
		}
		gmcpData.GMCPAccepted = false
		g.cache.Add(connectionId, gmcpData)

		return true
	}

	if len(iacCmd) >= 5 && iacCmd[len(iacCmd)-2] == term.TELNET_IAC && iacCmd[len(iacCmd)-1] == term.TELNET_SE {
		requestBody := iacCmd[3 : len(iacCmd)-2]

		spaceAt := 0
		for i := 0; i < len(requestBody); i++ {
			if requestBody[i] == 32 { // ASCII space separates command from JSON payload
				spaceAt = i
				break
			}
		}

		command := ``
		payload := []byte{}

		if spaceAt > 0 && spaceAt < len(requestBody) {
			command = string(requestBody[0:spaceAt])
			payload = requestBody[spaceAt+1:]
		} else {
			command = string(requestBody)
		}

		switch command {

		case `Core.Hello`:
			decoded := GMCPHello{}
			if err := json.Unmarshal(payload, &decoded); err == nil {

				gmcpData, ok := g.cache.Get(connectionId)
				if !ok {
					gmcpData = GMCPSettings{}
					gmcpData.GMCPAccepted = true
				}

				gmcpData.Client.Name = decoded.Client
				gmcpData.Client.Version = decoded.Version

				if strings.EqualFold(decoded.Client, `mudlet`) {
					gmcpData.Client.IsMudlet = true

					userId := 0
					for _, user := range users.GetAllActiveUsers() {
						if user.ConnectionId() == connectionId {
							userId = user.UserId
							break
						}
					}

					if userId > 0 {
						events.AddToQueue(GMCPMudletDetected{
							ConnectionId: connectionId,
							UserId:       userId,
						})
					}
				}

				g.cache.Add(connectionId, gmcpData)
			}
		case `Core.Supports.Set`:
			// Intentionally ignored - we send all modules regardless of client preferences
		case `Core.Supports.Remove`:
			// Intentionally ignored - we send all modules regardless of client preferences
		case `Char.Login`:
			decoded := GMCPLogin{}
			if err := json.Unmarshal(payload, &decoded); err == nil {
			}

		case `GMCP`:
			payloadStr := string(payload)
			userId := 0
			for _, user := range users.GetAllActiveUsers() {
				if user.ConnectionId() == connectionId {
					userId = user.UserId
					break
				}
			}

			if userId > 0 {
				switch {
				case payloadStr == `SendFullPayload`:
					SendFullGMCPUpdate(userId)
					mudlog.Info("GMCP", "type", "GMCP", "action", "Full refresh requested", "userId", userId)

				case strings.HasPrefix(payloadStr, `Send`):
					// Convert CamelCase commands to dot notation for module routing
					// Example: "SendCharInventoryBackpack" -> "Char.Inventory.Backpack"
					nodePath := payloadStr[4:]
					var dotPath strings.Builder
					for i, char := range nodePath {
						if i > 0 && char >= 'A' && char <= 'Z' {
							dotPath.WriteRune('.')
						}
						dotPath.WriteRune(char)
					}

					identifier := dotPath.String()
					mudlog.Info("GMCP", "type", "GMCP", "action", "Node refresh requested", "userId", userId, "node", identifier)

					if strings.HasPrefix(identifier, "Char") {
						events.AddToQueue(GMCPCharUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Room") {
						events.AddToQueue(GMCPRoomUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Party") {
						events.AddToQueue(GMCPPartyUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Game") {
						events.AddToQueue(GMCPGameUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Client.Map") {
						if mudletHandler != nil && mudletHandler.isMudletClient(userId) {
							mudletHandler.sendMudletMapConfig(userId)
						}
					} else if strings.HasPrefix(identifier, "Comm") {
						events.AddToQueue(GMCPOut{
							UserId: userId,
							Module: `Comm.Channel`,
							Payload: map[string]string{
								"channel": "",
								"sender":  "",
								"source":  "",
								"text":    "",
							},
						})
					}
				}
			}

		default:
			if strings.HasPrefix(command, "External.Discord") {
				// Try to find the user ID associated with this connection
				userId := 0
				for _, user := range users.GetAllActiveUsers() {
					if user.ConnectionId() == connectionId {
						userId = user.UserId
						break
					}
				}

				if userId > 0 {
					discordCommand := ""
					if parts := strings.Split(command, "."); len(parts) >= 3 {
						discordCommand = parts[2] // Extract command: External.Discord.Hello -> Hello
					}
					events.AddToQueue(GMCPDiscordMessage{
						ConnectionId: connectionId,
						Command:      discordCommand,
						Payload:      payload,
					})

				}
			}
		}

		return true
	}

	return true
}

func (g *GMCPModule) dispatchGMCP(e events.Event) events.ListenerReturn {

	gmcp, typeOk := e.(GMCPOut)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPOut", "Actual Type", e.Type())
		return events.Cancel
	}

	if gmcp.UserId < 1 {
		return events.Continue
	}

	connId := users.GetConnectionId(gmcp.UserId)
	if connId == 0 {
		return events.Continue
	}

	var gmcpSettings GMCPSettings
	var ok bool
	if !isGMCPEnabled(connId) {
		gmcpSettings, ok = g.cache.Get(connId)
		if !ok {
			gmcpSettings = GMCPSettings{}
			g.cache.Add(connId, gmcpSettings)

			g.sendGMCPEnableRequest(connId)

			return events.Continue
		}

		if !gmcpSettings.GMCPAccepted {
			return events.Continue
		}
	} else {
		gmcpSettings, ok = g.cache.Get(connId)
		if !ok {
			return events.Continue
		}
	}

	switch v := gmcp.Payload.(type) {
	case []byte:

		if len(gmcp.Module) > 0 {
			v = append([]byte(gmcp.Module+` `), v...)
		}

		if gmcpSettings.Client.Name == `WebClient` {
			connections.SendTo(GmcpWebPayload.BytesWithPayload(v), connId)
		} else {
			connections.SendTo(GmcpPayload.BytesWithPayload(v), connId)
		}

	case string:

		if len(gmcp.Module) > 0 {
			v = gmcp.Module + ` ` + v
		}

		if gmcpSettings.Client.Name == `WebClient` {
			connections.SendTo(GmcpWebPayload.BytesWithPayload([]byte(v)), connId)
		} else {
			connections.SendTo(GmcpPayload.BytesWithPayload([]byte(v)), connId)
		}

	default:
		payload, err := json.Marshal(gmcp.Payload)
		if err != nil {
			mudlog.Error("Event", "Type", "GMCPOut", "data", gmcp.Payload, "error", err)
			return events.Continue
		}

		if len(gmcp.Module) > 0 {
			payload = append([]byte(gmcp.Module+` `), payload...)
		}

		if gmcpSettings.Client.Name == `WebClient` {
			connections.SendTo(GmcpWebPayload.BytesWithPayload(payload), connId)
		} else {
			connections.SendTo(GmcpPayload.BytesWithPayload(payload), connId)
		}

	}

	return events.Continue
}

// SendFullGMCPUpdate sends all GMCP modules data to a specific user
// This is useful when a client needs to resync all GMCP data
func SendFullGMCPUpdate(userId int) {
	if userId < 1 {
		return
	}

	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return
	}

	events.AddToQueue(GMCPCharUpdate{UserId: userId, Identifier: `Char`})
	events.AddToQueue(GMCPRoomUpdate{UserId: userId, Identifier: `Room`})

	// Schema establishment - empty structures define expected fields for clients
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Wrongdir`,
		Payload: map[string]string{"dir": ""},
	})

	events.AddToQueue(GMCPPartyUpdate{UserId: userId, Identifier: `Party`})
	events.AddToQueue(GMCPGameUpdate{UserId: userId, Identifier: `Game`})
	events.AddToQueue(GMCPCombatStatusUpdate{UserId: userId})

	events.AddToQueue(GMCPCombatTargetUpdate{UserId: userId})
	events.AddToQueue(GMCPCombatEnemiesUpdate{UserId: userId})

	timingConfig := configs.GetTimingConfig()
	events.AddToQueue(GMCPCombatCooldownUpdate{
		UserId:          userId,
		CooldownSeconds: 0.0,
		MaxSeconds:      float64(timingConfig.RoundSeconds),
		NameActive:      "Combat Round",
		NameIdle:        "Ready",
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.Damage`,
		Payload: map[string]interface{}{
			"amount": 0,
			"type":   "",
			"source": "",
			"target": "",
		},
	})

	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Remove.Player`,
		Payload: map[string]string{"name": ""},
	})
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Remove.Npc`,
		Payload: map[string]interface{}{"id": "", "name": ""},
	})
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Remove.Item`,
		Payload: map[string]interface{}{"id": "", "name": ""},
	})

	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Add.Player`,
		Payload: map[string]string{"name": ""},
	})
	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Room.Add.Npc`,
		Payload: map[string]interface{}{
			"id":            "",
			"name":          "",
			"threat_level":  "",
			"targeting_you": false,
		},
	})
	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Room.Add.Item`,
		Payload: map[string]interface{}{
			"id": "", "name": "", "quest_flag": false,
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.Started`,
		Payload: map[string]interface{}{
			"role":         "",
			"target_id":    0,
			"target_type":  "",
			"target_name":  "",
			"initiated_by": "",
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.Ended`,
		Payload: map[string]interface{}{
			"reason":   "",
			"duration": 0,
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.DamageDealt`,
		Payload: map[string]interface{}{
			"target_id":       0,
			"target_type":     "",
			"target_name":     "",
			"amount":          0,
			"damage_type":     "",
			"weapon_name":     "",
			"spell_name":      "",
			"is_critical":     false,
			"is_killing_blow": false,
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.DamageReceived`,
		Payload: map[string]interface{}{
			"source_id":       0,
			"source_type":     "",
			"source_name":     "",
			"amount":          0,
			"damage_type":     "",
			"weapon_name":     "",
			"spell_name":      "",
			"is_critical":     false,
			"is_killing_blow": false,
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.AttackMissed`,
		Payload: map[string]interface{}{
			"defender_id":   0,
			"defender_type": "",
			"defender_name": "",
			"avoid_type":    "",
			"weapon_name":   "",
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.AttackAvoided`,
		Payload: map[string]interface{}{
			"attacker_id":   0,
			"attacker_type": "",
			"attacker_name": "",
			"avoid_type":    "",
			"weapon_name":   "",
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Char.Combat.Fled`,
		Payload: map[string]interface{}{
			"direction":    "",
			"success":      false,
			"prevented_by": "",
		},
	})

	events.AddToQueue(GMCPOut{
		UserId: userId,
		Module: `Comm.Channel`,
		Payload: map[string]string{
			"channel": "",
			"sender":  "",
			"source":  "",
			"text":    "",
		},
	})
}
