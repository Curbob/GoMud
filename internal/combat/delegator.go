package combat

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// These functions delegate to the active combat system
// They maintain the same signatures as the existing combat functions
// to ensure backward compatibility

// AttackPlayerVsMob delegates to the active combat system
func AttackPlayerVsMob(user *users.UserRecord, mob *mobs.Mob) AttackResult {
	system := GetActiveCombatSystem()
	if system == nil {
		// Fallback to existing implementation
		return attackPlayerVsMob(user, mob)
	}

	// For now, use the existing implementation
	// In the future, this will delegate to the combat system
	return attackPlayerVsMob(user, mob)
}

// AttackPlayerVsPlayer delegates to the active combat system
func AttackPlayerVsPlayer(userAtk *users.UserRecord, userDef *users.UserRecord) AttackResult {
	system := GetActiveCombatSystem()
	if system == nil {
		// Fallback to existing implementation
		return attackPlayerVsPlayer(userAtk, userDef)
	}

	// For now, use the existing implementation
	// In the future, this will delegate to the combat system
	return attackPlayerVsPlayer(userAtk, userDef)
}

// AttackMobVsPlayer delegates to the active combat system
func AttackMobVsPlayer(mob *mobs.Mob, user *users.UserRecord) AttackResult {
	system := GetActiveCombatSystem()
	if system == nil {
		// Fallback to existing implementation
		return attackMobVsPlayer(mob, user)
	}

	// For now, use the existing implementation
	// In the future, this will delegate to the combat system
	return attackMobVsPlayer(mob, user)
}

// AttackMobVsMob delegates to the active combat system
func AttackMobVsMob(mobAtk *mobs.Mob, mobDef *mobs.Mob) AttackResult {
	system := GetActiveCombatSystem()
	if system == nil {
		// Fallback to existing implementation
		return attackMobVsMob(mobAtk, mobDef)
	}

	// For now, use the existing implementation
	// In the future, this will delegate to the combat system
	return attackMobVsMob(mobAtk, mobDef)
}
