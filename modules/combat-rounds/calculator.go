package combatrounds

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// RoundBasedCalculator implements combat calculations for round-based combat
type RoundBasedCalculator struct{}

// NewRoundBasedCalculator creates a new calculator instance
func NewRoundBasedCalculator() *RoundBasedCalculator {
	return &RoundBasedCalculator{}
}

// CalculateHitChance determines if an attack hits
func (calc *RoundBasedCalculator) CalculateHitChance(attacker, defender *characters.Character) bool {
	// Use the existing Hits function from combat package
	return combat.Hits(
		attacker.Stats.Speed.ValueAdj,
		defender.Stats.Speed.ValueAdj,
		0, // hitModifier - can be enhanced later
	)
}

// CalculateDamage computes damage for an attack
func (calc *RoundBasedCalculator) CalculateDamage(attacker, defender *characters.Character, weapon *items.Item) int {
	damage := 0

	if weapon != nil && weapon.GetSpec().Damage.DiceRoll != "" {
		// Roll weapon damage dice
		_, dCount, dSides, bonus, _ := util.ParseDiceRoll(weapon.GetSpec().Damage.DiceRoll)
		damage = util.RollDice(dCount, dSides) + bonus
	} else {
		// Unarmed damage based on strength
		strMod := int(math.Round(float64(attacker.Stats.Strength.ValueAdj) / 10))
		damage = util.RollDice(1, 3) + strMod
	}

	// Apply stat modifiers
	damage += int(math.Round(float64(attacker.Stats.Strength.ValueAdj) / 20))

	// Apply defense reduction
	defense := calc.CalculateDefense(defender)
	damage -= defense

	if damage < 1 {
		damage = 1 // Minimum 1 damage
	}

	return damage
}

// CalculateCriticalChance determines if an attack is critical
func (calc *RoundBasedCalculator) CalculateCriticalChance(attacker, defender *characters.Character) bool {
	// Use the existing Crits function from combat package
	return combat.Crits(*attacker, *defender)
}

// CalculateDefense computes defensive value
func (calc *RoundBasedCalculator) CalculateDefense(defender *characters.Character) int {
	// Base defense from vitality
	defense := defender.Stats.Vitality.ValueAdj / 10

	// Add armor contributions (using DamageReduction as armor value)
	if defender.Equipment.Head.ItemId > 0 {
		defense += defender.Equipment.Head.GetSpec().DamageReduction
	}
	if defender.Equipment.Body.ItemId > 0 {
		defense += defender.Equipment.Body.GetSpec().DamageReduction
	}
	if defender.Equipment.Legs.ItemId > 0 {
		defense += defender.Equipment.Legs.GetSpec().DamageReduction
	}
	if defender.Equipment.Feet.ItemId > 0 {
		defense += defender.Equipment.Feet.GetSpec().DamageReduction
	}
	if defender.Equipment.Offhand.ItemId > 0 {
		defense += defender.Equipment.Offhand.GetSpec().DamageReduction
	}

	return defense
}

// CalculateInitiative determines action order
func (calc *RoundBasedCalculator) CalculateInitiative(actor *characters.Character) int {
	// In round-based combat, initiative is based on speed
	return actor.Stats.Speed.ValueAdj
}

// CalculateAttackCount determines number of attacks per round
func (calc *RoundBasedCalculator) CalculateAttackCount(attacker, defender *characters.Character) int {
	// Based on speed differential
	speedDiff := attacker.Stats.Speed.ValueAdj - defender.Stats.Speed.ValueAdj
	attackCount := int(math.Ceil(float64(speedDiff) / 25))

	if attackCount < 1 {
		attackCount = 1
	} else if attackCount > 5 {
		attackCount = 5 // Cap at 5 attacks
	}

	return attackCount
}

// PowerRanking calculates relative power between actors
func (calc *RoundBasedCalculator) PowerRanking(attacker, defender *characters.Character) float64 {
	attacks, dCount, dSides, dBonus, _ := attacker.Equipment.Weapon.GetDiceRoll()
	atkDmg := attacks * (dCount*dSides + dBonus)

	attacks, dCount, dSides, dBonus, _ = defender.Equipment.Weapon.GetDiceRoll()
	defDmg := attacks * (dCount*dSides + dBonus)

	pct := 0.0
	if defDmg == 0 {
		pct += 0.4
	} else {
		pct += 0.4 * float64(atkDmg) / float64(defDmg)
	}

	if defender.Stats.Speed.ValueAdj == 0 {
		pct += 0.3
	} else {
		pct += 0.3 * float64(attacker.Stats.Speed.ValueAdj) / float64(defender.Stats.Speed.ValueAdj)
	}

	if defender.HealthMax.Value == 0 {
		pct += 0.2
	} else {
		pct += 0.2 * float64(attacker.HealthMax.Value) / float64(defender.HealthMax.Value)
	}

	if defender.GetDefense() == 0 {
		pct += 0.1
	} else {
		pct += 0.1 * float64(attacker.GetDefense()) / float64(defender.GetDefense())
	}

	return pct
}
