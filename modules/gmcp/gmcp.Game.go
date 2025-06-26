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

}

type GMCPGameModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

func (g *GMCPGameModule) onJoinLeave(e events.Event) events.ListenerReturn {

	c := configs.GetConfig()

	tFormat := string(c.TextFormats.Time)

	// Build player list
	players := []map[string]string{}
	for _, user := range users.GetAllActiveUsers() {
		players = append(players, map[string]string{
			"level": IntToString(user.Character.Level),
			"name":  user.Character.Name,
			"title": user.Role,
		})
	}

	// Send to each active user
	for _, user := range users.GetAllActiveUsers() {
		payload := map[string]interface{}{
			"Info": map[string]string{
				"logintime": user.GetConnectTime().Format(tFormat),
				"name":      string(c.Server.MudName),
			},
			"Who": map[string]interface{}{
				"Players": players,
			},
		}

		events.AddToQueue(GMCPOut{
			UserId:  user.UserId,
			Module:  `Game`,
			Payload: payload,
		})
	}

	return events.Continue
}
