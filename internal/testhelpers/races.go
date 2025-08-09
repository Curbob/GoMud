// Package testhelpers provides utilities for testing GoMud components
package testhelpers

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// InitializeTestRaces sets up minimal race data for testing combat and other systems
// that depend on race information. This should be called in test init() functions.
func InitializeTestRaces() {
	// Create a basic human race for testing
	human := &races.Race{
		RaceId:           1,
		Name:             "Human",
		Description:      "A standard human for testing",
		DefaultAlignment: 0,
		Size:             races.Medium,
		TNLScale:         1.0,
		UnarmedName:      "fist",
		Tameable:         false,
		Damage: items.Damage{
			Attacks:     1,
			DiceRoll:    "1d4",
			DiceCount:   1,
			SideCount:   4,
			BonusDamage: 0,
		},
		Selectable:    true,
		KnowsFirstAid: true,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 10},
			Speed:      stats.StatInfo{Base: 10},
			Smarts:     stats.StatInfo{Base: 10},
			Vitality:   stats.StatInfo{Base: 10},
			Mysticism:  stats.StatInfo{Base: 10},
			Perception: stats.StatInfo{Base: 10},
		},
	}

	// Create a basic orc race for variety in testing
	orc := &races.Race{
		RaceId:           2,
		Name:             "Orc",
		Description:      "A standard orc for testing",
		DefaultAlignment: -20,
		Size:             races.Medium,
		TNLScale:         1.1,
		UnarmedName:      "claw",
		Tameable:         false,
		Damage: items.Damage{
			Attacks:     1,
			DiceRoll:    "1d6",
			DiceCount:   1,
			SideCount:   6,
			BonusDamage: 1,
		},
		Selectable:    false,
		KnowsFirstAid: false,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 12},
			Speed:      stats.StatInfo{Base: 8},
			Smarts:     stats.StatInfo{Base: 6},
			Vitality:   stats.StatInfo{Base: 12},
			Mysticism:  stats.StatInfo{Base: 4},
			Perception: stats.StatInfo{Base: 8},
		},
	}

	// Add a small creature for testing size differences
	mouse := &races.Race{
		RaceId:           3,
		Name:             "Mouse",
		Description:      "A small mouse for testing",
		DefaultAlignment: 0,
		Size:             races.Small,
		TNLScale:         0.5,
		UnarmedName:      "bite",
		Tameable:         true,
		Damage: items.Damage{
			Attacks:     1,
			DiceRoll:    "1d2",
			DiceCount:   1,
			SideCount:   2,
			BonusDamage: 0,
		},
		Selectable:    false,
		KnowsFirstAid: false,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 2},
			Speed:      stats.StatInfo{Base: 15},
			Smarts:     stats.StatInfo{Base: 3},
			Vitality:   stats.StatInfo{Base: 3},
			Mysticism:  stats.StatInfo{Base: 1},
			Perception: stats.StatInfo{Base: 12},
		},
	}

	// Use the AddRaceForTesting function to add races to the internal map
	AddRaceForTesting(human)
	AddRaceForTesting(orc)
	AddRaceForTesting(mouse)
}

// AddRaceForTesting adds a race directly to the races map for testing purposes.
// This bypasses the normal file loading mechanism.
func AddRaceForTesting(race *races.Race) {
	races.SetRaceForTesting(race)
}

// CleanupTestRaces removes all test races from the races map.
// This should be called in test cleanup to avoid polluting other tests.
func CleanupTestRaces() {
	races.ClearRacesForTesting()
}
