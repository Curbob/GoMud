package combat

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type SourceTarget string

const (
	User SourceTarget = "user"
	Mob  SourceTarget = "mob"
)

// executeAttack executes a combat attack between any two combatants
func executeAttack(sourceActor, targetActor combatActor, sourceType, targetType SourceTarget) AttackResult {
	// Get characters from actors
	sourceChar := sourceActor.getCharacter()
	targetChar := targetActor.getCharacter()

	// Calculate combat result
	attackResult := calculateCombat(*sourceChar, *targetChar, sourceType, targetType)

	// Apply damage to source if any
	if attackResult.DamageToSource != 0 {
		sourceActor.applyDamage(attackResult.DamageToSource)
	}

	// Apply damage to target
	if attackResult.DamageToTarget != 0 {
		targetActor.applyDamage(attackResult.DamageToTarget)
	}

	// Track damage for mobs
	targetActor.trackDamage(sourceActor, attackResult.DamageToTarget)

	// Play sounds for users
	sourceActor.playAttackSound(attackResult.Hit, true)
	targetActor.playAttackSound(attackResult.Hit, false)

	return attackResult
}

// combatActor interface for users and mobs in combat
type combatActor interface {
	getCharacter() *characters.Character
	applyDamage(damage int)
	trackDamage(attacker combatActor, damage int)
	playAttackSound(hit bool, isAttacker bool)
}

// userCombatActor wraps a user for combat
type userCombatActor struct {
	user *users.UserRecord
}

func (u userCombatActor) getCharacter() *characters.Character {
	return u.user.Character
}

func (u userCombatActor) applyDamage(damage int) {
	u.user.Character.ApplyHealthChange(damage*-1, u.user.UserId)
	u.user.WimpyCheck()
}

func (u userCombatActor) trackDamage(attacker combatActor, damage int) {
	// Users don't track damage
}

func (u userCombatActor) playAttackSound(hit bool, isAttacker bool) {
	if isAttacker {
		if hit {
			u.user.PlaySound(`hit-other`, `combat`)
		} else {
			u.user.PlaySound(`miss`, `combat`)
		}
	} else if hit {
		u.user.PlaySound(`hit-self`, `combat`)
	}
}

// mobCombatActor wraps a mob for combat
type mobCombatActor struct {
	mob *mobs.Mob
}

func (m mobCombatActor) getCharacter() *characters.Character {
	return &m.mob.Character
}

func (m mobCombatActor) applyDamage(damage int) {
	m.mob.Character.ApplyHealthChange(damage * -1)
}

func (m mobCombatActor) trackDamage(attacker combatActor, damage int) {
	// Track player damage
	if userActor, ok := attacker.(userCombatActor); ok {
		m.mob.Character.TrackPlayerDamage(userActor.user.UserId, damage)
	} else if mobActor, ok := attacker.(mobCombatActor); ok {
		// If attacking mob was player charmed, attribute damage to that player
		if charmedUserId := mobActor.mob.Character.GetCharmedUserId(); charmedUserId > 0 {
			m.mob.Character.TrackPlayerDamage(charmedUserId, damage)
		}
	}
}

func (m mobCombatActor) playAttackSound(hit bool, isAttacker bool) {
	// Mobs don't play sounds
}

// Legacy wrapper functions for backward compatibility
func attackPlayerVsMob(user *users.UserRecord, mob *mobs.Mob) AttackResult {
	return executeAttack(userCombatActor{user}, mobCombatActor{mob}, User, Mob)
}

func attackPlayerVsPlayer(userAtk *users.UserRecord, userDef *users.UserRecord) AttackResult {
	return executeAttack(userCombatActor{userAtk}, userCombatActor{userDef}, User, User)
}

func attackMobVsPlayer(mob *mobs.Mob, user *users.UserRecord) AttackResult {
	return executeAttack(mobCombatActor{mob}, userCombatActor{user}, Mob, User)
}

func attackMobVsMob(mobAtk *mobs.Mob, mobDef *mobs.Mob) AttackResult {
	return executeAttack(mobCombatActor{mobAtk}, mobCombatActor{mobDef}, Mob, Mob)
}

// createCombatTokens creates a map of token replacements for combat messages
func createCombatTokens(sourceChar *characters.Character, targetChar *characters.Character, sourceType SourceTarget, targetType SourceTarget, damage string) map[items.TokenName]string {
	weaponName := races.GetRace(sourceChar.RaceId).UnarmedName
	if sourceChar.Equipment.Weapon.ItemId > 0 {
		weaponName = sourceChar.Equipment.Weapon.DisplayName()
	}

	sourceName := sourceChar.Name
	if sourceType == Mob {
		sourceName = sourceChar.GetMobName(0).String()
	}

	targetName := targetChar.Name
	if targetType == Mob {
		targetName = targetChar.GetMobName(0).String()
	}

	tokens := map[items.TokenName]string{
		items.TokenItemName:     weaponName,
		items.TokenSource:       sourceName,
		items.TokenSourceType:   string(sourceType) + `name`,
		items.TokenTarget:       targetName,
		items.TokenTargetType:   string(targetType) + `name`,
		items.TokenUsesLeft:     `[Invalid]`,
		items.TokenDamage:       damage,
		items.TokenEntranceName: `unknown`,
		items.TokenExitName:     `unknown`,
	}

	// Find exit names if in different rooms
	if sourceChar.RoomId != targetChar.RoomId {
		if atkRoom := rooms.LoadRoom(sourceChar.RoomId); atkRoom != nil {
			for exitName, exit := range atkRoom.Exits {
				if exit.RoomId == targetChar.RoomId {
					tokens[items.TokenExitName] = exitName
					break
				}
			}
		}
		if defRoom := rooms.LoadRoom(targetChar.RoomId); defRoom != nil {
			for exitName, exit := range defRoom.Exits {
				if exit.RoomId == sourceChar.RoomId {
					tokens[items.TokenEntranceName] = exitName
					break
				}
			}
		}
	}

	return tokens
}

// applyTokensToMessages applies token replacements to combat messages
func applyTokensToMessages(tokens map[items.TokenName]string, messages ...items.ItemMessage) []items.ItemMessage {
	result := make([]items.ItemMessage, len(messages))
	for i, msg := range messages {
		for tokenName, tokenValue := range tokens {
			msg = msg.SetTokenValue(tokenName, tokenValue)
		}
		result[i] = msg
	}
	return result
}

func GetWaitMessages(stepType items.Intensity, sourceChar *characters.Character, targetChar *characters.Character, sourceType SourceTarget, targetType SourceTarget) AttackResult {

	attackResult := AttackResult{}

	msgs := items.GetPreAttackMessage(sourceChar.Equipment.Weapon.GetSpec().Subtype, stepType)

	var toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg items.ItemMessage

	// zero means randomly selected, otherwise use the ItemId to consistently choose a message
	msgSeed := 0
	if configs.GetGamePlayConfig().ConsistentAttackMessages {
		msgSeed = sourceChar.Equipment.Weapon.ItemId
	}

	if sourceChar.RoomId == targetChar.RoomId {
		toAttackerMsg = msgs.Together.ToAttacker.Get(msgSeed)
		toDefenderMsg = msgs.Together.ToDefender.Get(msgSeed)
		toAttackerRoomMsg = msgs.Together.ToRoom.Get(msgSeed)
		toDefenderRoomMsg = items.ItemMessage("")
	} else {
		toAttackerMsg = msgs.Separate.ToAttacker.Get(msgSeed)
		toDefenderMsg = msgs.Separate.ToDefender.Get(msgSeed)
		toAttackerRoomMsg = msgs.Separate.ToAttackerRoom.Get(msgSeed)
		toDefenderRoomMsg = msgs.Separate.ToDefenderRoom.Get(msgSeed)
	}

	// Create tokens and apply them
	tokens := createCombatTokens(sourceChar, targetChar, sourceType, targetType, `[Invalid]`)
	appliedMsgs := applyTokensToMessages(tokens, toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg)

	if len(appliedMsgs) > 0 && string(appliedMsgs[0]) != `` {
		attackResult.SendToSource(string(appliedMsgs[0]))
	}

	if !sourceChar.HasBuffFlag(buffs.Hidden) {

		if len(appliedMsgs) > 1 && string(appliedMsgs[1]) != `` {
			attackResult.SendToTarget(string(appliedMsgs[1]))
		}

		if len(appliedMsgs) > 2 && string(appliedMsgs[2]) != `` {
			attackResult.SendToSourceRoom(string(appliedMsgs[2]))
		}

		if sourceChar.RoomId != targetChar.RoomId {
			if len(appliedMsgs) > 3 && string(appliedMsgs[3]) != `` {
				attackResult.SendToTargetRoom(string(appliedMsgs[3]))
			}
		}

	}

	return attackResult
}

func calculateCombat(sourceChar characters.Character, targetChar characters.Character, sourceType SourceTarget, targetType SourceTarget) AttackResult {

	attackResult := AttackResult{}

	attackCount := int(math.Ceil(float64(sourceChar.Stats.Speed.ValueAdj-targetChar.Stats.Speed.ValueAdj) / 25))
	if attackCount < 1 {
		attackCount = 1
	}

	// Statmods can add a damage bonus...
	statModDBonus := sourceChar.StatMod(`damage`)
	// Add any additional attacks
	attackCount += sourceChar.StatMod(`attacks`)

	for i := 0; i < attackCount; i++ {

		if mudlog.IsInitialized() {
			mudlog.Debug(`calculateCombat`, `Atk`, fmt.Sprintf(`%d/%d`, i+1, attackCount), `Source`, fmt.Sprintf(`%s (%s)`, sourceChar.Name, sourceType), `Target`, fmt.Sprintf(`%s (%s)`, targetChar.Name, targetType))
		}

		attackWeapons := []items.Item{}

		dualWieldLevel := sourceChar.GetSkillLevel(skills.DualWield)

		if sourceChar.Equipment.Weapon.ItemId > 0 {
			attackWeapons = append(attackWeapons, sourceChar.Equipment.Weapon)
		}

		if sourceChar.Equipment.Offhand.ItemId > 0 && sourceChar.Equipment.Offhand.GetSpec().Type == items.Weapon {
			attackWeapons = append(attackWeapons, sourceChar.Equipment.Offhand)
		}

		// Put an empty weapon, so basically hands.
		if len(attackWeapons) == 0 {
			attackWeapons = append(attackWeapons, items.Item{
				ItemId: 0,
			})
		}

		if len(attackWeapons) > 1 {

			maxWeapons := 1
			if dualWieldLevel == 1 {
				maxWeapons = 1
			}

			if dualWieldLevel == 2 {

				roll := util.Rand(100)

				util.LogRoll(`Both Weapons`, roll, 50)

				if roll < 50 {
					maxWeapons = 2
				}
			}

			if dualWieldLevel >= 3 {
				maxWeapons = 2
			}

			// If two martial weapons are equipped, allow dual wielding even without the stat.
			if sourceChar.Equipment.Weapon.GetSpec().Subtype == items.Claws && sourceChar.Equipment.Offhand.GetSpec().Subtype == items.Claws {
				maxWeapons = 2
			}

			for len(attackWeapons) > maxWeapons {
				// Remove a random position
				rnd := util.Rand(len(attackWeapons))
				attackWeapons = append(attackWeapons[:rnd], attackWeapons[rnd+1:]...)
			}

		}

		attackMessagePrefix := ``
		// If they are backstabbing it's a free crit
		if sourceChar.Aggro != nil && sourceChar.Aggro.Type == characters.BackStab {
			attackResult.Crit = true
			attackMessagePrefix = `<ansi fg="magenta-bold">*[BACKSTAB]*</ansi> `
			// Failover to the default attack
			sourceChar.SetAggro(sourceChar.Aggro.UserId, sourceChar.Aggro.MobInstanceId, characters.DefaultAttack)
		}

		for _, weapon := range attackWeapons {

			penalty := 0
			if len(attackWeapons) > 1 {
				if dualWieldLevel < 4 {
					penalty = 35 //35% penalty to hit
				} else {
					penalty = 25 //25% penalty to hit
				}
			}

			// Set the default weapon info
			raceInfo := races.GetRace(sourceChar.RaceId)
			weaponName := raceInfo.UnarmedName
			weaponSubType := items.Generic

			// Get default racial dice rolls
			attacks, dCount, dSides, dBonus, critBuffs := sourceChar.GetDefaultDiceRoll()

			// Add damage bonus due to statmods
			dBonus += statModDBonus

			if weapon.ItemId > 0 {

				itemSpec := weapon.GetSpec()

				weaponName = weapon.DisplayName()

				weaponSubType = itemSpec.Subtype
				attacks, dCount, dSides, dBonus, critBuffs = weapon.GetDiceRoll()

				// If there is a bonus vs. a specific race, apply it
				dBonus += weapon.StatMod(string(statmods.RacialBonusPrefix) + strings.ToLower(targetChar.Race()))
			}

			// zero means randomly selected, otherwise use the ItemId to consistently choose a message
			msgSeed := 0
			if configs.GetGamePlayConfig().ConsistentAttackMessages {
				msgSeed = weapon.ItemId
			}

			if mudlog.IsInitialized() {
				mudlog.Debug("DiceRolls", "attacks", attacks, "dCount", dCount, "dSides", dSides, "dBonus", dBonus, "critBuffs", critBuffs)
			}

			// Individual weapons may get multiple attacks
			for j := 0; j < attacks; j++ {

				attackTargetDamage := 0
				attackTargetReduction := 0

				attackSourceDamage := 0
				attackSourceReduction := 0

				if Hits(sourceChar.Stats.Speed.ValueAdj, targetChar.Stats.Speed.ValueAdj, penalty) {
					attackResult.Hit = true
					attackTargetDamage = util.RollDice(dCount, dSides) + dBonus

					if attackResult.Crit || Crits(sourceChar, targetChar) {
						attackResult.Crit = true
						attackResult.BuffTarget = critBuffs
						attackTargetDamage += dCount*dSides + dBonus
					}
				}

				defenseAmt := util.Rand(targetChar.GetDefense())
				if defenseAmt > 0 {
					attackTargetReduction = int(math.Round((float64(defenseAmt) / 100) * float64(attackTargetDamage)))
					attackTargetDamage -= attackTargetReduction
				}

				defenseAmt = util.Rand(sourceChar.GetDefense())
				if defenseAmt > 0 {
					attackSourceReduction = int(math.Round((float64(defenseAmt) / 100) * float64(attackSourceDamage)))
					attackSourceDamage -= attackSourceReduction
				}

				// Calculate actual damage vs. possible damage pct
				pctDamage := math.Ceil(float64(attackTargetDamage) / float64(dCount*dSides+dBonus) * 100)

				msgs := items.GetAttackMessage(weaponSubType, int(pctDamage))

				var toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg items.ItemMessage

				// Create combat tokens
				// Note: weaponName is already set above
				tokenReplacements := createCombatTokens(&sourceChar, &targetChar, sourceType, targetType, strconv.Itoa(attackTargetDamage))
				tokenReplacements[items.TokenItemName] = weaponName // Override with the specific weapon used

				if sourceChar.RoomId == targetChar.RoomId {
					toAttackerMsg = msgs.Together.ToAttacker.Get(msgSeed)
					toDefenderMsg = msgs.Together.ToDefender.Get(msgSeed)
					toAttackerRoomMsg = msgs.Together.ToRoom.Get(msgSeed)
					toDefenderRoomMsg = items.ItemMessage("")
				} else {
					toAttackerMsg = msgs.Separate.ToAttacker.Get(msgSeed)
					toDefenderMsg = msgs.Separate.ToDefender.Get(msgSeed)
					toAttackerRoomMsg = msgs.Separate.ToAttackerRoom.Get(msgSeed)
					toDefenderRoomMsg = msgs.Separate.ToDefenderRoom.Get(msgSeed)
				}

				// Apply tokens to messages
				appliedMsgs := applyTokensToMessages(tokenReplacements, toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg)
				if len(appliedMsgs) >= 4 {
					toAttackerMsg = appliedMsgs[0]
					toDefenderMsg = appliedMsgs[1]
					toAttackerRoomMsg = appliedMsgs[2]
					toDefenderRoomMsg = appliedMsgs[3]
				}

				if attackResult.Crit {
					toAttackerMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					toDefenderMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					toAttackerRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					if len(string(toDefenderRoomMsg)) > 0 {
						toDefenderRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					}
				}

				if len(attackMessagePrefix) > 0 {
					toAttackerMsg = items.ItemMessage(attackMessagePrefix + string(toAttackerMsg))
					toDefenderMsg = items.ItemMessage(attackMessagePrefix + string(toDefenderMsg))
					toAttackerRoomMsg = items.ItemMessage(attackMessagePrefix + string(toAttackerRoomMsg))
					if len(string(toDefenderRoomMsg)) > 0 {
						toDefenderRoomMsg = items.ItemMessage(attackMessagePrefix + string(toDefenderRoomMsg))
					}
				}

				// Send to attacker
				attackerMsg := string(toAttackerMsg)
				if attackSourceDamage > 0 && attackSourceReduction > 0 {
					attackerMsg += fmt.Sprintf(` <ansi fg="white">[%d was blocked]</ansi>`, attackSourceReduction)
				}

				attackResult.SendToSource(
					string(attackerMsg),
				)

				// Send to victim
				defenderMsg := string(toDefenderMsg)
				if attackTargetDamage > 0 && attackTargetReduction > 0 {
					defenderMsg += fmt.Sprintf(` <ansi fg="red">[you blocked %d]</ansi>`, attackTargetReduction)
				}

				attackResult.SendToTarget(
					string(defenderMsg),
				)

				// Send to room
				attackResult.SendToSourceRoom(string(toAttackerRoomMsg))

				// Send to defender room if separate
				if len(string(toDefenderRoomMsg)) > 0 {
					attackResult.SendToTargetRoom(string(toDefenderRoomMsg))
				}

				attackResult.DamageToTarget += attackTargetDamage
				attackResult.DamageToTargetReduction += attackTargetReduction

				attackResult.DamageToSource += attackSourceDamage
				attackResult.DamageToSourceReduction += attackSourceReduction
			}

			if util.RollDice(1, 5) == 1 { // 20% chance to join
				if sourceChar.RoomId == targetChar.RoomId {
					if sourceChar.Pet.Exists() && sourceChar.Pet.Damage.DiceRoll != `` {

						attacks, dCount, dSides, dBonus, critBuffs = sourceChar.Pet.GetDiceRoll()

						for i := 0; i < attacks; i++ {

							attackTargetDamage := util.RollDice(dCount, dSides) + dBonus

							attackResult.DamageToTarget += attackTargetDamage

							toAttackerMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%d damage</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), attackTargetDamage, string(targetType), targetChar.Name)
							attackResult.SendToSource(toAttackerMsg)

							toDefenderMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%d damage</ansi> to you!`, sourceChar.Pet.DisplayName(), attackTargetDamage)
							attackResult.SendToTarget(toDefenderMsg)

							toAttackerRoomMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%d damage</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), attackTargetDamage, string(targetType), targetChar.Name)
							attackResult.SendToTargetRoom(toAttackerRoomMsg)

						}

					}
				}
			}

		}
	}
	return attackResult

}

// hit chance will be between 30 and 100
func hitChance(attackSpd, defendSpd int) int {
	atkPlusDef := float64(attackSpd + defendSpd)
	if atkPlusDef < 1 {
		atkPlusDef = 1
	}
	return 30 + int(float64(attackSpd)/atkPlusDef*70)
}

// Chance to hit
func Hits(attackSpd, defendSpd, hitModifier int) bool {
	// Attack speeds affect 90% of the hit chance
	toHit := hitChance(attackSpd, defendSpd)
	if hitModifier != 0 {
		toHit += hitModifier
	}

	// Always at leat a 5% chance
	if toHit < 5 {
		toHit = 5
	}

	// Always at most a 95% chance
	if toHit > 95 {
		toHit = 95
	}
	hitRoll := util.Rand(100)

	util.LogRoll(`Hits`, hitRoll, toHit)

	return hitRoll < toHit
}

// Whether they crit
func Crits(sourceChar characters.Character, targetChar characters.Character) bool {

	levelDiff := sourceChar.Level - targetChar.Level
	if levelDiff < 1 {
		levelDiff = 1
	}
	critChance := 5 + int(math.Round(float64(sourceChar.Stats.Strength.ValueAdj+sourceChar.Stats.Speed.ValueAdj)/float64(levelDiff)))

	if sourceChar.HasBuffFlag(buffs.Accuracy) {
		critChance *= 2
	}

	if targetChar.HasBuffFlag(buffs.Blink) {
		critChance /= 2
	}

	// Minimum 5% chance
	if critChance < 5 {
		critChance = 5
	}

	critRoll := util.Rand(100)

	util.LogRoll(`Crits`, critRoll, critChance)

	return critRoll < critChance
}
