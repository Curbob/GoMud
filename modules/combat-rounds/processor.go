package combatrounds

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ProcessCombatRound handles a complete round of combat for all actors
func (rbc *RoundBasedCombat) processCombatRound(roundNumber uint64) {
	if !rbc.active {
		return
	}

	// Process player combat
	affectedPlayers1, affectedMobs1 := rbc.processPlayerCombat(roundNumber)

	// Process mob combat
	affectedPlayers2, affectedMobs2 := rbc.processMobCombat(roundNumber)

	// Handle affected actors (deaths, etc)
	rbc.handleAffected(append(affectedPlayers1, affectedPlayers2...), append(affectedMobs1, affectedMobs2...))
}

// processPlayerCombat handles combat for all players
func (rbc *RoundBasedCombat) processPlayerCombat(roundNumber uint64) (affectedPlayerIds []int, affectedMobInstanceIds []int) {
	tStart := time.Now()

	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil {
			continue
		}

		// Skip if player has no combat buff
		if user.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		if user.Character.Aggro == nil {
			continue
		}

		// Cancel buffs that are cancelled by combat
		user.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

		roomId := user.Character.RoomId
		uRoom := rooms.LoadRoom(roomId)
		if uRoom == nil {
			continue
		}

		// Handle flee attempts
		if user.Character.Aggro.Type == characters.Flee {
			rbc.handlePlayerFlee(user, uRoom, &affectedPlayerIds, &affectedMobInstanceIds)
			continue
		}

		// Handle different attack types
		switch user.Character.Aggro.Type {
		case characters.SpellCast:
			rbc.handlePlayerSpellCast(user, uRoom, &affectedPlayerIds, &affectedMobInstanceIds)
		default:
			rbc.handlePlayerPhysicalCombat(user, uRoom, &affectedPlayerIds, &affectedMobInstanceIds)
		}
	}

	util.TrackTime(`World::processPlayerCombat()`, time.Since(tStart).Seconds())
	return affectedPlayerIds, affectedMobInstanceIds
}

// handlePlayerFlee processes a player's flee attempt
func (rbc *RoundBasedCombat) handlePlayerFlee(user *users.UserRecord, room *rooms.Room, 
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	// Revert to default combat regardless of outcome
	user.Character.SetAggro(user.Character.Aggro.UserId, user.Character.Aggro.MobInstanceId, characters.DefaultAttack)

	blockedByMob := ``
	for _, mobInstId := range room.GetMobs(rooms.FindFighting) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Aggro == nil || mob.Character.Aggro.UserId != user.UserId {
			continue
		}

		// Stat comparison accounts for up to 70% of chance to flee
		chanceIn100 := int(float64(user.Character.Stats.Speed.ValueAdj) / 
			(float64(user.Character.Stats.Speed.ValueAdj) + float64(mob.Character.Stats.Speed.ValueAdj)) * 70)
		chanceIn100 += 30

		roll := util.Rand(100)
		util.LogRoll(`Flee`, roll, chanceIn100)

		if roll >= chanceIn100 {
			blockedByMob = mob.Character.Name
			break
		}
	}

	// Handle flee result
	if blockedByMob != `` {
		user.SendText(fmt.Sprintf(`<ansi fg="red">%s blocks your escape!</ansi>`, blockedByMob))
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="red">tries to flee, but is blocked by %s!</ansi>`, 
				user.Character.Name, blockedByMob),
			user.UserId)
	} else {
		// Successful flee
		exitName, _ := room.GetRandomExit()
		if exitName == `` {
			user.SendText(`<ansi fg="red">There's nowhere to run!</ansi>`)
			return
		}

		// Execute flee
		events.AddToQueue(events.Input{
			UserId:    user.UserId,
			InputText: exitName,
		})

		user.SendText(`<ansi fg="yellow">You flee in panic!</ansi>`)
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow">flees in panic!</ansi>`, user.Character.Name),
			user.UserId)

		// Clear combat state
		user.Character.EndAggro()
		rbc.timer.RemovePlayer(user.UserId)
	}
}

// handlePlayerSpellCast processes spell casting in combat
func (rbc *RoundBasedCombat) handlePlayerSpellCast(user *users.UserRecord, room *rooms.Room,
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	if user.Character.Aggro.RoundsWaiting > 0 {
		user.Character.Aggro.RoundsWaiting--
		
		scripting.TrySpellScriptEvent(`onWait`, user.UserId, 0, user.Character.Aggro.SpellInfo)
		
		return
	}

	successChance := user.Character.GetBaseCastSuccessChance(user.Character.Aggro.SpellInfo.SpellId)
	if util.RollDice(1, 100) >= successChance {
		// fail
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to cast a spell but it <ansi fg="magenta">fizzles</ansi>!`, user.Character.Name), user.UserId)
		user.SendText(`Your spell fizzles and fails!`)
		user.Character.Aggro = nil
		return
	}

	allowRetaliation := true
	if handled, err := scripting.TrySpellScriptEvent(`onMagic`, user.UserId, 0, user.Character.Aggro.SpellInfo); err == nil {
		if handled {
			allowRetaliation = false
		}
	}

	if allowRetaliation {
		if spellData := spells.GetSpell(user.Character.Aggro.SpellInfo.SpellId); spellData != nil {
			if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmMulti || spellData.Type == spells.HarmArea {
				
				// Handle spell effects on player targets
				for _, userId := range user.Character.Aggro.SpellInfo.TargetUserIds {
					*affectedPlayerIds = append(*affectedPlayerIds, userId)
					
					if defUser := users.GetByUserId(userId); defUser != nil {
						defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)
						
						if defUser.Character.Health <= 0 {
							defUser.Character.EndAggro()
						} else if defUser.Character.Aggro == nil {
							defUser.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
							// Timer registration happens when attack command is executed
						}
					}
				}
				
				// Handle spell effects on mob targets
				for _, mobId := range user.Character.Aggro.SpellInfo.TargetMobInstanceIds {
					*affectedMobInstanceIds = append(*affectedMobInstanceIds, mobId)
					
					if defMob := mobs.GetInstance(mobId); defMob != nil {
						defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)
						
						if defMob.Character.Health <= 0 {
							defMob.Character.EndAggro()
						} else if defMob.Character.Aggro == nil {
							defMob.PreventIdle = true
							defMob.Command(fmt.Sprintf("attack @%d", user.UserId)) // @ means player
						}
					}
				}
			}
		}
	}

	user.Character.Aggro = nil
}

// handlePlayerPhysicalCombat processes physical attacks
func (rbc *RoundBasedCombat) handlePlayerPhysicalCombat(user *users.UserRecord, room *rooms.Room,
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	if user.Character.Aggro == nil {
		return
	}

	// Handle PvP combat
	if user.Character.Aggro.UserId > 0 {
		rbc.handlePlayerVsPlayer(user, room, affectedPlayerIds, affectedMobInstanceIds)
		return
	}

	// Handle PvE combat
	if user.Character.Aggro.MobInstanceId > 0 {
		rbc.handlePlayerVsMob(user, room, affectedPlayerIds, affectedMobInstanceIds)
		return
	}
}

// handlePlayerVsPlayer processes player vs player combat
func (rbc *RoundBasedCombat) handlePlayerVsPlayer(user *users.UserRecord, room *rooms.Room,
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	defUser := users.GetByUserId(user.Character.Aggro.UserId)
	if defUser == nil {
		user.SendText(`Your target can't be found.`)
		user.Character.Aggro = nil
		return
	}

	// Check if target is in same room or at an exit
	targetFound := true
	if defUser.Character.RoomId != user.Character.RoomId {
		if user.Character.Aggro.ExitName == `` {
			targetFound = false
		} else {
			// Check if the exit leads to target's room
			if _, exitRoomId := room.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defUser.Character.RoomId {
				targetFound = false
			}
		}
	}

	if !targetFound {
		user.SendText(`Your target can't be found.`)
		user.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defUser.Character.RoomId)
	if defRoom == nil {
		user.Character.Aggro = nil
		return
	}

	// Cancel combat-cancelled buffs on defender
	defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defUser.Character.Health < 1 {
		user.SendText(`Your rage subsides.`)
		user.Character.Aggro = nil
		return
	}

	// Handle wait rounds (for slow weapons)
	if user.Character.Aggro.RoundsWaiting > 0 {
		user.Character.Aggro.RoundsWaiting--
		
		roundResult := combat.GetWaitMessages(items.Wait, user.Character, defUser.Character, combat.User, combat.User)
		
		for _, msg := range roundResult.MessagesToSource {
			user.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToTarget {
			defUser.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToSourceRoom {
			room.SendText(msg, user.UserId, defUser.UserId)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			defRoom.SendText(msg, user.UserId, defUser.UserId)
		}
		return
	}

	// Can't attack hidden targets
	if defUser.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You can't seem to find your target.")
		return
	}

	*affectedPlayerIds = append(*affectedPlayerIds, user.Character.Aggro.UserId)

	// Perform the attack
	roundResult := combat.AttackPlayerVsPlayer(user, defUser)

	// Check for charmed mobs helping the defender
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
			if charmedMob.Character.IsCharmed(defUser.UserId) && charmedMob.Character.Aggro == nil {
				// Set aggro to prevent multiple triggers
				charmedMob.Character.Aggro = &characters.Aggro{
					Type: characters.DefaultAttack,
				}
				charmedMob.Command(fmt.Sprintf("attack @%d", user.UserId))
			}
		}
	}

	// Apply buffs
	for _, buffId := range roundResult.BuffSource {
		user.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		defUser.AddBuff(buffId, `combat`)
	}

	// Send messages
	for _, msg := range roundResult.MessagesToSource {
		user.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToTarget {
		defUser.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		room.SendText(msg, user.UserId, defUser.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		defRoom.SendText(msg, user.UserId, defUser.UserId)
	}

	// Check for equipment damage
	if roundResult.Hit && util.Rand(100) < 5 { // 5% chance
		rbc.checkEquipmentDamage(defUser, defRoom, roundResult.Crit)
	}

	// Update aggro based on health
	if user.Character.Health <= 0 || defUser.Character.Health <= 0 {
		defUser.Character.EndAggro()
		user.Character.EndAggro()
		rbc.timer.RemovePlayer(user.UserId)
		rbc.timer.RemovePlayer(defUser.UserId)
	} else {
		user.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
	}
}

// handlePlayerVsMob processes player vs mob combat
func (rbc *RoundBasedCombat) handlePlayerVsMob(user *users.UserRecord, room *rooms.Room,
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	*affectedMobInstanceIds = append(*affectedMobInstanceIds, user.Character.Aggro.MobInstanceId)

	defMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId)
	if defMob == nil {
		user.SendText("Your target can't be found.")
		user.Character.Aggro = nil
		return
	}

	// Check if target is in same room or at an exit
	targetFound := true
	if defMob.Character.RoomId != user.Character.RoomId {
		if user.Character.Aggro.ExitName == `` {
			targetFound = false
		} else {
			// Check if the exit leads to target's room
			if _, exitRoomId := room.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defMob.Character.RoomId {
				targetFound = false
			}
		}
	}

	if !targetFound {
		user.SendText("Your target can't be found.")
		user.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defMob.Character.RoomId)
	if defRoom == nil {
		user.Character.Aggro = nil
		return
	}

	// Cancel combat-cancelled buffs on defender
	defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defMob.Character.Health < 1 {
		user.SendText("Your rage subsides.")
		user.Character.Aggro = nil
		return
	}

	// Handle wait rounds (for slow weapons)
	if user.Character.Aggro.RoundsWaiting > 0 {
		user.Character.Aggro.RoundsWaiting--
		
		roundResult := combat.GetWaitMessages(items.Wait, user.Character, &defMob.Character, combat.User, combat.Mob)
		
		for _, msg := range roundResult.MessagesToSource {
			user.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToSourceRoom {
			room.SendText(msg, user.UserId)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			defRoom.SendText(msg, user.UserId)
		}
		return
	}

	// Can't attack hidden targets
	if defMob.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You can't seem to find your target.")
		return
	}

	// Perform the attack
	roundResult := combat.AttackPlayerVsMob(user, defMob)

	// Apply buffs
	for _, buffId := range roundResult.BuffSource {
		user.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		defMob.AddBuff(buffId, `combat`)
	}

	// Send messages
	for _, msg := range roundResult.MessagesToSource {
		user.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		room.SendText(msg, user.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		defRoom.SendText(msg, user.UserId)
	}

	// Handle scripted behavior
	if roundResult.Hit {
		scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, user.UserId, `user`, 
			map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
	}

	// Handle mob group hostility
	c := configs.GetConfig()
	for _, groupName := range defMob.Groups {
		mobs.MakeHostile(groupName, user.UserId, c.Timing.MinutesToRounds(2)-user.Character.Stats.Perception.ValueAdj)
	}

	// Set mob aggro if not already fighting
	if defMob.Character.Aggro == nil {
		defMob.PreventIdle = true
		
		// If not in same room, move to player's room first
		if user.Character.RoomId != defMob.Character.RoomId {
			if mobRoom := rooms.LoadRoom(defMob.Character.RoomId); mobRoom != nil {
				for exitName, exitInfo := range mobRoom.Exits {
					if exitInfo.RoomId == user.Character.RoomId {
						defMob.Command(fmt.Sprintf(`go %s`, exitName))
						if actionStr := defMob.GetAngryCommand(); actionStr != `` {
							defMob.Command(actionStr)
						}
						break
					}
				}
			}
		}
		
		defMob.Command(fmt.Sprintf("attack @%d", user.UserId))
	}

	// Remember who hit the mob
	defMob.Character.TrackPlayerDamage(user.UserId, roundResult.DamageToTarget)

	// Update aggro based on health
	if user.Character.Health <= 0 || defMob.Character.Health <= 0 {
		user.Character.EndAggro()
		defMob.Character.EndAggro()
		rbc.timer.RemovePlayer(user.UserId)
	} else {
		user.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
	}
}

// checkEquipmentDamage checks if any equipment breaks during combat
func (rbc *RoundBasedCombat) checkEquipmentDamage(user *users.UserRecord, room *rooms.Room, isCrit bool) {
	// For now, only check offhand items
	if user.Character.Equipment.Offhand.ItemId > 0 {
		modifier := 0
		if isCrit { // Crits double the chance of breakage
			modifier = int(user.Character.Equipment.Offhand.GetSpec().BreakChance)
		}

		if user.Character.Equipment.Offhand.BreakTest(modifier) {
			// Send message about the break
			room.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="username">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, 
				user.Character.Equipment.Offhand.NameSimple(), user.Character.Name))

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   user.Character.Equipment.Offhand,
				Gained: false,
			})

			user.Character.RemoveFromBody(user.Character.Equipment.Offhand)
			brokenItem := items.New(20) // Broken item
			if !user.Character.StoreItem(brokenItem) {
				room.AddItem(brokenItem, false)
				
				events.AddToQueue(events.ItemOwnership{
					UserId: user.UserId,
					Item:   brokenItem,
					Gained: true,
				})
			}
		}
	}
}

// processMobCombat handles combat for all mobs
func (rbc *RoundBasedCombat) processMobCombat(roundNumber uint64) (affectedPlayerIds []int, affectedMobInstanceIds []int) {
	c := configs.GetConfig()
	tStart := time.Now()

	// Handle mob round of combat
	for _, mobId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(mobId)
		
		// Only handling combat functions here, so ditch out if not in combat
		if mob == nil || mob.Character.Aggro == nil {
			continue
		}

		// If has a buff that prevents combat, skip the mob
		if mob.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		mobRoom := rooms.LoadRoom(mob.Character.RoomId)
		if mobRoom == nil {
			mob.Character.Aggro = nil
			continue
		}

		// Disable any buffs that are cancelled by combat
		mob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

		// Handle spell casting
		if mob.Character.Aggro.Type == characters.SpellCast {
			rbc.handleMobSpellCast(mob, mobRoom, &affectedPlayerIds, &affectedMobInstanceIds)
			continue
		}

		// Handle physical combat
		// H2H is the base level combat, can do combat commands then
		if mob.Character.Aggro.Type == characters.DefaultAttack {
			// If they have idle commands, maybe do one of them?
			cmdCt := len(mob.CombatCommands)
			if cmdCt > 0 {
				// Each mob has a 10% chance of doing an idle action.
				if util.Rand(100) < mob.ActivityLevel {
					combatAction := mob.CombatCommands[util.Rand(cmdCt)]
					
					if combatAction == `` { // blank is a no-op
						continue
					}

					var waitTime float64 = 0.0
					allCmds := strings.Split(combatAction, `;`)
					if len(allCmds) >= c.Timing.TurnsPerRound() {
						mob.Command(`say I have a CombatAction that is too long. Please notify an admin.`)
					} else {
						for _, action := range strings.Split(combatAction, `;`) {
							mob.Command(action, waitTime)
							waitTime += 0.1
						}
					}
					continue
				}
			}
		}

		// mob attacks player
		if mob.Character.Aggro.UserId > 0 {
			rbc.handleMobVsPlayer(mob, mobRoom, &affectedPlayerIds, &affectedMobInstanceIds)
		}

		// mob attacks mob
		if mob.Character.Aggro.MobInstanceId > 0 {
			rbc.handleMobVsMob(mob, mobRoom, &affectedPlayerIds, &affectedMobInstanceIds)
		}
	}

	util.TrackTime(`World::processMobCombat()`, time.Since(tStart).Seconds())
	return affectedPlayerIds, affectedMobInstanceIds
}

// handleAffected processes post-combat effects like deaths
func (rbc *RoundBasedCombat) handleAffected(affectedPlayerIds []int, affectedMobInstanceIds []int) {
	playersHandled := map[int]struct{}{}
	for _, userId := range affectedPlayerIds {
		if _, ok := playersHandled[userId]; ok {
			continue
		}
		playersHandled[userId] = struct{}{}

		user := users.GetByUserId(userId)
		if user == nil {
			continue
		}

		if user.Character.Health <= -10 {
			user.Command(`suicide`) // suicide drops all money/items and transports to land of the dead
		} else if user.Character.Health < 1 {
			events.AddToQueue(events.PlayerDrop{UserId: user.UserId, RoomId: user.Character.RoomId})
		}
	}

	mobsHandled := map[int]struct{}{}
	for _, mobId := range affectedMobInstanceIds {
		if _, ok := mobsHandled[mobId]; ok {
			continue
		}
		mobsHandled[mobId] = struct{}{}

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		if mob.Character.Health < 1 {
			mob.Command(`suicide`)
		}
	}
}

// handleMobSpellCast processes mob spell casting
func (rbc *RoundBasedCombat) handleMobSpellCast(mob *mobs.Mob, mobRoom *rooms.Room, 
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	if mob.Character.Aggro.RoundsWaiting > 0 {
		mob.Character.Aggro.RoundsWaiting--
		
		scripting.TrySpellScriptEvent(`onWait`, 0, mob.InstanceId, mob.Character.Aggro.SpellInfo)
		
		return
	}

	successChance := mob.Character.GetBaseCastSuccessChance(mob.Character.Aggro.SpellInfo.SpellId)
	if util.RollDice(1, 100) >= successChance {
		// fail
		mobRoom.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to cast a spell but it <ansi fg="magenta">fizzles</ansi>!`, mob.Character.Name))
		mob.Character.Aggro = nil
		return
	}

	allowRetaliation := true
	if handled, err := scripting.TrySpellScriptEvent(`onMagic`, 0, mob.InstanceId, mob.Character.Aggro.SpellInfo); err == nil {
		if handled {
			allowRetaliation = false
		}
	}

	if allowRetaliation {
		if spellData := spells.GetSpell(mob.Character.Aggro.SpellInfo.SpellId); spellData != nil {
			if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmMulti || spellData.Type == spells.HarmArea {
				
				for _, mobId := range mob.Character.Aggro.SpellInfo.TargetMobInstanceIds {
					*affectedMobInstanceIds = append(*affectedMobInstanceIds, mobId)
					
					if defMob := mobs.GetInstance(mobId); defMob != nil {
						defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)
						
						if defMob.Character.Health <= 0 {
							defMob.Character.EndAggro()
						} else if defMob.Character.Aggro == nil {
							defMob.PreventIdle = true
							defMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId)) // # means mob
						}
					}
				}
			}
		}
	}

	mob.Character.Aggro = nil
}

// handleMobVsPlayer processes mob vs player combat
func (rbc *RoundBasedCombat) handleMobVsPlayer(mob *mobs.Mob, mobRoom *rooms.Room,
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	*affectedMobInstanceIds = append(*affectedMobInstanceIds, mob.InstanceId)
	
	defUser := users.GetByUserId(mob.Character.Aggro.UserId)
	if defUser == nil || mob.Character.RoomId != defUser.Character.RoomId {
		mob.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defUser.Character.RoomId)
	if defRoom == nil {
		mob.Character.Aggro = nil
		return
	}

	defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defUser.Character.Health < 1 {
		mob.Character.Aggro = nil
		return
	}

	// Can't see them, can't fight them.
	if defUser.Character.HasBuffFlag(buffs.Hidden) {
		return
	}

	*affectedPlayerIds = append(*affectedPlayerIds, mob.Character.Aggro.UserId)

	// If no weapon but has stuff in the backpack, look for a weapon
	// Especially useful for when they get disarmed
	if mob.Character.Equipment.Weapon.ItemId == 0 && len(mob.Character.Items) > 0 {
		roll := util.Rand(100)
		util.LogRoll(`Look for weapon`, roll, mob.Character.Stats.Perception.ValueAdj)
		
		if roll < mob.Character.Stats.Perception.ValueAdj {
			possibleWeapons := []string{}
			for _, itm := range mob.Character.Items {
				iSpec := itm.GetSpec()
				if iSpec.Type == items.Weapon {
					possibleWeapons = append(possibleWeapons, itm.DisplayName())
				}
			}

			if len(possibleWeapons) > 0 {
				mob.Command(fmt.Sprintf("equip %s", possibleWeapons[util.Rand(len(possibleWeapons))]))
			}
		}
	}

	// Handle wait rounds (for slow weapons)
	if mob.Character.Aggro.RoundsWaiting > 0 {
		mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)
		
		mob.Character.Aggro.RoundsWaiting--
		
		roundResult := combat.GetWaitMessages(items.Wait, &mob.Character, defUser.Character, combat.Mob, combat.User)
		
		for _, msg := range roundResult.MessagesToTarget {
			defUser.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToSourceRoom {
			mobRoom.SendText(msg, defUser.UserId)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			defRoom.SendText(msg, defUser.UserId)
		}
		
		return
	}

	// Perform the attack
	roundResult := combat.AttackMobVsPlayer(mob, defUser)

	// If a mob attacks a player, check whether player has a charmed mob helping them
	for _, instanceId := range mobRoom.GetMobs(rooms.FindCharmed) {
		if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
			if charmedMob.Character.IsCharmed(defUser.UserId) && charmedMob.Character.Aggro == nil {
				// This is set to prevent it from triggering more than once
				charmedMob.Character.Aggro = &characters.Aggro{
					Type: characters.DefaultAttack,
				}
				charmedMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))
			}
		}
	}

	// Apply buffs
	for _, buffId := range roundResult.BuffSource {
		mob.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		defUser.AddBuff(buffId, `combat`)
	}

	// Send messages
	for _, msg := range roundResult.MessagesToTarget {
		defUser.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		mobRoom.SendText(msg, defUser.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		defRoom.SendText(msg, defUser.UserId)
	}

	// If the attack connected, check for damage to equipment.
	if roundResult.Hit && util.Rand(100) < 5 { // 5% chance
		rbc.checkEquipmentDamage(defUser, defRoom, roundResult.Crit)
	}

	// Update aggro based on health
	if mob.Character.Health <= 0 || defUser.Character.Health <= 0 {
		mob.Character.EndAggro()
		defUser.Character.EndAggro()
		rbc.timer.RemovePlayer(defUser.UserId)
	} else {
		mob.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
	}
}

// handleMobVsMob processes mob vs mob combat
func (rbc *RoundBasedCombat) handleMobVsMob(mob *mobs.Mob, mobRoom *rooms.Room,
	affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	
	*affectedMobInstanceIds = append(*affectedMobInstanceIds, mob.Character.Aggro.MobInstanceId)
	
	defMob := mobs.GetInstance(mob.Character.Aggro.MobInstanceId)
	
	if defMob == nil || mob.Character.RoomId != defMob.Character.RoomId {
		mob.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defMob.Character.RoomId)
	
	defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)
	
	if defMob.Character.Health < 1 {
		mob.Character.Aggro = nil
		return
	}

	// Handle wait rounds (for slow weapons)
	if mob.Character.Aggro.RoundsWaiting > 0 {
		mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)
		
		mob.Character.Aggro.RoundsWaiting--
		
		roundResult := combat.GetWaitMessages(items.Wait, &mob.Character, &defMob.Character, combat.Mob, combat.Mob)
		
		for _, msg := range roundResult.MessagesToSourceRoom {
			mobRoom.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			defRoom.SendText(msg)
		}
		
		return
	}

	// Can't see them, can't fight them.
	if defMob.Character.HasBuffFlag(buffs.Hidden) {
		return
	}

	// Perform the attack
	roundResult := combat.AttackMobVsMob(mob, defMob)

	// Apply buffs
	for _, buffId := range roundResult.BuffSource {
		mob.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		defMob.AddBuff(buffId, `combat`)
	}

	// Send messages
	for _, msg := range roundResult.MessagesToSourceRoom {
		mobRoom.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		defRoom.SendText(msg)
	}

	// Handle any scripted behavior now.
	if roundResult.Hit {
		scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, mob.InstanceId, `mob`, 
			map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
	}

	// Mobs get aggro when attacked
	if defMob.Character.Aggro == nil {
		defMob.PreventIdle = true
		defMob.Character.Aggro = &characters.Aggro{
			Type: characters.DefaultAttack,
		}
		defMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId)) // # means mob
	}

	// If the attack connected, check for damage to equipment.
	if roundResult.Hit {
		// For now, only focus on offhand items.
		if defMob.Character.Equipment.Offhand.ItemId > 0 {
			modifier := 0
			if roundResult.Crit { // Crits double the chance of breakage for offhand items.
				modifier = int(defMob.Character.Equipment.Offhand.GetSpec().BreakChance)
			}

			if defMob.Character.Equipment.Offhand.BreakTest(modifier) {
				// Send message about the break
				if defRoom != nil {
					defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="mobname">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, 
						defMob.Character.Equipment.Offhand.NameSimple(), defMob.Character.Name))

					events.AddToQueue(events.ItemOwnership{
						MobInstanceId: defMob.InstanceId,
						Item:          defMob.Character.Equipment.Offhand,
						Gained:        false,
					})

					defMob.Character.RemoveFromBody(defMob.Character.Equipment.Offhand)
					itm := items.New(20) // Broken item
					if !defMob.Character.StoreItem(itm) {
						defRoom.AddItem(itm, false)
						
						events.AddToQueue(events.ItemOwnership{
							MobInstanceId: defMob.InstanceId,
							Item:          itm,
							Gained:        true,
						})
					}
				}
			}
		}
	}

	// Update aggro based on health
	if mob.Character.Health <= 0 || defMob.Character.Health <= 0 {
		mob.Character.EndAggro()
		defMob.Character.EndAggro()
	} else {
		mob.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
	}
}