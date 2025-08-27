package combat

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// These functions delegate to the active combat system
// They maintain the same signatures as the existing combat functions
// to ensure backward compatibility

// AttackPlayerVsMob delegates to the active combat system
func AttackPlayerVsMob(user *users.UserRecord, mob *mobs.Mob) AttackResult {
	// Validate inputs
	if user == nil || mob == nil {
		mudlog.Error("AttackPlayerVsMob", "error", "nil combatant", "user", user != nil, "mob", mob != nil)
		return AttackResult{
			Hit:              false,
			MessagesToSource: []string{"Combat error: invalid target"},
		}
	}

	system := GetActiveCombatSystem()
	if system == nil || system.GetCalculator() == nil {
		mudlog.Debug("AttackPlayerVsMob", "fallback", "no active system or calculator")
		// Fallback to existing implementation
		return attackPlayerVsMob(user, mob)
	}

	// Use the combat system's calculator
	attackerId := user.UserId
	attackerName := user.Character.Name
	defenderId := mob.InstanceId
	defenderName := util.StripANSI(mob.Character.Name)
	
	return performAttack(user.Character, &mob.Character, User, Mob, system.GetCalculator(),
		attackerId, attackerName, defenderId, defenderName,
		func(result *AttackResult) {
			if result.DamageToSource != 0 {
				user.Character.ApplyHealthChange(result.DamageToSource * -1)
				user.WimpyCheck()
			}

			mob.Character.ApplyHealthChange(result.DamageToTarget * -1)

			// Remember who has hit him
			mob.Character.TrackPlayerDamage(user.UserId, result.DamageToTarget)

			if result.Hit {
				user.PlaySound(`hit-other`, `combat`)
			} else {
				user.PlaySound(`miss`, `combat`)
			}
		})
}

// AttackPlayerVsPlayer delegates to the active combat system
func AttackPlayerVsPlayer(userAtk *users.UserRecord, userDef *users.UserRecord) AttackResult {
	// Validate inputs
	if userAtk == nil || userDef == nil {
		mudlog.Error("AttackPlayerVsPlayer", "error", "nil combatant", "attacker", userAtk != nil, "defender", userDef != nil)
		return AttackResult{
			Hit:              false,
			MessagesToSource: []string{"Combat error: invalid target"},
		}
	}

	system := GetActiveCombatSystem()
	if system == nil || system.GetCalculator() == nil {
		mudlog.Debug("AttackPlayerVsPlayer", "fallback", "no active system or calculator")
		// Fallback to existing implementation
		return attackPlayerVsPlayer(userAtk, userDef)
	}

	// Use the combat system's calculator
	attackerId := userAtk.UserId
	attackerName := userAtk.Character.Name
	defenderId := userDef.UserId
	defenderName := userDef.Character.Name
	
	return performAttack(userAtk.Character, userDef.Character, User, User, system.GetCalculator(),
		attackerId, attackerName, defenderId, defenderName,
		func(result *AttackResult) {
			if result.DamageToSource != 0 {
				userAtk.Character.ApplyHealthChange(result.DamageToSource * -1)
				userAtk.WimpyCheck()
			}

			if result.DamageToTarget != 0 {
				userDef.Character.ApplyHealthChange(result.DamageToTarget * -1)
				userDef.WimpyCheck()
			}

			if result.Hit {
				userAtk.PlaySound(`hit-other`, `combat`)
				userDef.PlaySound(`hit-self`, `combat`)
			} else {
				userAtk.PlaySound(`miss`, `combat`)
			}
		})
}

// AttackMobVsPlayer delegates to the active combat system
func AttackMobVsPlayer(mob *mobs.Mob, user *users.UserRecord) AttackResult {
	// Validate inputs
	if mob == nil || user == nil {
		mudlog.Error("AttackMobVsPlayer", "error", "nil combatant", "mob", mob != nil, "user", user != nil)
		return AttackResult{
			Hit:              false,
			MessagesToTarget: []string{"Combat error: invalid attacker"},
		}
	}

	system := GetActiveCombatSystem()
	if system == nil || system.GetCalculator() == nil {
		mudlog.Debug("AttackMobVsPlayer", "fallback", "no active system or calculator")
		// Fallback to existing implementation
		return attackMobVsPlayer(mob, user)
	}

	// Use the combat system's calculator
	attackerId := mob.InstanceId
	attackerName := util.StripANSI(mob.Character.Name)
	defenderId := user.UserId
	defenderName := user.Character.Name
	
	return performAttack(&mob.Character, user.Character, Mob, User, system.GetCalculator(),
		attackerId, attackerName, defenderId, defenderName,
		func(result *AttackResult) {
			mob.Character.ApplyHealthChange(result.DamageToSource * -1)

			if result.DamageToTarget != 0 {
				user.Character.ApplyHealthChange(result.DamageToTarget * -1)
				user.WimpyCheck()
			}

			if result.Hit {
				user.PlaySound(`hit-self`, `combat`)
			}
		})
}

// AttackMobVsMob delegates to the active combat system
func AttackMobVsMob(mobAtk *mobs.Mob, mobDef *mobs.Mob) AttackResult {
	// Validate inputs
	if mobAtk == nil || mobDef == nil {
		mudlog.Error("AttackMobVsMob", "error", "nil combatant", "mobAtk", mobAtk != nil, "mobDef", mobDef != nil)
		return AttackResult{
			Hit: false,
		}
	}

	system := GetActiveCombatSystem()
	if system == nil || system.GetCalculator() == nil {
		mudlog.Debug("AttackMobVsMob", "fallback", "no active system or calculator")
		// Fallback to existing implementation
		return attackMobVsMob(mobAtk, mobDef)
	}

	// Use the combat system's calculator
	attackerId := mobAtk.InstanceId
	attackerName := util.StripANSI(mobAtk.Character.Name)
	defenderId := mobDef.InstanceId
	defenderName := util.StripANSI(mobDef.Character.Name)
	
	return performAttack(&mobAtk.Character, &mobDef.Character, Mob, Mob, system.GetCalculator(),
		attackerId, attackerName, defenderId, defenderName,
		func(result *AttackResult) {
			mobAtk.Character.ApplyHealthChange(result.DamageToSource * -1)
			mobDef.Character.ApplyHealthChange(result.DamageToTarget * -1)

			// If attacking mob was player charmed, attribute damage done to that player
			if charmedUserId := mobAtk.Character.GetCharmedUserId(); charmedUserId > 0 {
				// Remember who has hit him
				mobDef.Character.TrackPlayerDamage(charmedUserId, result.DamageToTarget)
			}
		})
}

// performAttack handles a single attack using the combat calculator
func performAttack(attacker, defender *characters.Character, attackerType, defenderType SourceTarget,
	calculator ICombatCalculator, attackerId int, attackerName string, defenderId int, defenderName string,
	postAttack func(*AttackResult)) AttackResult {

	result := AttackResult{}

	// Validate inputs
	if attacker == nil || defender == nil || calculator == nil {
		mudlog.Error("performAttack", "error", "nil parameter",
			"attacker", attacker != nil,
			"defender", defender != nil,
			"calculator", calculator != nil)
		result.Hit = false
		result.SendToSource("Combat error: system failure")
		return result
	}

	// Calculate number of attacks
	attackCount := calculator.CalculateAttackCount(attacker, defender)
	if attackCount < 1 {
		mudlog.Warn("performAttack", "warning", "invalid attack count", "count", attackCount)
		attackCount = 1
	}

	for i := 0; i < attackCount; i++ {
		// Check if attack hits
		hit := calculator.CalculateHitChance(attacker, defender)
		if !hit {
			result.Hit = false
			// Generate miss messages
			generateMissMessages(&result, attacker, defender, attackerType, defenderType)
			
			// Fire AttackAvoided event for GMCP
			attackerTypeStr := "mob"
			if attackerType == User {
				attackerTypeStr = "player"
			}
			defenderTypeStr := "mob"
			if defenderType == User {
				defenderTypeStr = "player"
			}
			
			events.AddToQueue(events.AttackAvoided{
				AttackerId:   attackerId,
				AttackerType: attackerTypeStr,
				AttackerName: attackerName,
				DefenderId:   defenderId,
				DefenderType: defenderTypeStr,
				DefenderName: defenderName,
			})
			
			continue
		}

		result.Hit = true

		// Check for critical hit
		result.Crit = calculator.CalculateCriticalChance(attacker, defender)

		// Get weapon for damage calculation
		var weapon *items.Item
		if attacker.Equipment.Weapon.ItemId > 0 {
			weapon = &attacker.Equipment.Weapon
		}

		// Calculate damage (calculator already applies defense reduction)
		damage := calculator.CalculateDamage(attacker, defender, weapon)

		result.DamageToTarget += damage

		// Generate hit messages
		generateHitMessages(&result, attacker, defender, attackerType, defenderType, damage, result.Crit)
		
		// Fire DamageDealt event for GMCP
		if damage > 0 {
			attackerTypeStr := "mob"
			if attackerType == User {
				attackerTypeStr = "player"
			}
			defenderTypeStr := "mob"
			if defenderType == User {
				defenderTypeStr = "player"
			}
			
			events.AddToQueue(events.DamageDealt{
				SourceId:     attackerId,
				SourceType:   attackerTypeStr,
				SourceName:   attackerName,
				TargetId:     defenderId,
				TargetType:   defenderTypeStr,
				TargetName:   defenderName,
				Amount:       damage,
				DamageType:   "physical",
			})
		}

		// Check for buffs to apply
		checkCombatBuffs(&result, attacker, defender)

		// Check for counter-attack damage
		checkCounterDamage(&result, attacker, defender)
	}

	// Apply post-attack effects
	if postAttack != nil {
		postAttack(&result)
	}

	return result
}

// generateMissMessages creates appropriate miss messages
func generateMissMessages(result *AttackResult, attacker, defender *characters.Character,
	attackerType, defenderType SourceTarget) {

	attackerName := getActorName(attacker, attackerType)
	defenderName := getActorName(defender, defenderType)

	result.SendToSource(fmt.Sprintf(`You miss <ansi fg="%sname">%s</ansi>.`, defenderType, defenderName))
	result.SendToTarget(fmt.Sprintf(`<ansi fg="%sname">%s</ansi> misses you.`, attackerType, attackerName))
	result.SendToSourceRoom(fmt.Sprintf(`<ansi fg="%sname">%s</ansi> misses <ansi fg="%sname">%s</ansi>.`,
		attackerType, attackerName, defenderType, defenderName))
}

// generateHitMessages creates appropriate hit messages
func generateHitMessages(result *AttackResult, attacker, defender *characters.Character,
	attackerType, defenderType SourceTarget, damage int, crit bool) {

	attackerName := getActorName(attacker, attackerType)
	defenderName := getActorName(defender, defenderType)

	critText := ""
	if crit {
		critText = " <ansi fg=\"red\">CRITICALLY</ansi>"
	}

	weaponName := "fists"
	if attacker.Equipment.Weapon.ItemId > 0 {
		weaponName = attacker.Equipment.Weapon.DisplayName()
	} else if attacker.RaceId > 0 {
		weaponName = races.GetRace(attacker.RaceId).UnarmedName
	}

	result.SendToSource(fmt.Sprintf(`You%s hit <ansi fg="%sname">%s</ansi> with your <ansi fg="item">%s</ansi> for <ansi fg="damage">%d damage</ansi>.`,
		critText, defenderType, defenderName, weaponName, damage))
	result.SendToTarget(fmt.Sprintf(`<ansi fg="%sname">%s</ansi>%s hits you with <ansi fg="item">%s</ansi> for <ansi fg="damage">%d damage</ansi>.`,
		attackerType, attackerName, critText, weaponName, damage))
	result.SendToSourceRoom(fmt.Sprintf(`<ansi fg="%sname">%s</ansi>%s hits <ansi fg="%sname">%s</ansi> with <ansi fg="item">%s</ansi>.`,
		attackerType, attackerName, critText, defenderType, defenderName, weaponName))
}

// getActorName returns the appropriate name for an actor
func getActorName(char *characters.Character, actorType SourceTarget) string {
	if actorType == Mob {
		return char.GetMobName(0).String()
	}
	return char.Name
}

// checkCombatBuffs checks for buffs that should be applied during combat
func checkCombatBuffs(result *AttackResult, attacker, defender *characters.Character) {
	// Check weapon procs
	if attacker.Equipment.Weapon.ItemId > 0 {
		spec := attacker.Equipment.Weapon.GetSpec()
		for _, buffId := range spec.BuffIds {
			if util.Rand(100) < 10 { // 10% chance
				result.BuffTarget = append(result.BuffTarget, buffId)
			}
		}
	}

	// Future: Check defensive buffs here
}

// checkCounterDamage checks for damage reflection or counter-attacks
func checkCounterDamage(result *AttackResult, attacker, defender *characters.Character) {
	// Future: Add counter-damage mechanics here
	// For example: thorns damage, riposte, etc.
}
