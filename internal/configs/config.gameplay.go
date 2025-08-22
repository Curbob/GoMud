package configs

import (
	"strconv"
	"strings"
)

type GamePlay struct {
	AllowItemBuffRemoval ConfigBool `yaml:"AllowItemBuffRemoval"`
	// Death related settings
	Death GameplayDeath `yaml:"Death"`

	LivesStart     ConfigInt `yaml:"LivesStart"`     // Starting permadeath lives
	LivesMax       ConfigInt `yaml:"LivesMax"`       // Maximum permadeath lives
	LivesOnLevelUp ConfigInt `yaml:"LivesOnLevelUp"` // # lives gained on level up
	PricePerLife   ConfigInt `yaml:"PricePerLife"`   // Price in gold to buy new lives
	// Shops/Conatiners
	ShopRestockRate  ConfigString `yaml:"ShopRestockRate"`  // Default time it takes to restock 1 quantity in shops
	ContainerSizeMax ConfigInt    `yaml:"ContainerSizeMax"` // How many objects containers can hold before overflowing
	// Alt chars
	MaxAltCharacters ConfigInt `yaml:"MaxAltCharacters"` // How many characters beyond the default character can they create?
	// Combat
	ConsistentAttackMessages ConfigBool     `yaml:"ConsistentAttackMessages"` // Whether each weapon has consistent attack messages
	Combat                   GameplayCombat `yaml:"Combat"`                   // Combat-specific settings

	// PVP Restrictions
	PVP             ConfigString `yaml:"PVP"`
	PVPMinimumLevel ConfigInt    `yaml:"PVPMinimumLevel"`
	// XpScale (difficulty)
	XPScale           ConfigFloat `yaml:"XPScale"`
	MobConverseChance ConfigInt   `yaml:"MobConverseChance"` // Chance 1-100 of attempting to converse when idle
}

type GameplayCombat struct {
	// Combat system selection
	Style ConfigString `yaml:"Style" default:"combat-rounds"` // Which combat module to use

	// Twitch combat settings for players
	TwitchUnarmedBaseMs     ConfigInt `yaml:"TwitchUnarmedBaseMs"`     // Base cooldown for unarmed combat in milliseconds
	TwitchArmedBaseMs       ConfigInt `yaml:"TwitchArmedBaseMs"`       // Base cooldown for armed combat in milliseconds
	TwitchMaxSpeedReduction ConfigInt `yaml:"TwitchMaxSpeedReduction"` // Maximum speed reduction percentage (0-100)
	TwitchSpeedDivisor      ConfigInt `yaml:"TwitchSpeedDivisor"`      // Divisor for speed bonus calculation

	// Twitch combat settings for mobs
	TwitchMobUnarmedBaseMs     ConfigInt `yaml:"TwitchMobUnarmedBaseMs"`     // Base cooldown for mob unarmed combat in milliseconds
	TwitchMobArmedBaseMs       ConfigInt `yaml:"TwitchMobArmedBaseMs"`       // Base cooldown for mob armed combat in milliseconds
	TwitchMobMaxSpeedReduction ConfigInt `yaml:"TwitchMobMaxSpeedReduction"` // Maximum speed reduction percentage for mobs (0-100)
	TwitchMobSpeedDivisor      ConfigInt `yaml:"TwitchMobSpeedDivisor"`      // Divisor for mob speed bonus calculation
}

type GameplayDeath struct {
	EquipmentDropChance ConfigFloat  `yaml:"EquipmentDropChance"` // Chance a player will drop a given piece of equipment on death
	AlwaysDropBackpack  ConfigBool   `yaml:"AlwaysDropBackpack"`  // If true, players will always drop their backpack items on death
	XPPenalty           ConfigString `yaml:"XPPenalty"`           // Possible values are: none, level, 10%, 25%, 50%, 75%, 90%, 100%
	ProtectionLevels    ConfigInt    `yaml:"ProtectionLevels"`    // How many levels is the user protected from death penalties for?
	PermaDeath          ConfigBool   `yaml:"PermaDeath"`          // Is permadeath enabled?
	CorpsesEnabled      ConfigBool   `yaml:"CorpsesEnabled"`      // Whether corpses are left behind after mob/player deaths
	CorpseDecayTime     ConfigString `yaml:"CorpseDecayTime"`     // How long until corpses decay to dust (go away)
}

func (g *GamePlay) Validate() {

	// Ignore AllowItemBuffRemoval
	// Ignore OnDeathAlwaysDropBackpack
	// Ignore ConsistentAttackMessages
	// Ignore CorpsesEnabled

	// Default combat style if not specified
	if g.Combat.Style == "" {
		g.Combat.Style = "combat-rounds"
	}

	if g.Death.EquipmentDropChance < 0.0 || g.Death.EquipmentDropChance > 1.0 {
		g.Death.EquipmentDropChance = 0.0 // default
	}

	g.Death.XPPenalty.Set(strings.ToLower(string(g.Death.XPPenalty)))

	if g.Death.XPPenalty != `none` && g.Death.XPPenalty != `level` {
		// If not a valid percent, set to default
		if !strings.HasSuffix(string(g.Death.XPPenalty), `%`) {
			g.Death.XPPenalty = `none` // default
		} else {
			// If not a valid percent, set to default
			percent, err := strconv.ParseInt(string(g.Death.XPPenalty)[0:len(g.Death.XPPenalty)-1], 10, 64)
			if err != nil || percent < 0 || percent > 100 {
				g.Death.XPPenalty = `none` // default
			}
		}
	}

	if g.Death.ProtectionLevels < 0 {
		g.Death.ProtectionLevels = 0 // default
	}

	if g.LivesStart < 0 {
		g.LivesStart = 0
	}

	if g.LivesMax < 0 {
		g.LivesMax = 0
	}

	if g.LivesOnLevelUp < 0 {
		g.LivesOnLevelUp = 0
	}

	if g.PricePerLife < 1 {
		g.PricePerLife = 1
	}

	if g.ShopRestockRate == `` {
		g.ShopRestockRate = `6 hours`
	}

	if g.ContainerSizeMax < 1 {
		g.ContainerSizeMax = 1
	}

	if g.MaxAltCharacters < 0 {
		g.MaxAltCharacters = 0
	}

	if g.Death.CorpseDecayTime == `` {
		g.Death.CorpseDecayTime = `1 hour`
	}

	if g.PVP != PVPEnabled && g.PVP != PVPDisabled && g.PVP != PVPLimited {
		if g.PVP == PVPOff {
			g.PVP = PVPDisabled
		} else {
			g.PVP = PVPEnabled
		}
	}

	if int(g.PVPMinimumLevel) < 0 {
		g.PVPMinimumLevel = 0
	}

	if g.XPScale <= 0 {
		g.XPScale = 100
	}

	if g.MobConverseChance < 0 {
		g.MobConverseChance = 0
	} else if g.MobConverseChance > 100 {
		g.MobConverseChance = 100
	}

	// Validate combat settings
	if g.Combat.TwitchUnarmedBaseMs == 0 {
		g.Combat.TwitchUnarmedBaseMs = 2500 // Default 2.5 seconds
	} else if g.Combat.TwitchUnarmedBaseMs < 100 || g.Combat.TwitchUnarmedBaseMs > 10000 {
		g.Combat.TwitchUnarmedBaseMs = 2500
	}

	if g.Combat.TwitchArmedBaseMs == 0 {
		g.Combat.TwitchArmedBaseMs = 2000 // Default 2 seconds
	} else if g.Combat.TwitchArmedBaseMs < 100 || g.Combat.TwitchArmedBaseMs > 10000 {
		g.Combat.TwitchArmedBaseMs = 2000
	}

	if g.Combat.TwitchMaxSpeedReduction == 0 {
		g.Combat.TwitchMaxSpeedReduction = 50 // Default 50%
	} else if g.Combat.TwitchMaxSpeedReduction < 0 || g.Combat.TwitchMaxSpeedReduction > 100 {
		g.Combat.TwitchMaxSpeedReduction = 50
	}

	if g.Combat.TwitchSpeedDivisor == 0 {
		g.Combat.TwitchSpeedDivisor = 200 // Default divisor
	} else if g.Combat.TwitchSpeedDivisor < 1 || g.Combat.TwitchSpeedDivisor > 1000 {
		g.Combat.TwitchSpeedDivisor = 200
	}

	// Validate mob combat settings
	if g.Combat.TwitchMobUnarmedBaseMs == 0 {
		g.Combat.TwitchMobUnarmedBaseMs = 2500 // Default 2.5 seconds
	} else if g.Combat.TwitchMobUnarmedBaseMs < 100 || g.Combat.TwitchMobUnarmedBaseMs > 10000 {
		g.Combat.TwitchMobUnarmedBaseMs = 2500
	}

	if g.Combat.TwitchMobArmedBaseMs == 0 {
		g.Combat.TwitchMobArmedBaseMs = 2000 // Default 2 seconds
	} else if g.Combat.TwitchMobArmedBaseMs < 100 || g.Combat.TwitchMobArmedBaseMs > 10000 {
		g.Combat.TwitchMobArmedBaseMs = 2000
	}

	if g.Combat.TwitchMobMaxSpeedReduction == 0 {
		g.Combat.TwitchMobMaxSpeedReduction = 50 // Default 50%
	} else if g.Combat.TwitchMobMaxSpeedReduction < 0 || g.Combat.TwitchMobMaxSpeedReduction > 100 {
		g.Combat.TwitchMobMaxSpeedReduction = 50
	}

	if g.Combat.TwitchMobSpeedDivisor == 0 {
		g.Combat.TwitchMobSpeedDivisor = 200 // Default divisor
	} else if g.Combat.TwitchMobSpeedDivisor < 1 || g.Combat.TwitchMobSpeedDivisor > 1000 {
		g.Combat.TwitchMobSpeedDivisor = 200
	}

}

func GetGamePlayConfig() GamePlay {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay
}
