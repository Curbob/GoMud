package combattwitch

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// registerCommands registers all combat-related commands
func (tc *TwitchCombat) registerCommands() {
	// Register user commands
	tc.plug.AddUserCommand("attack", tc.attackCommand, false, false)
	tc.plug.AddUserCommand("kill", tc.attackCommand, false, false) // Alias for attack
	tc.plug.AddUserCommand("balance", tc.balanceCommand, false, false)

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

	// Basic attack targeting logic (simplified for demo)
	if rest == "" {
		user.SendText("Attack whom?")
		return true, nil
	}

	// Find target using the same logic as round-based combat
	attackPlayerId, attackMobInstanceId := room.FindByName(rest)
	
	// Can't attack self
	if attackPlayerId == user.UserId {
		attackPlayerId = 0
	}
	
	// Check if we found a target
	if attackMobInstanceId == 0 && attackPlayerId == 0 {
		user.SendText("They aren't here.")
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
	cooldown := tc.calculateWeaponCooldown(user.Character)
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
		cooldown := tc.calculateWeaponCooldown(user.Character)
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
	cooldown := tc.calculateWeaponCooldown(&mob.Character)
	tc.timer.SetActorCooldown(mob.InstanceId, combat.Mob, cooldown)

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
func (tc *TwitchCombat) calculateWeaponCooldown(char *characters.Character) time.Duration {
	c := configs.GetConfig()
	baseRoundMs := float64(c.Timing.RoundSeconds) * 1000

	if char.Equipment.Weapon.ItemId > 0 {
		weaponSpec := char.Equipment.Weapon.GetSpec()

		// Convert wait rounds to cooldown
		// WaitRounds 0 = 2 seconds base
		// WaitRounds 1 = 6 seconds (2 + 4)
		// WaitRounds 2 = 10 seconds (2 + 8), etc.
		waitRounds := float64(weaponSpec.WaitRounds)
		if waitRounds < 0 {
			waitRounds = 0
		}

		// Base cooldown is 2 seconds + wait rounds * round time
		cooldownMs := 2000 + (waitRounds * baseRoundMs)

		// Speed stat reduces cooldown (max 50% reduction)
		speedBonus := float64(char.Stats.Speed.ValueAdj) / 200.0
		if speedBonus > 0.5 {
			speedBonus = 0.5
		}
		cooldownMs *= (1.0 - speedBonus)

		return time.Duration(cooldownMs) * time.Millisecond
	}

	// Unarmed combat: 2.5 second base cooldown
	cooldownMs := 2500.0

	// Speed bonus for unarmed
	speedBonus := float64(char.Stats.Speed.ValueAdj) / 200.0
	if speedBonus > 0.5 {
		speedBonus = 0.5
	}
	cooldownMs *= (1.0 - speedBonus)

	return time.Duration(cooldownMs) * time.Millisecond
}
