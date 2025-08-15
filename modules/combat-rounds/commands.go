package combatrounds

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// registerCommands registers all combat-related commands
func (rbc *RoundBasedCombat) registerCommands() {
	// Register user commands
	rbc.plug.AddUserCommand("attack", rbc.attackCommand, false, false)
	rbc.plug.AddUserCommand("kill", rbc.attackCommand, false, false) // Alias for attack
	rbc.plug.AddUserCommand("flee", rbc.fleeCommand, false, false)
	rbc.plug.AddUserCommand("consider", rbc.considerCommand, false, false)
	rbc.plug.AddUserCommand("combatinfo", rbc.combatInfoCommand, true, true) // Admin only
	rbc.plug.AddUserCommand("config", rbc.configCommand, true, true)         // Admin only

	// Register mob commands
	rbc.plug.AddMobCommand("attack", rbc.mobAttackCommand, false)
	rbc.plug.AddMobCommand("flee", rbc.mobFleeCommand, false)

	mudlog.Info("Combat Commands", "action", "registered", "module", "combat-rounds")
}

// unregisterCommands removes all combat-related commands
func (rbc *RoundBasedCombat) unregisterCommands() {
	// TODO: RemoveUserCommand and RemoveMobCommand don't exist in the plugin interface yet
	// For now, we'll rely on the command registration being overwritten when switching systems
	// rbc.plug.RemoveUserCommand("attack")
	// rbc.plug.RemoveUserCommand("kill")
	// rbc.plug.RemoveUserCommand("flee")
	// rbc.plug.RemoveUserCommand("consider")
	// rbc.plug.RemoveUserCommand("combatinfo")
	// rbc.plug.RemoveUserCommand("config")
	// rbc.plug.RemoveMobCommand("attack")
	// rbc.plug.RemoveMobCommand("flee")

	mudlog.Info("Combat Commands", "action", "unregistered", "module", "combat-rounds")
}

// fleeCommand handles player flee commands
func (rbc *RoundBasedCombat) fleeCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if user.Character.Aggro == nil || user.Character.Aggro.Type != characters.Flee {
		user.SendText(`You attempt to flee...`)

		user.Character.Aggro = &characters.Aggro{}
		user.Character.Aggro.Type = characters.Flee
	}

	return true, nil
}

// attackCommand handles player attack commands
func (rbc *RoundBasedCombat) attackCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	attackPlayerId := 0
	attackMobInstanceId := 0

	if rest == `` {
		partyInfo := parties.Get(user.UserId)

		// If no argument supplied, attack whoever is attacking the player currently.
		for _, mId := range room.GetMobs(rooms.FindFightingPlayer) {
			m := mobs.GetInstance(mId)
			if m.Character.Aggro == nil {
				continue
			}

			if m.Character.Aggro.UserId == user.UserId {
				attackMobInstanceId = m.InstanceId
				break
			}

			if partyInfo != nil {
				if partyInfo.IsMember(m.Character.Aggro.UserId) {
					attackMobInstanceId = m.InstanceId
					break
				}
			}
		}

		if attackMobInstanceId == 0 {
			for _, uId := range room.GetPlayers(rooms.FindFightingPlayer) {
				u := users.GetByUserId(uId)
				if u.Character.Aggro == nil {
					continue
				}

				if u.Character.Aggro.UserId == user.UserId {
					attackPlayerId = u.UserId
					break
				}

				if partyInfo != nil {
					if partyInfo.IsMember(u.Character.Aggro.UserId) {
						attackPlayerId = u.UserId
						break
					}
				}
			}
		}

		// Finally, if still no targets, check if any party members are aggroed and just glom onto that
		if attackMobInstanceId == 0 && attackPlayerId == 0 {
			if partyInfo != nil {
				for uId := range partyInfo.GetMembers() {
					if partyUser := users.GetByUserId(uId); partyUser != nil {
						if partyUser.Character.Aggro == nil {
							continue
						}

						if partyUser.Character.Aggro.MobInstanceId > 0 {
							attackMobInstanceId = partyUser.Character.Aggro.MobInstanceId
							break
						}

						if partyUser.Character.Aggro.UserId > 0 {
							attackPlayerId = partyUser.Character.Aggro.UserId
							break
						}

					}
				}
			}
		}

	} else if rest[0] == '*' { // choose a target at random. Friend or foe.

		if rest == `*` { // * ANYONE

			allMobs := room.GetMobs()
			allPlayers := []int{}
			for _, userId := range room.GetPlayers() {
				if userId == user.UserId {
					continue
				}
				allPlayers = append(allPlayers, userId)
			}

			randomSelection := util.Rand(len(allMobs) + len(allPlayers))

			if randomSelection < len(allMobs) {
				attackMobInstanceId = allMobs[randomSelection]
			} else {
				randomSelection -= len(allMobs)
				attackPlayerId = allPlayers[randomSelection]
			}

		} else if rest == `*mob` { // *mob ANY MOB

			if allMobs := room.GetMobs(); len(allMobs) > 0 {
				attackMobInstanceId = allMobs[util.Rand(len(allMobs))]
			}

		} else { // *user etc. ANY PLAYER

			allPlayers := []int{}
			for _, userId := range room.GetPlayers() {
				if userId == user.UserId {
					continue
				}
				allPlayers = append(allPlayers, userId)
			}

			if len(allPlayers) > 0 {
				attackPlayerId = allPlayers[util.Rand(len(allPlayers))]
			}

		}

	} else {
		attackPlayerId, attackMobInstanceId = room.FindByName(rest)
	}

	if attackPlayerId == user.UserId { // Can't attack self!
		attackPlayerId = 0
	}

	if attackMobInstanceId == 0 && attackPlayerId == 0 {
		user.SendText("You attack the darkness!")
		return true, nil
	}

	isSneaking := user.Character.HasBuffFlag(buffs.Hidden)

	if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {
			if m.Character.IsCharmed(user.UserId) {
				user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is your friend!`, m.Character.Name))
				return true, nil
			}

			if party := parties.Get(user.UserId); party != nil {
				if party.IsLeader(user.UserId) {
					for _, id := range party.GetAutoAttackUserIds() {
						if id == user.UserId {
							continue
						}
						if partyUser := users.GetByUserId(id); partyUser != nil {
							if partyUser.Character.RoomId == user.Character.RoomId {

								partyUser.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId)) // # denotes a specific mob instanceId

							}
						}

					}
				}
			}

			user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)

			// Register player with combat timer
			if rbc.timer != nil {
				rbc.timer.AddPlayer(user.UserId)
			}

			user.SendText(
				fmt.Sprintf(`You prepare to enter into mortal combat with <ansi fg="mobname">%s</ansi>.`, m.Character.Name),
			)

			if !isSneaking {
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>.`, user.Character.Name, m.Character.Name),
					user.UserId,
				)
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if m.Character.Aggro == nil && m.Character.IsCharmed(user.UserId) { // Charmed mobs help the player

						m.Command(fmt.Sprintf(`attack #%d`, attackMobInstanceId)) // # denotes a specific mob instanceId

					}
				}
			}

		}

	} else if attackPlayerId > 0 {

		if p := users.GetByUserId(attackPlayerId); p != nil {

			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendText(pvpErr.Error())
				return true, nil
			}

			if partyInfo := parties.Get(user.UserId); partyInfo != nil {
				if partyInfo.IsMember(attackPlayerId) {
					user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
					return true, nil
				}
			}

			if party := parties.Get(user.UserId); party != nil {
				if party.IsLeader(user.UserId) {
					for _, id := range party.GetAutoAttackUserIds() {
						if id == user.UserId {
							continue
						}
						if partyUser := users.GetByUserId(id); partyUser != nil {
							if partyUser.Character.RoomId == user.Character.RoomId {
								partyUser.Command(fmt.Sprintf(`attack @%d`, attackPlayerId)) // @ denotes a specific user id
							}
						}
					}
				}
			}

			user.Character.SetAggro(attackPlayerId, 0, characters.DefaultAttack)

			// Register player with combat timer
			if rbc.timer != nil {
				rbc.timer.AddPlayer(user.UserId)
			}

			user.SendText(
				fmt.Sprintf(`You prepare to enter into mortal combat with <ansi fg="username">%s</ansi>.`, p.Character.Name),
			)

			if !isSneaking {

				p.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight you!`, user.Character.Name),
				)

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>.`, user.Character.Name, p.Character.Name),
					user.UserId, attackPlayerId)
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				if m := mobs.GetInstance(instId); m != nil {
					if m.Character.Aggro == nil && m.Character.IsCharmed(user.UserId) { // Charmed mobs help the player

						m.Command(fmt.Sprintf(`attack @%d`, attackPlayerId)) // @ denotes a specific user id

					}
				}
			}

		}

	}

	return true, nil
}

// considerCommand shows combat information about a target
func (rbc *RoundBasedCombat) considerCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(rest)

	// Looking AT something?
	if len(args) > 0 {
		lookAt := args[0]

		// look for any mobs, players, npcs
		playerId, mobId := room.FindByName(lookAt)
		if playerId == user.UserId {
			playerId = 0
		}

		if playerId > 0 || mobId > 0 {

			ratio := 0.0

			considerType := "mob"
			considerName := "nobody"

			if playerId > 0 {
				u := users.GetByUserId(playerId)

				p1 := rbc.calculator.PowerRanking(user.Character, u.Character)
				p2 := rbc.calculator.PowerRanking(u.Character, user.Character)

				ratio = p1 / p2
				considerType = "user"
				considerName = u.Character.Name

			} else if mobId > 0 {

				m := mobs.GetInstance(mobId)

				p1 := rbc.calculator.PowerRanking(user.Character, &m.Character)
				p2 := rbc.calculator.PowerRanking(&m.Character, user.Character)

				ratio = p1 / p2
				considerType = "mob"
				considerName = m.Character.Name
			}

			prediction := `Unknown`
			if ratio > 4 {
				prediction = `<ansi fg="blue-bold">Very Favorable</ansi>`
			} else if ratio > 3 {
				prediction = `<ansi fg="green">Favorable</ansi>`
			} else if ratio > 2 {
				prediction = `<ansi fg="green">Good</ansi>`
			} else if ratio > 1 {
				prediction = `<ansi fg="yellow">Okay</ansi>`
			} else if ratio > 0.5 {
				prediction = `<ansi fg="red-bold">Bad</ansi>`
			} else if ratio > 0 {
				prediction = `<ansi fg="red-bold">Very Bad</ansi>`
			} else {
				prediction = `<ansi fg="red-bold">YOU WILL DIE</ansi>`
			}

			user.SendText(
				fmt.Sprintf(`You consider <ansi fg="%sname">%s</ansi>...`, considerType, considerName),
			)
			user.SendText(
				fmt.Sprintf(`It is estimated that your chances to kill <ansi fg="%sname">%s</ansi> are %s (%f)`, considerType, considerName, prediction, ratio),
			)
		}
	}

	return true, nil
}

// mobAttackCommand handles mob attack commands
func (rbc *RoundBasedCombat) mobAttackCommand(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 && rest != "" {
		return true, nil
	}

	attackPlayerId := 0
	attackMobInstanceId := 0

	if rest == `` {
		// If no argument supplied, attack whoever is attacking the mob currently.
		for _, mId := range room.GetMobs(rooms.FindFightingMob) {
			m := mobs.GetInstance(mId)
			if m.Character.Aggro != nil && m.Character.Aggro.MobInstanceId == mob.InstanceId {
				attackMobInstanceId = m.InstanceId
				break
			}
		}

		if attackMobInstanceId == 0 {
			for _, uId := range room.GetPlayers(rooms.FindFightingMob) {
				u := users.GetByUserId(uId)
				if u.Character.Aggro != nil && u.Character.Aggro.MobInstanceId == mob.InstanceId {
					attackPlayerId = u.UserId
					break
				}
			}
		}
	} else if rest[0] == '*' { // choose a target at random. Friend or foe.

		if rest == `*` { // * ANYONE

			allMobs := []int{}
			allPlayers := room.GetPlayers()
			for _, mobInstanceId := range room.GetMobs() {
				if mobInstanceId == mob.InstanceId {
					continue
				}
				allMobs = append(allMobs, mobInstanceId)
			}

			randomSelection := util.Rand(len(allMobs) + len(allPlayers))

			if randomSelection < len(allMobs) {
				attackMobInstanceId = allMobs[randomSelection]
			} else {
				randomSelection -= len(allMobs)
				attackPlayerId = allPlayers[randomSelection]
			}

		} else if rest == `*mob` { // *mob ANY MOB

			allMobs := []int{}
			for _, mobInstanceId := range room.GetMobs() {
				if mobInstanceId == mob.InstanceId {
					continue
				}
				allMobs = append(allMobs, mobInstanceId)
			}

			if len(allMobs) > 0 {
				attackMobInstanceId = allMobs[util.Rand(len(allMobs))]
			}

		} else { // *user etc. ANY PLAYER

			if allPlayers := room.GetPlayers(); len(allPlayers) > 0 {
				attackPlayerId = allPlayers[util.Rand(len(allPlayers))]
			}

		}

	} else if rest[0] == '#' && len(rest) > 1 {
		// Direct instance ID targeting for mobs
		if instanceId, err := strconv.Atoi(rest[1:]); err == nil {
			attackMobInstanceId = instanceId
		}
	} else if rest[0] == '@' && len(rest) > 1 {
		// Direct user ID targeting
		if userId, err := strconv.Atoi(rest[1:]); err == nil {
			attackPlayerId = userId
		}
	} else {
		attackPlayerId, attackMobInstanceId = room.FindByName(rest)
	}

	if attackMobInstanceId == mob.InstanceId { // Can't attack self!
		attackMobInstanceId = 0
	}

	isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

	if attackPlayerId > 0 {

		u := users.GetByUserId(attackPlayerId)

		if u != nil {

			// Track that they've attacked this player
			mob.PlayerAttacked(attackPlayerId)

			mob.Character.SetAggro(attackPlayerId, 0, characters.DefaultAttack)

			if !isSneaking {

				u.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight you!`, mob.Character.Name))

				room.SendText(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="username">%s</ansi>`, mob.Character.Name, u.Character.Name),
					u.UserId)

			}
		}

		return true, nil

	} else if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {

			mob.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)

			if !isSneaking {

				room.SendText(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to fight <ansi fg="mobname">%s</ansi>`, mob.Character.Name, m.Character.Name))

			}

		}

		return true, nil
	}

	if !isSneaking {
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks confused and upset.`, mob.Character.Name))
	}

	return true, nil
}

// mobFleeCommand handles mob flee commands
func (rbc *RoundBasedCombat) mobFleeCommand(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if mob.Character.Aggro == nil || mob.Character.Aggro.Type != characters.Flee {
		// Mobs don't send text when fleeing, they just set their aggro type
		mob.Character.Aggro = &characters.Aggro{}
		mob.Character.Aggro.Type = characters.Flee

		// Optionally show room message
		room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to flee!`, mob.Character.Name))
	}

	return true, nil
}

// combatInfoCommand shows detailed combat calculations for a target
func (rbc *RoundBasedCombat) combatInfoCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	var targetChar *characters.Character
	var targetName string
	var isPlayer bool

	if rest == "" {
		// Show info for the player themselves
		targetChar = user.Character
		targetName = user.Character.Name
		isPlayer = true
	} else {
		// Find target using the same logic as attack command
		attackPlayerId, attackMobInstanceId := room.FindByName(rest)

		if attackPlayerId > 0 && attackPlayerId != user.UserId {
			if targetUser := users.GetByUserId(attackPlayerId); targetUser != nil {
				targetChar = targetUser.Character
				targetName = targetUser.Character.Name
				isPlayer = true
			}
		} else if attackMobInstanceId > 0 {
			if targetMob := mobs.GetInstance(attackMobInstanceId); targetMob != nil {
				targetChar = &targetMob.Character
				targetName = fmt.Sprintf("%s (#%d)", targetMob.Character.Name, attackMobInstanceId)
				isPlayer = false
			}
		}

		if targetChar == nil {
			user.SendText("Target not found.")
			return true, nil
		}
	}

	c := configs.GetConfig()
	baseRoundSeconds := c.Timing.RoundSeconds

	// Header
	user.SendText(fmt.Sprintf(`<ansi fg="yellow">═══ Combat Information: %s ═══</ansi>`, targetName))
	user.SendText("")

	// Character type
	if isPlayer {
		user.SendText(`<ansi fg="cyan">Type:</ansi>            Player`)
	} else {
		user.SendText(`<ansi fg="cyan">Type:</ansi>            Mob`)
	}

	// For round-based combat, show different information
	user.SendText("")
	user.SendText(`<ansi fg="yellow">── Combat Statistics ──</ansi>`)
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Level:</ansi>           %d`, targetChar.Level))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Health:</ansi>          %d / %d`, targetChar.Health, targetChar.HealthMax.Value))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Alignment:</ansi>       %d`, targetChar.Alignment))

	// Weapon info
	user.SendText("")
	user.SendText(`<ansi fg="yellow">── Weapon Information ──</ansi>`)

	if targetChar.Equipment.Weapon.ItemId > 0 {
		weaponSpec := targetChar.Equipment.Weapon.GetSpec()
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Weapon:</ansi>          %s`, weaponSpec.Name))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Damage:</ansi>          %s`, weaponSpec.Damage.DiceRoll))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Wait Rounds:</ansi>     %d rounds`, weaponSpec.WaitRounds))

		if weaponSpec.WaitRounds > 0 {
			user.SendText(fmt.Sprintf(`<ansi fg="cyan">Attack Interval:</ansi> Every %d rounds`, weaponSpec.WaitRounds+1))
		} else {
			user.SendText(`<ansi fg="cyan">Attack Interval:</ansi> Every round`)
		}
	} else {
		user.SendText(`<ansi fg="cyan">Weapon:</ansi>          None (unarmed)`)
		user.SendText(`<ansi fg="cyan">Damage:</ansi>          1d3 (fists)`)
		user.SendText(`<ansi fg="cyan">Attack Interval:</ansi> Every round`)
	}

	// All stats for reference
	user.SendText("")
	user.SendText(`<ansi fg="yellow">── Character Stats ──</ansi>`)
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Strength:</ansi>        %d (adj: %d)`, targetChar.Stats.Strength.Value, targetChar.Stats.Strength.ValueAdj))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Speed:</ansi>           %d (adj: %d)`, targetChar.Stats.Speed.Value, targetChar.Stats.Speed.ValueAdj))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Smarts:</ansi>          %d (adj: %d)`, targetChar.Stats.Smarts.Value, targetChar.Stats.Smarts.ValueAdj))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Vitality:</ansi>        %d (adj: %d)`, targetChar.Stats.Vitality.Value, targetChar.Stats.Vitality.ValueAdj))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Mysticism:</ansi>       %d (adj: %d)`, targetChar.Stats.Mysticism.Value, targetChar.Stats.Mysticism.ValueAdj))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Perception:</ansi>      %d (adj: %d)`, targetChar.Stats.Perception.Value, targetChar.Stats.Perception.ValueAdj))

	// Combat system info
	user.SendText("")
	user.SendText(`<ansi fg="yellow">── Combat System ──</ansi>`)
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Active System:</ansi>   %s`, combat.GetActiveCombatSystemName()))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Round Duration:</ansi>  %d seconds`, baseRoundSeconds))
	user.SendText(`<ansi fg="cyan">Note:</ansi>            In round-based combat, attacks happen automatically`)
	user.SendText(`                  based on weapon speed and combat rounds.`)

	return true, nil
}

// configCommand handles combat configuration
func (rbc *RoundBasedCombat) configCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	parts := strings.Fields(rest)

	// No arguments - show available config categories
	if len(parts) == 0 {
		user.SendText(`<ansi fg="yellow">Configuration Categories:</ansi>`)
		user.SendText(`  config combat      - Combat system configuration`)
		// Other config categories would be listed here in the future
		return true, nil
	}

	// Check for "combat" subcommand
	if parts[0] != "combat" {
		user.SendText(`Usage: config combat [...]`)
		return true, nil
	}

	// Remove "combat" from parts
	parts = parts[1:]

	// No arguments after combat - show combat config help
	if len(parts) == 0 {
		user.SendText(`<ansi fg="yellow">Combat Configuration Commands:</ansi>`)
		user.SendText(`  config combat info         - Show current combat settings`)
		user.SendText(`  config combat system <name>  - Switch combat system`)
		user.SendText(`  config combat systems      - List available combat systems`)
		user.SendText(``)
		user.SendText(`<ansi fg="yellow">Note:</ansi> Round-based combat has no configurable cooldown settings.`)
		user.SendText(`      Attack frequency is determined by weapon wait rounds.`)
		return true, nil
	}

	// Handle subcommands
	switch parts[0] {
	case "info":
		// Show current combat settings
		user.SendText(fmt.Sprintf(`<ansi fg="yellow">═══ Current Combat Configuration ═══</ansi>`))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Active System:</ansi>            %s`, combat.GetActiveCombatSystemName()))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Round Duration:</ansi>           %d seconds`, configs.GetConfig().Timing.RoundSeconds))
		user.SendText(``)
		user.SendText(`<ansi fg="yellow">Note:</ansi> Round-based combat uses weapon wait rounds and round`)
		user.SendText(`      duration to determine attack frequency.`)
		return true, nil

	case "system":
		// Switch combat system
		if len(parts) < 2 {
			user.SendText(`Usage: config combat system <name>`)
			return true, nil
		}

		newSystem := parts[1]
		currentSystem := combat.GetActiveCombatSystemName()

		if newSystem == currentSystem {
			user.SendText(fmt.Sprintf(`Combat system is already set to: %s`, currentSystem))
			return true, nil
		}

		// Collect all combat states before switching
		type CombatState struct {
			userId int
			aggro  *characters.Aggro
		}
		savedStates := []CombatState{}

		// Pause all combat - save and clear aggro states
		for _, u := range users.GetAllActiveUsers() {
			if u.Character.Aggro != nil {
				savedStates = append(savedStates, CombatState{
					userId: u.UserId,
					aggro: &characters.Aggro{
						Type:          u.Character.Aggro.Type,
						MobInstanceId: u.Character.Aggro.MobInstanceId,
						UserId:        u.Character.Aggro.UserId,
					},
				})
				u.Character.Aggro = nil
				u.SendText(`<ansi fg="yellow">[SYSTEM] Combat paused for system transition...</ansi>`)
			}
		}

		// Also handle mob aggro - iterate through all active mobs
		savedMobStates := make(map[int]*characters.Aggro)
		// Get unique rooms with users or mobs
		checkedRooms := make(map[int]bool)
		for _, u := range users.GetAllActiveUsers() {
			if roomObj := rooms.LoadRoom(u.Character.RoomId); roomObj != nil {
				if !checkedRooms[roomObj.RoomId] {
					checkedRooms[roomObj.RoomId] = true
					for _, mobId := range roomObj.GetMobs() {
						if mob := mobs.GetInstance(mobId); mob != nil && mob.Character.Aggro != nil {
							savedMobStates[mobId] = &characters.Aggro{
								Type:          mob.Character.Aggro.Type,
								MobInstanceId: mob.Character.Aggro.MobInstanceId,
								UserId:        mob.Character.Aggro.UserId,
							}
							mob.Character.Aggro = nil
						}
					}
				}
			}
		}

		// Convert saved states to the combat package format
		combatStates := make([]combat.CombatState, len(savedStates))
		for i, state := range savedStates {
			combatStates[i] = combat.CombatState{
				UserId: state.userId,
				Aggro:  state.aggro,
			}
		}

		// Queue the combat system switch event
		user.SendText(`<ansi fg="yellow">Switching combat systems...</ansi>`)
		events.AddToQueue(combat.SwitchCombatSystemEvent{
			NewSystem:      newSystem,
			OldSystem:      currentSystem,
			RequestingUser: user.UserId,
			SavedStates:    combatStates,
			SavedMobStates: savedMobStates,
		})

		return true, nil

	case "systems":
		// List available systems
		systems := combat.ListCombatSystems()
		current := combat.GetActiveCombatSystemName()

		user.SendText(`<ansi fg="yellow">Available Combat Systems:</ansi>`)
		for _, sys := range systems {
			if sys == current {
				user.SendText(fmt.Sprintf(`  <ansi fg="green">%s (active)</ansi>`, sys))
			} else {
				user.SendText(fmt.Sprintf(`  %s`, sys))
			}
		}
		return true, nil

	default:
		user.SendText(fmt.Sprintf(`Unknown combat subcommand: %s`, parts[0]))
		user.SendText(`Valid subcommands: system, systems`)
		return true, nil
	}
}
