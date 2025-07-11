package gmcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	TELNET_GMCP term.IACByte = 201 // https://tintin.mudhalla.net/protocols/gmcp/
)

var (
	///////////////////////////
	// GMCP COMMANDS
	///////////////////////////
	GmcpEnable  = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_WILL, TELNET_GMCP}, EndChars: []byte{}} // Indicates the server wants to enable GMCP.
	GmcpDisable = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_WONT, TELNET_GMCP}, EndChars: []byte{}} // Indicates the server wants to disable GMCP.

	GmcpAccept = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_DO, TELNET_GMCP}, EndChars: []byte{}}   // Indicates the client accepts GMCP sub-negotiations.
	GmcpRefuse = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_DONT, TELNET_GMCP}, EndChars: []byte{}} // Indicates the client refuses GMCP sub-negotiations.

	GmcpPayload               = term.TerminalCommand{Chars: []byte{term.TELNET_IAC, term.TELNET_SB, TELNET_GMCP}, EndChars: []byte{term.TELNET_IAC, term.TELNET_SE}} // Wrapper for sending GMCP payloads
	GmcpWebPayload            = term.TerminalCommand{Chars: []byte("!!GMCP("), EndChars: []byte{')'}}                                                                // Wrapper for sending GMCP payloads
	gmcpModule     GMCPModule = GMCPModule{}
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
	gmcpModule = GMCPModule{
		plug: plugins.New(`gmcp`, `1.0`),
	}

	// connectionId to map[string]int
	gmcpModule.cache, _ = lru.New[uint64, GMCPSettings](128)

	gmcpModule.plug.ExportFunction(`SendGMCPEvent`, gmcpModule.sendGMCPEvent)
	gmcpModule.plug.ExportFunction(`IsMudlet`, gmcpModule.IsMudletExportedFunction)
	gmcpModule.plug.ExportFunction(`TriggerRoomUpdate`, gmcpModule.triggerRoomUpdate)

	gmcpModule.plug.Callbacks.SetIACHandler(gmcpModule.HandleIAC)
	gmcpModule.plug.Callbacks.SetOnNetConnect(gmcpModule.onNetConnect)

	events.RegisterListener(GMCPOut{}, gmcpModule.dispatchGMCP)
	events.RegisterListener(events.PlayerSpawn{}, gmcpModule.handlePlayerJoin)

	// Initialize the combat cooldown timer system
	InitCombatCooldownTimer()

	// Note: Other combat GMCP modules (CombatStatus, CombatTarget, CombatEnemies, CombatDamage)
	// are initialized via their own init() functions
}

func isGMCPEnabled(connectionId uint64) bool {

	//return true
	if gmcpData, ok := gmcpModule.cache.Get(connectionId); ok {
		return gmcpData.GMCPAccepted
	}

	return false
}

// ///////////////////
// EVENTS
// ///////////////////

type GMCPOut struct {
	UserId  int
	Module  string
	Payload any
}

func (g GMCPOut) Type() string { return `GMCPOut` }

// ///////////////////
// END EVENTS
// ///////////////////
type GMCPModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
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

// / SETTINGS
type GMCPSettings struct {
	Client struct {
		Name     string
		Version  string
		IsMudlet bool // Knowing whether is a mudlet client can be useful, since Mudlet hates certain ANSI/Escape codes.
	}
	GMCPAccepted   bool           // Do they accept GMCP data?
	EnabledModules map[string]int // Legacy field - Core.Supports.Set is ignored, kept for compatibility
}

func (gs *GMCPSettings) IsMudlet() bool {
	return gs.Client.IsMudlet
}

/// END SETTINGS

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

func (g *GMCPModule) sendGMCPEvent(userId int, moduleName string, payload any) {

	evt := GMCPOut{
		UserId:  userId,
		Module:  moduleName,
		Payload: payload,
	}

	events.AddToQueue(evt)
}

func (g *GMCPModule) triggerRoomUpdate(userId int) {
	// This triggers a full room update, sending all room sub-nodes
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
		// Send request to enable GMCP
		g.sendGMCPEnableRequest(evt.ConnectionId)
	}

	// Send full GMCP update on player spawn (login/reconnect/respawn)
	// This ensures the UI has all structures available to create visuals
	if evt.UserId > 0 {
		SendFullGMCPUpdate(evt.UserId)
		mudlog.Info("GMCP", "type", "PlayerSpawn", "action", "Full GMCP sent on login", "userId", evt.UserId)
	}

	return events.Continue
}

// Sends a telnet IAC request to enable GMCP
func (g *GMCPModule) sendGMCPEnableRequest(connectionId uint64) {
	connections.SendTo(
		GmcpEnable.BytesWithPayload(nil),
		connectionId,
	)
}

// Returns a map of module name to version number
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

	if ok, payload := term.Matches(iacCmd, GmcpAccept); ok {

		gmcpData, ok := g.cache.Get(connectionId)
		if !ok {
			gmcpData = GMCPSettings{}
		}
		gmcpData.GMCPAccepted = true
		g.cache.Add(connectionId, gmcpData)

		mudlog.Debug("Received", "type", "IAC (Client-GMCP Accept)", "data", term.BytesString(payload))
		return true
	}

	if ok, payload := term.Matches(iacCmd, GmcpRefuse); ok {

		gmcpData, ok := g.cache.Get(connectionId)
		if !ok {
			gmcpData = GMCPSettings{}
		}
		gmcpData.GMCPAccepted = false
		g.cache.Add(connectionId, gmcpData)

		mudlog.Debug("Received", "type", "IAC (Client-GMCP Refuse)", "data", term.BytesString(payload))
		return true
	}

	if len(iacCmd) >= 5 && iacCmd[len(iacCmd)-2] == term.TELNET_IAC && iacCmd[len(iacCmd)-1] == term.TELNET_SE {
		// Unhanlded IAC command, log it

		requestBody := iacCmd[3 : len(iacCmd)-2]
		//mudlog.Debug("Received", "type", "GMCP", "size", len(iacCmd), "data", string(requestBody))

		spaceAt := 0
		for i := 0; i < len(requestBody); i++ {
			if requestBody[i] == 32 {
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

		mudlog.Debug("Received", "type", "GMCP (Handling)", "command", command, "payload", string(payload))

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

					// Trigger the Mudlet detected event
					userId := 0
					// Try to find the user ID associated with this connection
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
			decoded := GMCPSupportsSet{}
			if err := json.Unmarshal(payload, &decoded); err == nil {

				gmcpData, ok := g.cache.Get(connectionId)
				if !ok {
					gmcpData = GMCPSettings{}
					gmcpData.GMCPAccepted = true
				}

				gmcpData.EnabledModules = map[string]int{}

				for name, value := range decoded.GetSupportedModules() {

					// Break it down into:
					// Char.Inventory.Backpack
					// Char.Inventory
					// Char
					for {
						gmcpData.EnabledModules[name] = value
						idx := strings.LastIndex(name, ".")
						if idx == -1 {
							break
						}
						name = name[:idx]
					}

				}

				g.cache.Add(connectionId, gmcpData)

			}
		case `Core.Supports.Remove`:
			decoded := GMCPSupportsRemove{}
			if err := json.Unmarshal(payload, &decoded); err == nil {

				gmcpData, ok := g.cache.Get(connectionId)
				if !ok {
					gmcpData = GMCPSettings{}
					gmcpData.GMCPAccepted = true
				}

				if len(gmcpData.EnabledModules) > 0 {
					for _, name := range decoded {
						delete(gmcpData.EnabledModules, name)
					}
				}

				g.cache.Add(connectionId, gmcpData)

			}
		case `Char.Login`:
			decoded := GMCPLogin{}
			if err := json.Unmarshal(payload, &decoded); err == nil {
				mudlog.Debug("GMCP LOGIN", "username", decoded.Name, "password", strings.Repeat(`*`, len(decoded.Password)))
			}

		case `GMCP`:
			// Handle GMCP refresh request
			payloadStr := string(payload)

			// Find the user ID associated with this connection
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
					// Send full GMCP refresh
					SendFullGMCPUpdate(userId)
					mudlog.Info("GMCP", "type", "GMCP", "action", "Full refresh requested", "userId", userId)

				case strings.HasPrefix(payloadStr, `Send`):
					// Handle individual node requests like "SendCharInventoryBackpack"
					// Remove "Send" prefix and convert to dot notation
					nodePath := payloadStr[4:] // Remove "Send"

					// Convert camelCase to dot notation
					// SendCharInventoryBackpack -> Char.Inventory.Backpack
					var dotPath strings.Builder
					for i, char := range nodePath {
						if i > 0 && char >= 'A' && char <= 'Z' {
							dotPath.WriteRune('.')
						}
						dotPath.WriteRune(char)
					}

					identifier := dotPath.String()
					mudlog.Info("GMCP", "type", "GMCP", "action", "Node refresh requested", "userId", userId, "node", identifier)

					// Trigger appropriate update based on the module
					if strings.HasPrefix(identifier, "Char") {
						events.AddToQueue(GMCPCharUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Room") {
						events.AddToQueue(GMCPRoomUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Party") {
						events.AddToQueue(GMCPPartyUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Game") {
						events.AddToQueue(GMCPGameUpdate{UserId: userId, Identifier: identifier})
					} else if strings.HasPrefix(identifier, "Comm") {
						// For Comm.Channel, send an empty structure
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

		// Handle Discord-related messages
		default:
			// Check if it's a Discord message
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
					// Extract the Discord command (Hello, Get, Status)
					discordCommand := ""
					if parts := strings.Split(command, "."); len(parts) >= 3 {
						discordCommand = parts[2] // External.Discord.Hello -> Hello
					}

					// Dispatch a GMCPDiscordMessage event
					events.AddToQueue(GMCPDiscordMessage{
						ConnectionId: connectionId,
						Command:      discordCommand,
						Payload:      payload,
					})

					mudlog.Debug("GMCP", "type", "Discord", "command", discordCommand, "userId", userId)
				}
			}
		}

		return true
	}

	// Unhanlded IAC command, log it
	mudlog.Debug("Received", "type", "GMCP?", "data-size", len(iacCmd), "data-string", string(iacCmd), "data-bytes", iacCmd)

	return true
}

// Checks whether their level is too high for a guide
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

		// Get enabled modules... if none, skip out.
		if !gmcpSettings.GMCPAccepted {
			return events.Continue
		}
	} else {
		gmcpSettings, ok = g.cache.Get(connId)
		if !ok {
			return events.Continue
		}
	}

	// Legacy Core.Supports.Set filtering removed - send all GMCP modules
	// The client's Core.Supports.Set has no bearing on what modules we send

	switch v := gmcp.Payload.(type) {
	case []byte:

		// DEBUG ONLY
		// TODO: REMOVE
		if gmcp.UserId == 1 && os.Getenv("CONSOLE_GMCP_OUTPUT") == "1" {
			var prettyJSON bytes.Buffer
			json.Indent(&prettyJSON, v, "", "\t")
			fmt.Print(gmcp.Module + ` `)
			fmt.Println(prettyJSON.String())
		}

		// Regular code follows...
		if len(gmcp.Module) > 0 {
			v = append([]byte(gmcp.Module+` `), v...)
		}

		if gmcpSettings.Client.Name == `WebClient` {
			connections.SendTo(GmcpWebPayload.BytesWithPayload(v), connId)
		} else {
			connections.SendTo(GmcpPayload.BytesWithPayload(v), connId)
		}

	case string:

		// DEBUG ONLY
		// TODO: REMOVE
		if gmcp.UserId == 1 && os.Getenv("CONSOLE_GMCP_OUTPUT") == "1" {
			var prettyJSON bytes.Buffer
			json.Indent(&prettyJSON, []byte(v), "", "\t")
			fmt.Print(gmcp.Module + ` `)
			fmt.Println(prettyJSON.String())
		}

		// Regular code follows...
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

		// DEBUG ONLY
		// TODO: REMOVE
		if gmcp.UserId == 1 && os.Getenv("CONSOLE_GMCP_OUTPUT") == "1" {
			var prettyJSON bytes.Buffer
			json.Indent(&prettyJSON, payload, "", "\t")
			fmt.Print(gmcp.Module + ` `)
			fmt.Println(prettyJSON.String())
		}

		// Regular code follows...
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

	// Make sure they have GMCP enabled
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return
	}

	// Trigger updates for all modules using the consistent event pattern

	// Char module - sends all character sub-nodes
	events.AddToQueue(GMCPCharUpdate{UserId: userId, Identifier: `Char`})

	// Room module - sends all room sub-nodes
	events.AddToQueue(GMCPRoomUpdate{UserId: userId, Identifier: `Room`})

	// Send Room.Wrongdir with empty data to establish the structure
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Wrongdir`,
		Payload: map[string]string{"dir": ""},
	})

	// Party module - sends all party sub-nodes
	events.AddToQueue(GMCPPartyUpdate{UserId: userId, Identifier: `Party`})

	// Game module - sends all game sub-nodes
	events.AddToQueue(GMCPGameUpdate{UserId: userId, Identifier: `Game`})

	// Combat module - send all combat status nodes
	// Send current combat status
	events.AddToQueue(GMCPCombatStatusUpdate{UserId: userId})

	// Send combat target (if any)
	events.AddToQueue(GMCPCombatTargetUpdate{UserId: userId})

	// Send combat enemies list
	events.AddToQueue(GMCPCombatEnemiesUpdate{UserId: userId})

	// Send combat cooldown timer with default values
	timingConfig := configs.GetTimingConfig()
	events.AddToQueue(GMCPCombatCooldownUpdate{
		UserId:          userId,
		CooldownSeconds: 0.0,
		MaxSeconds:      float64(timingConfig.RoundSeconds),
		NameActive:      "Combat Round",
		NameIdle:        "Ready",
	})

	// Send empty damage structure to establish it
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

	// Send Room.Remove.Player with empty data to establish the structure
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Remove.Player`,
		Payload: map[string]string{"name": ""},
	})

	// Send Room.Remove.Npc with empty data to establish the structure
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Remove.Npc`,
		Payload: map[string]interface{}{"id": "", "name": ""},
	})

	// Send Room.Remove.Item with empty data to establish the structure
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Remove.Item`,
		Payload: map[string]interface{}{"id": "", "name": ""},
	})

	// Send Room.Add.Player with empty data to establish the structure
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Add.Player`,
		Payload: map[string]string{"name": ""},
	})

	// Send Room.Add.Npc with empty data to establish the structure
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

	// Send Room.Add.Item with empty data to establish the structure
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Room.Add.Item`,
		Payload: map[string]interface{}{"id": "", "name": "", "quest_flag": false},
	})

	// Send all combat event structures with expected fields
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

	// Comm module - send channel structure with all fields
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
