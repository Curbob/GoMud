package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/plugins"
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
	g := GMCPGameModule{
		plug: plugins.New(`gmcp.Game`, `1.0`),
	}

	events.RegisterListener(events.PlayerDespawn{}, g.onJoinLeave)
	events.RegisterListener(events.PlayerSpawn{}, g.onJoinLeave)
	events.RegisterListener(GMCPGameUpdate{}, g.buildAndSendGMCPPayload)

}

type GMCPGameModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

// GMCPGameUpdate is used to request Game module updates
type GMCPGameUpdate struct {
	UserId     int
	Identifier string
}

func (g GMCPGameUpdate) Type() string { return `GMCPGameUpdate` }

func (g *GMCPGameModule) onJoinLeave(e events.Event) events.ListenerReturn {

	// Handle PlayerSpawn - send Game modules to the spawning player
	if spawnEvt, isSpawn := e.(events.PlayerSpawn); isSpawn {
		if spawnEvt.UserId == 0 {
			return events.Continue
		}

		// Don't send Game modules here - SendFullGMCPUpdate handles it
		// This prevents duplicate sending and potential race conditions
		// g.sendAllGameNodes(spawnEvt.UserId)
	}

	// For both spawn and despawn, update Game.Who for all active users
	// since the player list has changed
	players := []map[string]interface{}{}

	for _, user := range users.GetAllActiveUsers() {
		players = append(players, map[string]interface{}{
			"level": user.Character.Level,
			"name":  user.Character.Name,
			"title": user.Role,
		})
	}

	// Send updated Game.Who.Players to all active users
	for _, user := range users.GetAllActiveUsers() {
		events.AddToQueue(GMCPOut{
			UserId:  user.UserId,
			Module:  `Game.Who.Players`,
			Payload: players,
		})
	}

	return events.Continue
}

func (g *GMCPGameModule) buildAndSendGMCPPayload(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPGameUpdate)
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

	// If requesting "Game", send all sub-nodes
	if evt.Identifier == `Game` {
		g.sendAllGameNodes(evt.UserId)
		return events.Continue
	}

	// Otherwise, send the specific node requested (not implemented for now)
	// Individual Game sub-nodes could be added here if needed

	return events.Continue
}

// sendAllGameNodes sends all Game nodes as individual GMCP messages
func (g *GMCPGameModule) sendAllGameNodes(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return
	}

	c := configs.GetConfig()
	tFormat := string(c.TextFormats.Time)

	// Send Game.Info
	infoPayload := map[string]interface{}{
		"engine":     "GoMud",
		"login_time": user.GetConnectTime().Format(tFormat),
		"name":       string(c.Server.MudName),
	}

	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Game.Info`,
		Payload: infoPayload,
	})

	// Build and send Game.Who.Players
	players := []map[string]interface{}{}

	for _, u := range users.GetAllActiveUsers() {
		players = append(players, map[string]interface{}{
			"level": u.Character.Level,
			"name":  u.Character.Name,
			"title": u.Role,
		})
	}

	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Game.Who.Players`,
		Payload: players,
	})
}
