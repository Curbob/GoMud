package combatrounds

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
	plug := plugins.New(`combat-rounds`, `1.0`)

	// Attach the embedded filesystem
	if err := plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	// Create and register the combat system
	combatSystem := NewRoundBasedCombat(plug)
	if err := combat.RegisterCombatSystem("combat-rounds", combatSystem); err != nil {
		panic("Failed to register combat-rounds module: " + err.Error())
	}

	// Store the plugin reference for later use
	combatSystem.plug = plug
}
