package stats

import (
	"math"
	"strings"
)

const (
	BaseModFactor         = 0.3333333334 // How much of a scaling to aply to levels before multiplying by racial stat
	NaturalGainsModFactor = 0.5          // Free stats gained per level modded by this.
)

type Statistics struct {
	Strength   StatInfo `yaml:"strength,omitempty"`   // Muscular strength (damage?)
	Speed      StatInfo `yaml:"speed,omitempty"`      // Speed and agility (dodging)
	Smarts     StatInfo `yaml:"smarts,omitempty"`     // Intelligence and wisdom (magic power, memory, deduction, etc)
	Vitality   StatInfo `yaml:"vitality,omitempty"`   // Health and stamina (health capacity)
	Mysticism  StatInfo `yaml:"mysticism,omitempty"`  // Magic and mana (magic capacity)
	Perception StatInfo `yaml:"perception,omitempty"` // How well you notice things
}

// GetStatInfoNames returns a list of all stat names in the order they are defined.
// TODO: This should be a representation of the stats in the game, not hardcoded.
func (s *Statistics) GetStatInfoNames() []string {
	names := []string{
		"Strength",
		"Speed",
		"Smarts",
		"Vitality",
		"Mysticism",
		"Perception",
	}
	return names
}

// Get returns a pointer to the StatInfo for the given name.
func (s *Statistics) Get(name string) *StatInfo {
	key := strings.ToLower(name)

	// TODO: When we load the stats from a file, we need to check the map
	// if stat, ok := s.Stats[key]; ok {
	// 	copy := stat
	// 	return &copy
	// }

	switch key {
	case "strength":
		return &s.Strength
	case "speed":
		return &s.Speed
	case "smarts":
		return &s.Smarts
	case "vitality":
		return &s.Vitality
	case "mysticism":
		return &s.Mysticism
	case "perception":
		return &s.Perception
	}

	return &StatInfo{}
}

// When saving to a file, we don't need to write all the properties that we calculate.
// Just keep track of "Training" because that's not calculated.
type StatInfo struct {
	Training int `yaml:"training,omitempty"` // How much it's been trained with Training Points spending
	Value    int `yaml:"-"`                  // Final calculated value
	ValueAdj int `yaml:"-"`                  // Final calculated value (Adjusted)
	Racial   int `yaml:"-"`                  // Value provided by racial benefits
	Base     int `yaml:"base,omitempty"`     // Base stat value
	Mods     int `yaml:"-"`                  // How much it's modded by equipment, spells, etc.
}

func (si *StatInfo) SetMod(mod ...int) {
	if len(mod) == 0 {
		si.Mods = 0
		return
	}
	si.Mods = 0
	for _, m := range mod {
		si.Mods += m
	}
}

func (si *StatInfo) GainsForLevel(level int) int {
	if level < 1 {
		level = 1
	}
	levelScale := float64(level-1) * BaseModFactor
	basePoints := int(levelScale * float64(si.Base))

	// every x levels we get natural gains
	freeStatPoints := int(float64(level) * NaturalGainsModFactor)

	return basePoints + freeStatPoints
}

func (si *StatInfo) Recalculate(level int) {
	si.Racial = si.GainsForLevel(level)
	si.Value = si.Racial + si.Training + si.Mods
	si.ValueAdj = si.Value
	if si.ValueAdj >= 105 {
		overage := si.ValueAdj - 100
		si.ValueAdj = 100 + int(math.Round(math.Sqrt(float64(overage))*2))
	}
}
