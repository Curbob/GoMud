package configs

type Statistics struct {
	BaseStats StatDefaults `yaml:"BaseStats"`
	Factors   StatFactors  `yaml:"Factors"`
}

type StatDefaults struct {
	Strength   ConfigInt `yaml:"Strength"`
	Speed      ConfigInt `yaml:"Speed"`
	Smarts     ConfigInt `yaml:"Smarts"`
	Vitality   ConfigInt `yaml:"Vitality"`
	Mysticism  ConfigInt `yaml:"Mysticism"`
	Perception ConfigInt `yaml:"Perception"`
}

type StatFactors struct {
	BaseModFactor         ConfigFloat `yaml:"BaseModFactor"`         // How much of a scaling to aply to levels before multiplying by racial stat
	NaturalGainsModFactor ConfigFloat `yaml:"NaturalGainsModFactor"` // Free stats gained per level modded by this.
}

func (s *Statistics) Validate() {
	if s.BaseStats.Strength < 1 {
		s.BaseStats.Strength = 1
	}
	if s.BaseStats.Speed < 1 {
		s.BaseStats.Speed = 1
	}
	if s.BaseStats.Smarts < 1 {
		s.BaseStats.Smarts = 1
	}
	if s.BaseStats.Vitality < 1 {
		s.BaseStats.Vitality = 1
	}
	if s.BaseStats.Mysticism < 1 {
		s.BaseStats.Mysticism = 1
	}
	if s.BaseStats.Perception < 1 {
		s.BaseStats.Perception = 1
	}

	// Validate factors are reasonable
	if s.Factors.BaseModFactor <= 0 {
		s.Factors.BaseModFactor = 0.3333333334 // default
	}
	if s.Factors.NaturalGainsModFactor <= 0 {
		s.Factors.NaturalGainsModFactor = 0.5 // default
	}
}

func GetStatisticsConfig() Statistics {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Statistics
}
