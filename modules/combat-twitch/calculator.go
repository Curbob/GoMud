package combattwitch

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TwitchCalculator implements combat calculations for twitch-based combat
type TwitchCalculator struct{}

// NewTwitchCalculator creates a new calculator instance
func NewTwitchCalculator() *TwitchCalculator {
	return &TwitchCalculator{}
}

// CalculateHitChance determines if an attack hits
func (calc *TwitchCalculator) CalculateHitChance(attacker, defender *characters.Character) bool {
	// Twitch combat might have different hit calculations
	// For example, faster weapons might be more accurate
	hitBonus := 0
	if attacker.Equipment.Weapon.ItemId > 0 {
		// Lighter weapons get hit bonus
		weaponSpec := attacker.Equipment.Weapon.GetSpec()
		if weaponSpec.Hands == 1 {
			hitBonus = 10
		}
	}

	return combat.Hits(
		attacker.Stats.Speed.ValueAdj+hitBonus,
		defender.Stats.Speed.ValueAdj,
		0,
	)
}

// CalculateDamage computes damage for an attack
func (calc *TwitchCalculator) CalculateDamage(attacker, defender *characters.Character, weapon *items.Item) int {
	damage := 0

	if weapon != nil && weapon.GetSpec().Damage.DiceRoll != "" {
		// Roll weapon damage dice
		_, dCount, dSides, bonus, _ := util.ParseDiceRoll(weapon.GetSpec().Damage.DiceRoll)
		damage = util.RollDice(dCount, dSides) + bonus

		// Twitch combat might reduce damage for faster weapons
		if weapon.GetSpec().WaitRounds < 2 {
			damage = int(float64(damage) * 0.8) // 20% less damage for fast weapons
		}
	} else {
		// Unarmed damage
		damage = util.RollDice(1, 2) + attacker.Stats.Strength.ValueAdj/15
	}

	// Apply stat modifiers (less impactful in twitch combat)
	damage += int(math.Round(float64(attacker.Stats.Strength.ValueAdj) / 30))

	// Apply defense reduction
	defense := calc.CalculateDefense(defender)
	damage -= defense

	if damage < 1 {
		damage = 1 // Minimum 1 damage
	}

	return damage
}

// CalculateCriticalChance determines if an attack is critical
func (calc *TwitchCalculator) CalculateCriticalChance(attacker, defender *characters.Character) bool {
	// In twitch combat, perception matters more for crits
	critBonus := attacker.Stats.Perception.ValueAdj / 10

	// Fast weapons crit more often
	if attacker.Equipment.Weapon.ItemId > 0 {
		weaponSpec := attacker.Equipment.Weapon.GetSpec()
		if weaponSpec.WaitRounds < 2 {
			critBonus += 5
		}
	}

	baseChance := 5 + critBonus
	return util.Rand(100) < baseChance
}

// CalculateDefense computes defensive value
func (calc *TwitchCalculator) CalculateDefense(defender *characters.Character) int {
	// Base defense from vitality (less effective in twitch combat)
	defense := defender.Stats.Vitality.ValueAdj / 15

	// Armor is less effective in twitch combat
	armorValue := 0
	if defender.Equipment.Head.ItemId > 0 {
		armorValue += defender.Equipment.Head.GetSpec().DamageReduction
	}
	if defender.Equipment.Body.ItemId > 0 {
		armorValue += defender.Equipment.Body.GetSpec().DamageReduction
	}
	if defender.Equipment.Legs.ItemId > 0 {
		armorValue += defender.Equipment.Legs.GetSpec().DamageReduction
	}
	if defender.Equipment.Feet.ItemId > 0 {
		armorValue += defender.Equipment.Feet.GetSpec().DamageReduction
	}
	if defender.Equipment.Offhand.ItemId > 0 {
		armorValue += defender.Equipment.Offhand.GetSpec().DamageReduction
	}

	// Armor is 50% as effective in twitch combat
	defense += armorValue / 2

	return defense
}

// CalculateInitiative determines action order (not used in twitch combat)
func (calc *TwitchCalculator) CalculateInitiative(actor *characters.Character) int {
	// In twitch combat, initiative is based on cooldowns, not rounds
	return 0
}

// CalculateAttackCount determines number of attacks (not used in twitch combat)
func (calc *TwitchCalculator) CalculateAttackCount(attacker, defender *characters.Character) int {
	// In twitch combat, attack count is determined by cooldowns
	return 1
}

// PowerRanking calculates relative power between actors
func (calc *TwitchCalculator) PowerRanking(attacker, defender *characters.Character) float64 {
	// In twitch combat, weapon speed is more important
	attacks, dCount, dSides, dBonus, _ := attacker.Equipment.Weapon.GetDiceRoll()
	atkDmg := attacks * (dCount*dSides + dBonus)

	// Adjust damage based on weapon speed
	if attacker.Equipment.Weapon.ItemId > 0 {
		waitRounds := attacker.Equipment.Weapon.GetSpec().WaitRounds
		if waitRounds == 0 {
			atkDmg = int(float64(atkDmg) * 1.5) // Fast weapons get bonus
		}
	}

	attacks, dCount, dSides, dBonus, _ = defender.Equipment.Weapon.GetDiceRoll()
	defDmg := attacks * (dCount*dSides + dBonus)

	pct := 0.0
	if defDmg == 0 {
		pct += 0.3 // Weapon damage matters less in twitch combat
	} else {
		pct += 0.3 * float64(atkDmg) / float64(defDmg)
	}

	// Speed matters more in twitch combat
	if defender.Stats.Speed.ValueAdj == 0 {
		pct += 0.4
	} else {
		pct += 0.4 * float64(attacker.Stats.Speed.ValueAdj) / float64(defender.Stats.Speed.ValueAdj)
	}

	if defender.HealthMax.Value == 0 {
		pct += 0.2
	} else {
		pct += 0.2 * float64(attacker.HealthMax.Value) / float64(defender.HealthMax.Value)
	}

	// Defense matters less in twitch combat
	if defender.GetDefense() == 0 {
		pct += 0.1
	} else {
		pct += 0.1 * float64(attacker.GetDefense()) / float64(defender.GetDefense())
	}

	return pct
}
