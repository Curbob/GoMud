package combat

import (
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	// Register the combat subsystem with copyover
	copyover.Register("combat", gatherCombatState, restoreCombatState)
}

// gatherCombatState collects state about active combat
func gatherCombatState() (interface{}, error) {
	registry := GetStateRegistry()

	// Update round number to current
	registry.UpdateRound(util.GetRoundCount())

	// Cleanup stale combats before saving
	registry.CleanupStaleCombats()

	// Save the state
	data, err := registry.SaveState()
	if err != nil {
		mudlog.Error("Copyover", "subsystem", "combat", "error", "Failed to save combat state", "err", err)
		return nil, err
	}

	mudlog.Info("Copyover", "subsystem", "combat",
		"saved_combats", len(registry.GetAllCombats()),
		"saved_spells", len(registry.GetAllSpellCasts()))

	return data, nil
}

// restoreCombatState restores combat state after copyover
func restoreCombatState(data interface{}) error {
	// Type assertion for byte data
	byteData, ok := data.([]byte)
	if !ok {
		// Try to handle if it was marshaled as a string
		if strData, ok := data.(string); ok {
			byteData = []byte(strData)
		} else {
			mudlog.Error("Copyover", "subsystem", "combat", "error", "Invalid data type for combat state")
			return nil // Don't fail copyover for this
		}
	}

	registry := GetStateRegistry()

	if err := registry.RestoreState(byteData); err != nil {
		mudlog.Error("Copyover", "subsystem", "combat", "error", "Failed to restore combat state", "err", err)
		return err
	}

	mudlog.Info("Copyover", "subsystem", "combat",
		"restored_combats", len(registry.GetAllCombats()),
		"restored_spells", len(registry.GetAllSpellCasts()))

	return nil
}
