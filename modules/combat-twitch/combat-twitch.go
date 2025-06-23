package combattwitch

import (
	"embed"
	_ "embed"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/plugins"
)

//go:embed files/*
var files embed.FS

func init() {
	// Create the plugin
	plug := plugins.New(`combat-twitch`, `1.0`)

	// Attach the embedded filesystem
	if err := plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	// Create and register the combat system
	combatSystem := NewTwitchCombat(plug)
	if err := combat.RegisterCombatSystem("combat-twitch", combatSystem); err != nil {
		panic("Failed to register combat-twitch module: " + err.Error())
	}

	// Store the plugin reference for later use
	combatSystem.plug = plug
}
