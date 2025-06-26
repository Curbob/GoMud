package combattwitch

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// registerCommands registers all combat-related commands
func (tc *TwitchCombat) registerCommands() {
	// Register user commands
	tc.plug.AddUserCommand("attack", tc.attackCommand, false, false)
	tc.plug.AddUserCommand("kill", tc.attackCommand, false, false) // Alias for attack
	tc.plug.AddUserCommand("balance", tc.balanceCommand, false, false)
	tc.plug.AddUserCommand("cleartarget", tc.clearTargetCommand, false, false)
	tc.plug.AddUserCommand("target", tc.targetCommand, false, false)
	tc.plug.AddUserCommand("flee", tc.fleeCommand, false, false)
	tc.plug.AddUserCommand("combatinfo", tc.combatInfoCommand, true, true) // Admin only
	tc.plug.AddUserCommand("config", tc.configCommand, true, true)         // Admin only

	// Register mob commands
	tc.plug.AddMobCommand("attack", tc.mobAttackCommand, false)

	mudlog.Info("Combat Commands", "action", "registered", "module", "combat-twitch")
}

// attackCommand handles player attack commands
func (tc *TwitchCombat) attackCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Check balance
	if !tc.timer.CanPerformAction(user.UserId, combat.User) {
		nextAction := tc.timer.GetNextActionTime(user.UserId, combat.User)
		remaining := time.Until(nextAction)
		user.SendText(fmt.Sprintf(`<ansi fg="red">You are unbalanced! (%.1f seconds)</ansi>`, remaining.Seconds()))
		return true, nil
	}

	var attackPlayerId, attackMobInstanceId int
	var targetName string

	// If no target specified, check if we have a persistent target
	if rest == "" {
		// First check if we're in active combat with a specific mob
		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
			// Reuse existing mob target
			attackMobInstanceId = user.Character.Aggro.MobInstanceId
			// Verify the mob is still in the room
			targetMob := mobs.GetInstance(attackMobInstanceId)
			if targetMob == nil || targetMob.Character.RoomId != user.Character.RoomId {
				user.SendText("Your target is no longer here.")
				user.Character.Aggro = nil
				// Try to use stored target name
				targetName = tc.GetUserTarget(user.UserId)
				if targetName != "" {
					attackPlayerId, attackMobInstanceId = room.FindByName(targetName)
				}
			}
		} else {
			// Not in active combat, check for stored target name
			targetName = tc.GetUserTarget(user.UserId)
			if targetName != "" {
				attackPlayerId, attackMobInstanceId = room.FindByName(targetName)
			} else {
				user.SendText("Attack whom?")
				return true, nil
			}
		}
	} else {
		// New target specified
		targetName = rest
		// Find target using the same logic as round-based combat
		attackPlayerId, attackMobInstanceId = room.FindByName(rest)

		// Can't attack self
		if attackPlayerId == user.UserId {
			attackPlayerId = 0
		}
	}

	// Check if we found a target
	if attackMobInstanceId == 0 && attackPlayerId == 0 {
		user.SendText("They aren't here.")
		// Clear stored target if it's no longer valid
		if targetName != "" {
			tc.ClearUserTarget(user.UserId)
		}
		return true, nil
	}

	// For now, twitch combat only supports attacking mobs
	if attackMobInstanceId == 0 {
		user.SendText("You can only attack mobs in twitch combat.")
		return true, nil
	}

	targetMob := mobs.GetInstance(attackMobInstanceId)
	if targetMob == nil {
		user.SendText("They aren't here.")
		return true, nil
	}

	// Store the actual mob name for persistent targeting
	// Always use the mob's actual name, not what the user typed
	tc.SetUserTarget(user.UserId, targetMob.Character.Name)

	// Register the player with the timer if not already registered
	tc.timer.RegisterActor(user.UserId, combat.User)

	// Set combat state first
	user.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)

	// Perform the attack and get results
	attackResult := combat.AttackPlayerVsMob(user, targetMob)

	// Send combat messages
	// Send messages to attacker
	for _, msg := range attackResult.MessagesToSource {
		user.SendText(msg)
	}

	// Send messages to room
	for _, msg := range attackResult.MessagesToSourceRoom {
		room.SendText(msg, user.UserId)
	}

	// Calculate weapon cooldown
	cooldown := tc.calculateWeaponCooldown(user.Character, false) // false = player
	tc.timer.SetActorCooldown(user.UserId, combat.User, cooldown)

	// Display balance info
	user.SendText(fmt.Sprintf(`<ansi fg="yellow">[Balance: %.1fs]</ansi>`, cooldown.Seconds()))

	// Send initial GMCP balance update
	tc.SendGMCPBalanceUpdate(user.UserId, cooldown.Seconds(), cooldown.Seconds())

	// Check if mob died
	if targetMob.Character.Health <= 0 {
		targetMob.Character.EndAggro()
		user.Character.Aggro = nil
	} else if targetMob.Character.Aggro == nil {
		// Mob retaliates
		targetMob.PreventIdle = true
		targetMob.Command(fmt.Sprintf("attack @%d", user.UserId))
	}

	return true, nil
}

// balanceCommand shows current balance status
func (tc *TwitchCombat) balanceCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	nextAction := tc.timer.GetNextActionTime(user.UserId, combat.User)
	now := time.Now()

	if now.Before(nextAction) {
		remaining := time.Until(nextAction)
		user.SendText(fmt.Sprintf(`<ansi fg="yellow">You are unbalanced: %.1f seconds remaining</ansi>`, remaining.Seconds()))
	} else {
		user.SendText(`<ansi fg="green">You are balanced and ready to act!</ansi>`)
	}

	// Show weapon speed info
	if user.Character.Equipment.Weapon.ItemId > 0 {
		weaponSpec := user.Character.Equipment.Weapon.GetSpec()
		cooldown := tc.calculateWeaponCooldown(user.Character, false) // false = player
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Weapon Recovery Time: %.1fs</ansi>`, cooldown.Seconds()))

		if weaponSpec.WaitRounds > 0 {
			user.SendText(fmt.Sprintf(`<ansi fg="cyan">Base Wait: %d rounds (in round-based combat)</ansi>`, weaponSpec.WaitRounds))
		}
	} else {
		user.SendText(`<ansi fg="cyan">Unarmed Recovery Time: 2.5s</ansi>`)
	}

	return true, nil
}

// mobAttackCommand handles mob attack commands
func (tc *TwitchCombat) mobAttackCommand(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	// Check cooldown
	if !tc.timer.CanPerformAction(mob.InstanceId, combat.Mob) {
		// Mobs silently wait
		return true, nil
	}

	attackPlayerId := 0

	// Parse target - simplified for now, only handles @userId syntax
	if len(rest) > 1 && rest[0] == '@' {
		var userId int
		if _, err := fmt.Sscanf(rest[1:], "%d", &userId); err == nil && userId > 0 {
			attackPlayerId = userId
		}
	}

	if attackPlayerId == 0 {
		return true, nil
	}

	// Get target user
	targetUser := users.GetByUserId(attackPlayerId)
	if targetUser == nil || targetUser.Character.RoomId != mob.Character.RoomId {
		return true, nil
	}

	// Set combat state
	mob.Character.SetAggro(attackPlayerId, 0, characters.DefaultAttack)

	// Perform attack
	attackResult := combat.AttackMobVsPlayer(mob, targetUser)

	// Send combat messages
	for _, msg := range attackResult.MessagesToTarget {
		targetUser.SendText(msg)
	}

	for _, msg := range attackResult.MessagesToSourceRoom {
		room.SendText(msg, attackPlayerId)
	}

	// Calculate and set cooldown
	cooldown := tc.calculateWeaponCooldown(&mob.Character, true) // true = mob
	tc.timer.SetActorCooldown(mob.InstanceId, combat.Mob, cooldown)

	// Send GMCP update to the player showing updated HP
	tc.SendCombatUpdate(attackPlayerId)

	// Check if player died
	if targetUser.Character.Health <= 0 {
		mob.Character.EndAggro()
	} else if mob.Character.Aggro != nil {
		// Set callback for next attack if mob is still in combat
		tc.timer.SetActorCallback(mob.InstanceId, combat.Mob, cooldown, func() {
			// Make sure mob still exists and has aggro
			if m := mobs.GetInstance(mob.InstanceId); m != nil && m.Character.Aggro != nil {
				m.Command(fmt.Sprintf("attack @%d", m.Character.Aggro.UserId))
			}
		})
	}

	return true, nil
}

// calculateWeaponCooldown determines cooldown based on weapon
func (tc *TwitchCombat) calculateWeaponCooldown(char *characters.Character, isMob bool) time.Duration {
	c := configs.GetConfig()
	baseRoundMs := float64(c.Timing.RoundSeconds) * 1000

	// Get config values from main config
	gameConfig := configs.GetGamePlayConfig()

	var armedBaseMs, unarmedBaseMs, maxSpeedReduction, speedDivisor float64

	if isMob {
		// Use mob-specific settings
		armedBaseMs = float64(gameConfig.Combat.TwitchMobArmedBaseMs)
		unarmedBaseMs = float64(gameConfig.Combat.TwitchMobUnarmedBaseMs)
		maxSpeedReduction = float64(gameConfig.Combat.TwitchMobMaxSpeedReduction) / 100.0
		speedDivisor = float64(gameConfig.Combat.TwitchMobSpeedDivisor)
	} else {
		// Use player settings
		armedBaseMs = float64(gameConfig.Combat.TwitchArmedBaseMs)
		unarmedBaseMs = float64(gameConfig.Combat.TwitchUnarmedBaseMs)
		maxSpeedReduction = float64(gameConfig.Combat.TwitchMaxSpeedReduction) / 100.0
		speedDivisor = float64(gameConfig.Combat.TwitchSpeedDivisor)
	}

	if char.Equipment.Weapon.ItemId > 0 {
		weaponSpec := char.Equipment.Weapon.GetSpec()

		// Convert wait rounds to cooldown
		waitRounds := float64(weaponSpec.WaitRounds)
		if waitRounds < 0 {
			waitRounds = 0
		}

		// Base cooldown is armed base + wait rounds * round time
		cooldownMs := armedBaseMs + (waitRounds * baseRoundMs)

		// Speed stat reduces cooldown
		speedBonus := float64(char.Stats.Speed.ValueAdj) / speedDivisor
		if speedBonus > maxSpeedReduction {
			speedBonus = maxSpeedReduction
		}
		cooldownMs *= (1.0 - speedBonus)

		return time.Duration(cooldownMs) * time.Millisecond
	}

	// Unarmed combat
	cooldownMs := unarmedBaseMs

	// Speed bonus for unarmed
	speedBonus := float64(char.Stats.Speed.ValueAdj) / speedDivisor
	if speedBonus > maxSpeedReduction {
		speedBonus = maxSpeedReduction
	}
	cooldownMs *= (1.0 - speedBonus)

	return time.Duration(cooldownMs) * time.Millisecond
}

// combatInfoCommand shows detailed combat calculations for a target
func (tc *TwitchCombat) combatInfoCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
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

	// Get config values from main config
	gameConfig := configs.GetGamePlayConfig()

	// Use appropriate settings based on whether it's a mob or player
	var speedDivisor, maxSpeedReduction, armedBaseMs, unarmedBaseMs float64
	if !isPlayer {
		// Use mob settings
		speedDivisor = float64(gameConfig.Combat.TwitchMobSpeedDivisor)
		maxSpeedReduction = float64(gameConfig.Combat.TwitchMobMaxSpeedReduction)
		armedBaseMs = float64(gameConfig.Combat.TwitchMobArmedBaseMs)
		unarmedBaseMs = float64(gameConfig.Combat.TwitchMobUnarmedBaseMs)
	} else {
		// Use player settings
		speedDivisor = float64(gameConfig.Combat.TwitchSpeedDivisor)
		maxSpeedReduction = float64(gameConfig.Combat.TwitchMaxSpeedReduction)
		armedBaseMs = float64(gameConfig.Combat.TwitchArmedBaseMs)
		unarmedBaseMs = float64(gameConfig.Combat.TwitchUnarmedBaseMs)
	}

	// Speed stats
	user.SendText("")
	user.SendText(`<ansi fg="yellow">── Speed Statistics ──</ansi>`)
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Speed Base:</ansi>      %d`, targetChar.Stats.Speed.Value))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Speed Adjusted:</ansi>  %d`, targetChar.Stats.Speed.ValueAdj))
	speedBonus := float64(targetChar.Stats.Speed.ValueAdj) / speedDivisor
	if speedBonus > maxSpeedReduction/100.0 {
		speedBonus = maxSpeedReduction / 100.0
	}
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Speed Bonus:</ansi>     %.1f%% (max %.0f%%)`, speedBonus*100, maxSpeedReduction))

	// Weapon info
	user.SendText("")
	user.SendText(`<ansi fg="yellow">── Weapon Information ──</ansi>`)

	if targetChar.Equipment.Weapon.ItemId > 0 {
		weaponSpec := targetChar.Equipment.Weapon.GetSpec()
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Weapon:</ansi>          %s`, weaponSpec.Name))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Damage:</ansi>          %s`, weaponSpec.Damage.DiceRoll))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Wait Rounds:</ansi>     %d`, weaponSpec.WaitRounds))

		// Calculate weapon cooldown
		waitRounds := float64(weaponSpec.WaitRounds)
		if waitRounds < 0 {
			waitRounds = 0
		}
		baseCooldownMs := armedBaseMs + (waitRounds * float64(baseRoundSeconds) * 1000)
		finalCooldownMs := baseCooldownMs * (1.0 - speedBonus)

		user.SendText("")
		user.SendText(`<ansi fg="yellow">── Weapon Cooldown Calculation ──</ansi>`)
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Base:</ansi>            %.1f seconds`, armedBaseMs/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Wait penalty:</ansi>    %.1f seconds (%d rounds × %d sec/round)`,
			waitRounds*float64(baseRoundSeconds), int(waitRounds), baseRoundSeconds))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Subtotal:</ansi>        %.1f seconds`, baseCooldownMs/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Speed reduction:</ansi> -%.1f%% (%.1f seconds)`,
			speedBonus*100, (baseCooldownMs*speedBonus)/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="green">Final cooldown:</ansi>  %.1f seconds</ansi>`, finalCooldownMs/1000))

	} else {
		user.SendText(`<ansi fg="cyan">Weapon:</ansi>          None (unarmed)`)

		// Calculate unarmed cooldown
		baseCooldownMs := unarmedBaseMs
		finalCooldownMs := baseCooldownMs * (1.0 - speedBonus)

		user.SendText("")
		user.SendText(`<ansi fg="yellow">── Unarmed Cooldown Calculation ──</ansi>`)
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Base:</ansi>            %.1f seconds`, unarmedBaseMs/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Speed reduction:</ansi> -%.1f%% (%.1f seconds)`,
			speedBonus*100, (baseCooldownMs*speedBonus)/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="green">Final cooldown:</ansi>  %.1f seconds</ansi>`, finalCooldownMs/1000))
	}

	// Combat style
	user.SendText("")
	user.SendText(`<ansi fg="yellow">── Combat System ──</ansi>`)
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Active System:</ansi>   %s`, combat.GetActiveCombatSystemName()))
	user.SendText(fmt.Sprintf(`<ansi fg="cyan">Round Duration:</ansi>  %d seconds`, baseRoundSeconds))

	return true, nil
}

// configCommand handles combat configuration
func (tc *TwitchCombat) configCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
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
		user.SendText(`  config combat info               - Show current combat settings`)
		user.SendText(`  config combat system <name>      - Switch combat system`)
		user.SendText(`  config combat systems            - List available combat systems`)
		user.SendText(`  config combat <setting> <value>  - Change a combat setting`)
		user.SendText(``)
		user.SendText(`<ansi fg="yellow">Available player settings:</ansi>`)
		user.SendText(`  unarmed_base_ms                  - Base cooldown for unarmed combat (milliseconds)`)
		user.SendText(`  armed_base_ms                    - Base cooldown for armed combat (milliseconds)`)
		user.SendText(`  max_speed_reduction              - Maximum speed reduction percentage (0-100)`)
		user.SendText(`  speed_divisor                    - Divisor for speed bonus calculation`)
		user.SendText(``)
		user.SendText(`<ansi fg="yellow">Available mob settings:</ansi>`)
		user.SendText(`  mob_unarmed_base_ms              - Base cooldown for mob unarmed combat`)
		user.SendText(`  mob_armed_base_ms                - Base cooldown for mob armed combat`)
		user.SendText(`  mob_max_speed_reduction          - Maximum speed reduction for mobs`)
		user.SendText(`  mob_speed_divisor                - Divisor for mob speed bonus`)
		return true, nil
	}

	// Handle subcommands
	switch parts[0] {
	case "info":
		// Show current combat settings with better spacing
		user.SendText(fmt.Sprintf(`<ansi fg="yellow">═══ Current Combat Configuration ═══</ansi>`))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Active System:</ansi>            %s`, combat.GetActiveCombatSystemName()))
		user.SendText(``)
		user.SendText(`<ansi fg="yellow">── Twitch Combat Settings (Players) ──</ansi>`)

		// Get current settings from main config
		gameConfig := configs.GetGamePlayConfig()
		unarmedBaseMs := int(gameConfig.Combat.TwitchUnarmedBaseMs)
		armedBaseMs := int(gameConfig.Combat.TwitchArmedBaseMs)
		maxSpeedReduction := int(gameConfig.Combat.TwitchMaxSpeedReduction)
		speedDivisor := int(gameConfig.Combat.TwitchSpeedDivisor)

		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Unarmed Base:</ansi>            %d ms (%.1f seconds)`, unarmedBaseMs, float64(unarmedBaseMs)/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Armed Base:</ansi>              %d ms (%.1f seconds)`, armedBaseMs, float64(armedBaseMs)/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Max Speed Reduction:</ansi>     %d%%`, maxSpeedReduction))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Speed Divisor:</ansi>           %d (speed bonus = adjusted speed / %d)`, speedDivisor, speedDivisor))

		user.SendText(``)
		user.SendText(`<ansi fg="yellow">── Twitch Combat Settings (Mobs) ──</ansi>`)

		mobUnarmedBaseMs := int(gameConfig.Combat.TwitchMobUnarmedBaseMs)
		mobArmedBaseMs := int(gameConfig.Combat.TwitchMobArmedBaseMs)
		mobMaxSpeedReduction := int(gameConfig.Combat.TwitchMobMaxSpeedReduction)
		mobSpeedDivisor := int(gameConfig.Combat.TwitchMobSpeedDivisor)

		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Mob Unarmed Base:</ansi>        %d ms (%.1f seconds)`, mobUnarmedBaseMs, float64(mobUnarmedBaseMs)/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Mob Armed Base:</ansi>          %d ms (%.1f seconds)`, mobArmedBaseMs, float64(mobArmedBaseMs)/1000))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Mob Max Speed Reduction:</ansi> %d%%`, mobMaxSpeedReduction))
		user.SendText(fmt.Sprintf(`<ansi fg="cyan">Mob Speed Divisor:</ansi>       %d (speed bonus = adjusted speed / %d)`, mobSpeedDivisor, mobSpeedDivisor))

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

	case "unarmed_base_ms", "armed_base_ms", "max_speed_reduction", "speed_divisor",
		"mob_unarmed_base_ms", "mob_armed_base_ms", "mob_max_speed_reduction", "mob_speed_divisor":
		// Set a configuration value
		if len(parts) < 2 {
			user.SendText(fmt.Sprintf(`Usage: config combat %s <value>`, parts[0]))
			return true, nil
		}

		value, err := strconv.Atoi(parts[1])
		if err != nil {
			user.SendText(`<ansi fg="red">Value must be a number</ansi>`)
			return true, nil
		}

		// Validate ranges
		switch parts[0] {
		case "unarmed_base_ms", "armed_base_ms", "mob_unarmed_base_ms", "mob_armed_base_ms":
			if value < 100 || value > 10000 {
				user.SendText(`<ansi fg="red">Cooldown must be between 100 and 10000 milliseconds</ansi>`)
				return true, nil
			}
		case "max_speed_reduction", "mob_max_speed_reduction":
			if value < 0 || value > 100 {
				user.SendText(`<ansi fg="red">Max speed reduction must be between 0 and 100 percent</ansi>`)
				return true, nil
			}
		case "speed_divisor", "mob_speed_divisor":
			if value < 1 || value > 1000 {
				user.SendText(`<ansi fg="red">Speed divisor must be between 1 and 1000</ansi>`)
				return true, nil
			}
		}

		// Map the setting name to the config path
		configPath := ""
		switch parts[0] {
		case "unarmed_base_ms":
			configPath = "GamePlay.Combat.TwitchUnarmedBaseMs"
		case "armed_base_ms":
			configPath = "GamePlay.Combat.TwitchArmedBaseMs"
		case "max_speed_reduction":
			configPath = "GamePlay.Combat.TwitchMaxSpeedReduction"
		case "speed_divisor":
			configPath = "GamePlay.Combat.TwitchSpeedDivisor"
		case "mob_unarmed_base_ms":
			configPath = "GamePlay.Combat.TwitchMobUnarmedBaseMs"
		case "mob_armed_base_ms":
			configPath = "GamePlay.Combat.TwitchMobArmedBaseMs"
		case "mob_max_speed_reduction":
			configPath = "GamePlay.Combat.TwitchMobMaxSpeedReduction"
		case "mob_speed_divisor":
			configPath = "GamePlay.Combat.TwitchMobSpeedDivisor"
		}

		// Set the value in the main config
		if err := configs.SetVal(configPath, strconv.Itoa(value)); err != nil {
			user.SendText(fmt.Sprintf(`<ansi fg="red">Failed to save setting: %s</ansi>`, err.Error()))
			return true, nil
		}

		user.SendText(fmt.Sprintf(`<ansi fg="green">Set %s to %d</ansi>`, parts[0], value))
		return true, nil

	default:
		user.SendText(fmt.Sprintf(`Unknown combat setting: %s`, parts[0]))
		return true, nil
	}
}

// clearTargetCommand clears the stored combat target
func (tc *TwitchCombat) clearTargetCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	tc.ClearUserTarget(user.UserId)
	user.SendText(`<ansi fg="yellow">Combat target cleared.</ansi>`)
	return true, nil
}

// targetCommand shows or sets the combat target
func (tc *TwitchCombat) targetCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if rest == "" {
		// Show current target
		currentTarget := tc.GetUserTarget(user.UserId)
		if currentTarget == "" {
			user.SendText(`<ansi fg="yellow">No target set.</ansi>`)
		} else {
			user.SendText(fmt.Sprintf(`<ansi fg="yellow">Current target: %s</ansi>`, currentTarget))
		}
	} else {
		// Set new target
		// Verify the target exists in the room
		_, mobId := room.FindByName(rest)
		if mobId == 0 {
			user.SendText("They aren't here.")
			return true, nil
		}

		// Get the actual mob to use its real name
		targetMob := mobs.GetInstance(mobId)
		if targetMob == nil {
			user.SendText("They aren't here.")
			return true, nil
		}

		tc.SetUserTarget(user.UserId, targetMob.Character.Name)
		user.SendText(fmt.Sprintf(`<ansi fg="yellow">Target set to: %s</ansi>`, targetMob.Character.Name))
	}
	return true, nil
}

// fleeCommand handles flee attempts in twitch combat
func (tc *TwitchCombat) fleeCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Check if user is in combat
	if user.Character.Aggro == nil {
		user.SendText(`You aren't in combat!`)
		return true, nil
	}

	// Check if still on cooldown from last action
	if !tc.timer.CanPerformAction(user.UserId, combat.User) {
		nextAction := tc.timer.GetNextActionTime(user.UserId, combat.User)
		remaining := time.Until(nextAction)
		user.SendText(fmt.Sprintf(`<ansi fg="red">You are unbalanced! Can't flee for %.1f seconds.</ansi>`, remaining.Seconds()))
		return true, nil
	}

	user.SendText(`You attempt to flee...`)

	// Immediate flee check based on speed stats
	blockedByMob := ""

	// Check each mob in the room that has aggro on the player
	for _, mobId := range room.GetMobs() {
		mob := mobs.GetInstance(mobId)
		if mob == nil || mob.Character.Aggro == nil || mob.Character.Aggro.UserId != user.UserId {
			continue
		}

		// Speed-based flee chance calculation (same as combat-rounds)
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
	if blockedByMob != "" {
		user.SendText(fmt.Sprintf(`<ansi fg="red">%s blocks your escape!</ansi>`, blockedByMob))
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="red">tries to flee, but is blocked by %s!</ansi>`,
				user.Character.Name, blockedByMob),
			user.UserId)

		// Set a short cooldown for failed flee
		tc.timer.SetActorCooldown(user.UserId, combat.User, 2*time.Second)
		tc.SendCombatUpdate(user.UserId)
	} else {
		// Successful flee
		exitName, _ := room.GetRandomExit()
		if exitName == "" {
			user.SendText(`<ansi fg="red">There's nowhere to run!</ansi>`)
			// Set a short cooldown
			tc.timer.SetActorCooldown(user.UserId, combat.User, 2*time.Second)
			tc.SendCombatUpdate(user.UserId)
			return true, nil
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
		tc.ClearUserTarget(user.UserId)
		tc.timer.ClearActorCooldown(user.UserId, combat.User)

		// Notify all mobs that were fighting this player
		for _, mobId := range room.GetMobs() {
			mob := mobs.GetInstance(mobId)
			if mob != nil && mob.Character.Aggro != nil && mob.Character.Aggro.UserId == user.UserId {
				mob.Character.EndAggro()
			}
		}
	}

	return true, nil
}
