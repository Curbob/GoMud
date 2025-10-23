# GoMud Combat System Context

## Overview

The GoMud combat system provides comprehensive turn-based combat mechanics with support for player vs player, player vs mob, and mob vs mob encounters. It features sophisticated damage calculations, dual wielding, critical hits, backstab mechanics, pet participation, alignment-based consequences, and detailed combat messaging with cross-room attack support.

## Architecture

The combat system is built around several key components:

### Core Components

**Combat Resolution Engine:**
- Turn-based combat with speed-based attack frequency
- Multi-attack system based on speed differentials
- Weapon-based damage calculations with racial bonuses
- Defense reduction and damage mitigation
- Critical hit system with buff effects

**Attack Result System:**
- Comprehensive result tracking for damage, hits, and effects
- Multi-target messaging system for attacker, defender, and rooms
- Support for cross-room combat with directional messaging
- Buff application tracking for combat effects

**Combat Calculations:**
- Hit chance calculations based on speed statistics
- Critical hit probability with level and stat modifiers
- Damage reduction through defense statistics
- Power ranking system for combat assessment
- Alignment change calculations for PvP consequences

## Key Features

### 1. **Multi-Type Combat Support**
- Player vs Mob combat with damage tracking
- Player vs Player combat with alignment consequences
- Mob vs Player combat with AI integration
- Mob vs Mob combat with charm attribution

### 2. **Advanced Combat Mechanics**
- Speed-based multiple attacks per round
- Dual wielding with skill-based penalties
- Backstab mechanics with guaranteed critical hits
- Pet participation in combat (20% chance)
- Cross-room combat support with directional messaging

### 3. **Weapon and Equipment Integration**
- Weapon-specific damage dice and bonuses
- Racial weapon preferences and unarmed combat
- Equipment-based defense calculations
- Weapon subtype messaging and effects
- Stat modification integration

### 4. **Combat Messaging System**
- Dynamic message selection based on damage percentage
- Token-based message customization
- Separate messaging for same-room vs cross-room combat
- Critical hit and backstab message highlighting
- Damage reduction feedback

## Combat Structure

### Attack Result Data Structure
```go
type AttackResult struct {
    Hit                     bool     // Whether the attack connected
    Crit                    bool     // Whether it was a critical hit
    BuffSource              []int    // Buffs applied to attacker
    BuffTarget              []int    // Buffs applied to target
    DamageToTarget          int      // Total damage dealt to target
    DamageToTargetReduction int      // Damage blocked by target's defense
    DamageToSource          int      // Damage dealt to attacker (rare)
    DamageToSourceReduction int      // Damage blocked by attacker's defense
    MessagesToSource        []string // Messages sent to attacker
    MessagesToTarget        []string // Messages sent to target
    MessagesToSourceRoom    []string // Messages sent to attacker's room
    MessagesToTargetRoom    []string // Messages sent to target's room
}
```

### Combat Type Enumeration
```go
type SourceTarget string

const (
    User SourceTarget = "user"  // Player character
    Mob  SourceTarget = "mob"   // NPC character
)
```

## Combat Resolution System

### Player vs Mob Combat
```go
// Main combat function for player attacking mob
func AttackPlayerVsMob(user *users.UserRecord, mob *mobs.Mob) AttackResult {
    attackResult := calculateCombat(*user.Character, mob.Character, User, Mob)
    
    // Apply damage to attacker if any
    if attackResult.DamageToSource != 0 {
        user.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
        user.WimpyCheck() // Check if player should flee
    }
    
    // Apply damage to target
    mob.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
    
    // Track damage for loot distribution
    mob.Character.TrackPlayerDamage(user.UserId, attackResult.DamageToTarget)
    
    // Play appropriate sound effects
    if attackResult.Hit {
        user.PlaySound("hit-other", "combat")
    } else {
        user.PlaySound("miss", "combat")
    }
    
    return attackResult
}
```

### Player vs Player Combat
```go
// PvP combat with alignment consequences
func AttackPlayerVsPlayer(userAtk *users.UserRecord, userDef *users.UserRecord) AttackResult {
    attackResult := calculateCombat(*userAtk.Character, *userDef.Character, User, User)
    
    // Apply damage to both participants
    if attackResult.DamageToSource != 0 {
        userAtk.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
        userAtk.WimpyCheck()
    }
    
    if attackResult.DamageToTarget != 0 {
        userDef.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
        userDef.WimpyCheck()
    }
    
    // Play sound effects for both players
    if attackResult.Hit {
        userAtk.PlaySound("hit-other", "combat")
        userDef.PlaySound("hit-self", "combat")
    } else {
        userAtk.PlaySound("miss", "combat")
    }
    
    return attackResult
}
```

### Mob vs Player Combat
```go
// NPC attacking player with AI integration
func AttackMobVsPlayer(mob *mobs.Mob, user *users.UserRecord) AttackResult {
    attackResult := calculateCombat(mob.Character, *user.Character, Mob, User)
    
    // Apply damage to mob (rare, usually from weapon effects)
    mob.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
    
    // Apply damage to player
    if attackResult.DamageToTarget != 0 {
        user.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
        user.WimpyCheck()
    }
    
    // Player hears when they're hit
    if attackResult.Hit {
        user.PlaySound("hit-self", "combat")
    }
    
    return attackResult
}
```

### Mob vs Mob Combat
```go
// NPC vs NPC combat with charm attribution
func AttackMobVsMob(mobAtk *mobs.Mob, mobDef *mobs.Mob) AttackResult {
    attackResult := calculateCombat(mobAtk.Character, mobDef.Character, Mob, User)
    
    // Apply damage to both mobs
    mobAtk.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
    mobDef.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
    
    // If attacking mob is charmed, attribute damage to controlling player
    if charmedUserId := mobAtk.Character.GetCharmedUserId(); charmedUserId > 0 {
        mobDef.Character.TrackPlayerDamage(charmedUserId, attackResult.DamageToTarget)
    }
    
    return attackResult
}
```

## Combat Calculation Engine

### Main Combat Resolution
```go
func calculateCombat(sourceChar characters.Character, targetChar characters.Character, 
                    sourceType SourceTarget, targetType SourceTarget) AttackResult {
    
    attackResult := AttackResult{}
    
    // Calculate number of attacks based on speed differential
    attackCount := int(math.Ceil(float64(sourceChar.Stats.Speed.ValueAdj-targetChar.Stats.Speed.ValueAdj) / 25))
    if attackCount < 1 {
        attackCount = 1
    }
    
    // Add stat modification bonuses
    statModDBonus := sourceChar.StatMod("damage")
    attackCount += sourceChar.StatMod("attacks")
    
    // Process each attack
    for i := 0; i < attackCount; i++ {
        // Determine weapons to use
        attackWeapons := getAttackWeapons(sourceChar)
        
        // Handle backstab mechanics
        if sourceChar.Aggro.Type == characters.BackStab {
            attackResult.Crit = true
            attackMessagePrefix = "<ansi fg=\"magenta-bold\">*[BACKSTAB]*</ansi> "
            sourceChar.SetAggro(sourceChar.Aggro.UserId, sourceChar.Aggro.MobInstanceId, characters.DefaultAttack)
        }
        
        // Process each weapon attack
        for _, weapon := range attackWeapons {
            processWeaponAttack(weapon, &attackResult, sourceChar, targetChar, sourceType, targetType)
        }
        
        // Pet participation (20% chance)
        if util.RollDice(1, 5) == 1 && sourceChar.Pet.Exists() {
            processPetAttack(sourceChar, targetChar, &attackResult, sourceType, targetType)
        }
    }
    
    return attackResult
}
```

### Hit Calculation System
```go
// Calculate hit chance based on speed statistics
func hitChance(attackSpd, defendSpd int) int {
    atkPlusDef := float64(attackSpd + defendSpd)
    if atkPlusDef < 1 {
        atkPlusDef = 1
    }
    return 30 + int(float64(attackSpd)/atkPlusDef*70) // 30-100% base hit chance
}

// Determine if attack hits with modifiers
func Hits(attackSpd, defendSpd, hitModifier int) bool {
    toHit := hitChance(attackSpd, defendSpd)
    if hitModifier != 0 {
        toHit += hitModifier
    }
    
    // Clamp hit chance between 5% and 95%
    if toHit < 5 {
        toHit = 5
    }
    if toHit > 95 {
        toHit = 95
    }
    
    hitRoll := util.Rand(100)
    util.LogRoll("Hits", hitRoll, toHit)
    
    return hitRoll < toHit
}
```

### Critical Hit System
```go
// Calculate critical hit probability
func Crits(sourceChar characters.Character, targetChar characters.Character) bool {
    levelDiff := sourceChar.Level - targetChar.Level
    if levelDiff < 1 {
        levelDiff = 1
    }
    
    // Base crit chance: 5% + (strength + speed) / level difference
    critChance := 5 + int(math.Round(float64(sourceChar.Stats.Strength.ValueAdj+sourceChar.Stats.Speed.ValueAdj)/float64(levelDiff)))
    
    // Buff modifications
    if sourceChar.HasBuffFlag(buffs.Accuracy) {
        critChance *= 2 // Double crit chance with Accuracy buff
    }
    
    if targetChar.HasBuffFlag(buffs.Blink) {
        critChance /= 2 // Half crit chance against Blink buff
    }
    
    // Minimum 5% crit chance
    if critChance < 5 {
        critChance = 5
    }
    
    critRoll := util.Rand(100)
    util.LogRoll("Crits", critRoll, critChance)
    
    return critRoll < critChance
}
```

## Dual Wielding System

### Weapon Selection and Penalties
```go
// Determine weapons available for attack
func getAttackWeapons(sourceChar characters.Character) []items.Item {
    attackWeapons := []items.Item{}
    dualWieldLevel := sourceChar.GetSkillLevel(skills.DualWield)
    
    // Add primary weapon
    if sourceChar.Equipment.Weapon.ItemId > 0 {
        attackWeapons = append(attackWeapons, sourceChar.Equipment.Weapon)
    }
    
    // Add offhand weapon if it's a weapon type
    if sourceChar.Equipment.Offhand.ItemId > 0 && 
       sourceChar.Equipment.Offhand.GetSpec().Type == items.Weapon {
        attackWeapons = append(attackWeapons, sourceChar.Equipment.Offhand)
    }
    
    // Default to unarmed if no weapons
    if len(attackWeapons) == 0 {
        attackWeapons = append(attackWeapons, items.Item{ItemId: 0})
    }
    
    // Apply dual wield skill restrictions
    if len(attackWeapons) > 1 {
        maxWeapons := 1
        
        if dualWieldLevel == 2 {
            // 50% chance to use both weapons at level 2
            if util.Rand(100) < 50 {
                maxWeapons = 2
            }
        } else if dualWieldLevel >= 3 {
            maxWeapons = 2 // Always dual wield at level 3+
        }
        
        // Special case: martial weapons (claws) can always dual wield
        if sourceChar.Equipment.Weapon.GetSpec().Subtype == items.Claws && 
           sourceChar.Equipment.Offhand.GetSpec().Subtype == items.Claws {
            maxWeapons = 2
        }
        
        // Remove excess weapons randomly
        for len(attackWeapons) > maxWeapons {
            rnd := util.Rand(len(attackWeapons))
            attackWeapons = append(attackWeapons[:rnd], attackWeapons[rnd+1:]...)
        }
    }
    
    return attackWeapons
}

// Calculate dual wield penalty
func getDualWieldPenalty(weaponCount int, dualWieldLevel int) int {
    penalty := 0
    if weaponCount > 1 {
        if dualWieldLevel < 4 {
            penalty = 35 // 35% penalty to hit
        } else {
            penalty = 25 // 25% penalty to hit at mastery level
        }
    }
    return penalty
}
```

## Combat Messaging System

### Message Token System
```go
// Token replacement for dynamic combat messages
func buildTokenReplacements(sourceChar, targetChar characters.Character, 
                          sourceType, targetType SourceTarget, 
                          weaponName string, damage int) map[items.TokenName]string {
    
    tokenReplacements := map[items.TokenName]string{
        items.TokenItemName:     weaponName,
        items.TokenSource:       sourceChar.Name,
        items.TokenSourceType:   string(sourceType) + "name",
        items.TokenTarget:       targetChar.Name,
        items.TokenTargetType:   string(targetType) + "name",
        items.TokenDamage:       strconv.Itoa(damage),
        items.TokenEntranceName: "unknown",
        items.TokenExitName:     "unknown",
    }
    
    // Use mob display names for NPCs
    if sourceType == Mob {
        tokenReplacements[items.TokenSource] = sourceChar.GetMobName(0).String()
    }
    
    if targetType == Mob {
        tokenReplacements[items.TokenTarget] = targetChar.GetMobName(0).String()
    }
    
    return tokenReplacements
}
```

### Cross-Room Combat Messaging
```go
// Handle messaging for cross-room combat
func handleCrossRoomMessages(sourceChar, targetChar characters.Character, 
                           attackResult *AttackResult, msgs items.AttackMessages) {
    
    if sourceChar.RoomId == targetChar.RoomId {
        // Same room combat
        attackResult.SendToSource(string(msgs.Together.ToAttacker))
        attackResult.SendToTarget(string(msgs.Together.ToDefender))
        attackResult.SendToSourceRoom(string(msgs.Together.ToRoom))
    } else {
        // Cross-room combat with directional information
        attackResult.SendToSource(string(msgs.Separate.ToAttacker))
        attackResult.SendToTarget(string(msgs.Separate.ToDefender))
        attackResult.SendToSourceRoom(string(msgs.Separate.ToAttackerRoom))
        attackResult.SendToTargetRoom(string(msgs.Separate.ToDefenderRoom))
        
        // Add directional information
        addDirectionalTokens(sourceChar, targetChar, tokenReplacements)
    }
}

// Add exit/entrance direction tokens for cross-room combat
func addDirectionalTokens(sourceChar, targetChar characters.Character, 
                         tokens map[items.TokenName]string) {
    
    // Find exit from source to target
    if atkRoom := rooms.LoadRoom(sourceChar.RoomId); atkRoom != nil {
        for exitName, exit := range atkRoom.Exits {
            if exit.RoomId == targetChar.RoomId {
                tokens[items.TokenExitName] = exitName
                break
            }
        }
    }
    
    // Find entrance from target to source
    if defRoom := rooms.LoadRoom(targetChar.RoomId); defRoom != nil {
        for exitName, exit := range defRoom.Exits {
            if exit.RoomId == sourceChar.RoomId {
                tokens[items.TokenEntranceName] = exitName
                break
            }
        }
    }
}
```

## Combat Calculations and Utilities

### Power Ranking System
```go
// Calculate relative combat power between characters
func PowerRanking(atkChar characters.Character, defChar characters.Character) float64 {
    // Calculate potential damage output
    attacks, dCount, dSides, dBonus, _ := atkChar.Equipment.Weapon.GetDiceRoll()
    atkDmg := attacks * (dCount*dSides + dBonus)
    
    attacks, dCount, dSides, dBonus, _ = defChar.Equipment.Weapon.GetDiceRoll()
    defDmg := attacks * (dCount*dSides + dBonus)
    
    pct := 0.0
    
    // Damage comparison (40% weight)
    if defDmg == 0 {
        pct += 0.4
    } else {
        pct += 0.4 * float64(atkDmg) / float64(defDmg)
    }
    
    // Speed comparison (30% weight)
    if defChar.Stats.Speed.ValueAdj == 0 {
        pct += 0.3
    } else {
        pct += 0.3 * float64(atkChar.Stats.Speed.ValueAdj) / float64(defChar.Stats.Speed.ValueAdj)
    }
    
    // Health comparison (20% weight)
    if defChar.HealthMax.Value == 0 {
        pct += 0.2
    } else {
        pct += 0.2 * float64(atkChar.HealthMax.Value) / float64(defChar.HealthMax.Value)
    }
    
    // Defense comparison (10% weight)
    if defChar.GetDefense() == 0 {
        pct += 0.1
    } else {
        pct += 0.1 * float64(atkChar.GetDefense()) / float64(defChar.GetDefense())
    }
    
    return pct
}
```

### Taming Mechanics
```go
// Calculate chance to tame a mob
func ChanceToTame(s *users.UserRecord, t *mobs.Mob) int {
    const (
        MOD_SKILL_MIN    = 1    // Minimum base tame ability
        MOD_SKILL_MAX    = 100  // Maximum base tame ability
        MOD_SIZE_SMALL   = 0    // Modifier for small creatures
        MOD_SIZE_MEDIUM  = -10  // Modifier for medium creatures
        MOD_SIZE_LARGE   = -25  // Modifier for large creatures
        MOD_LEVELDIFF_MIN = -25 // Lowest level delta modifier
        MOD_LEVELDIFF_MAX = 25  // Highest level delta modifier
        MOD_HEALTHPERCENT_MAX = 50.0 // Max bonus for reduced target HP
        FACTOR_IS_AGGRO  = 0.50 // Reduction if target is aggressive
    )
    
    // Base proficiency with this mob type
    proficiencyModifier := s.Character.MobMastery.GetTame(int(t.MobId))
    proficiencyModifier = clamp(proficiencyModifier, MOD_SKILL_MIN, MOD_SKILL_MAX)
    
    // Size-based difficulty
    raceInfo := races.GetRace(s.Character.RaceId)
    sizeModifier := 0
    switch raceInfo.Size {
    case races.Large:
        sizeModifier = MOD_SIZE_LARGE
    case races.Small:
        sizeModifier = MOD_SIZE_SMALL
    default: // Medium
        sizeModifier = MOD_SIZE_MEDIUM
    }
    
    // Level difference bonus/penalty
    levelDiff := s.Character.Level - t.Character.Level
    levelDiff = clamp(levelDiff, MOD_LEVELDIFF_MIN, MOD_LEVELDIFF_MAX)
    
    // Health-based bonus (lower health = easier to tame)
    healthModifier := MOD_HEALTHPERCENT_MAX - 
        math.Ceil(float64(s.Character.Health)/float64(s.Character.HealthMax.Value)*MOD_HEALTHPERCENT_MAX)
    
    // Aggro penalty
    aggroModifier := 1.0
    if t.Character.IsAggro(s.UserId, 0) {
        aggroModifier = FACTOR_IS_AGGRO
    }
    
    // Calculate final tame chance
    baseChance := float64(proficiencyModifier) + float64(levelDiff) + healthModifier + float64(sizeModifier)
    return int(math.Ceil(baseChance * aggroModifier))
}
```

### Alignment Change System
```go
// Calculate alignment change from PvP combat
func AlignmentChange(killerAlignment int8, killedAlignment int8) int {
    isKillerGood := killerAlignment > characters.AlignmentNeutralHigh
    isKillerEvil := killerAlignment < characters.AlignmentNeutralLow
    isKillerNeutral := killerAlignment >= characters.AlignmentNeutralLow && 
                       killerAlignment <= characters.AlignmentNeutralHigh
    
    isKilledGood := killedAlignment > characters.AlignmentNeutralHigh
    isKilledEvil := killedAlignment < characters.AlignmentNeutralLow
    isKilledNeutral := killedAlignment >= characters.AlignmentNeutralLow && 
                       killedAlignment <= characters.AlignmentNeutralHigh
    
    // Calculate alignment delta (0-100 scale)
    deltaAbs := math.Abs(math.Max(float64(killerAlignment), float64(killedAlignment)) - 
                        math.Min(float64(killerAlignment), float64(killedAlignment))) * 0.5
    
    // Determine change magnitude based on alignment difference
    changeAmt := 0
    if deltaAbs <= 10 {
        changeAmt = 0
    } else if deltaAbs <= 30 {
        changeAmt = 1
    } else if deltaAbs <= 60 {
        changeAmt = 2
    } else if deltaAbs <= 80 {
        changeAmt = 3
    } else {
        changeAmt = 4
    }
    
    // Calculate direction factor
    factor := 0
    
    if isKillerGood {
        if isKilledGood { // Good vs Good = especially evil
            factor = -2
            changeAmt = int(math.Max(float64(changeAmt), 1))
        } else if isKilledEvil { // Good vs Evil = good act
            factor = 1
        } else if isKilledNeutral { // Good vs Neutral = evil act
            factor = -1
        }
    } else if isKillerEvil {
        if isKilledGood { // Evil vs Good = evil act
            factor = -1
        } else if isKilledEvil { // Evil vs Evil = especially good
            factor = 2
            changeAmt = int(math.Max(float64(changeAmt), 1))
        } else if isKilledNeutral { // Evil vs Neutral = evil act
            factor = -1
        }
    } else if isKillerNeutral {
        if isKilledGood { // Neutral vs Good = evil act
            factor = -1
        } else if isKilledEvil { // Neutral vs Evil = good act
            factor = 1
        } else if isKilledNeutral { // Neutral vs Neutral = no change
            factor = 0
        }
    }
    
    return factor * changeAmt
}
```

## Pet Combat System

### Pet Participation
```go
// Handle pet joining combat (20% chance per round)
func processPetAttack(sourceChar characters.Character, targetChar characters.Character, 
                     attackResult *AttackResult, sourceType, targetType SourceTarget) {
    
    if sourceChar.RoomId != targetChar.RoomId {
        return // Pets only fight in same room
    }
    
    if !sourceChar.Pet.Exists() || sourceChar.Pet.Damage.DiceRoll == "" {
        return // No pet or pet has no combat ability
    }
    
    attacks, dCount, dSides, dBonus, _ := sourceChar.Pet.GetDiceRoll()
    
    for i := 0; i < attacks; i++ {
        attackTargetDamage := util.RollDice(dCount, dSides) + dBonus
        attackResult.DamageToTarget += attackTargetDamage
        
        // Send pet attack messages
        petName := sourceChar.Pet.DisplayName()
        
        toAttackerMsg := fmt.Sprintf("%s jumps into the fray and deals <ansi fg=\"damage\">%d damage</ansi> to <ansi fg=\"%sname\">%s</ansi>!", 
                                   petName, attackTargetDamage, string(targetType), targetChar.Name)
        attackResult.SendToSource(toAttackerMsg)
        
        toDefenderMsg := fmt.Sprintf("%s jumps into the fray and deals <ansi fg=\"damage\">%d damage</ansi> to you!", 
                                   petName, attackTargetDamage)
        attackResult.SendToTarget(toDefenderMsg)
        
        toRoomMsg := fmt.Sprintf("%s jumps into the fray and deals <ansi fg=\"damage\">%d damage</ansi> to <ansi fg=\"%sname\">%s</ansi>!", 
                               petName, attackTargetDamage, string(targetType), targetChar.Name)
        attackResult.SendToTargetRoom(toRoomMsg)
    }
}
```

## Integration Patterns

### Character System Integration
```go
// Combat integrates deeply with character stats and equipment
- sourceChar.Stats.Speed.ValueAdj  // Speed for hit chance and attack frequency
- sourceChar.Stats.Strength.ValueAdj  // Strength for critical hit chance
- sourceChar.Equipment.Weapon.GetDiceRoll()  // Weapon damage calculations
- sourceChar.GetDefense()  // Defense for damage reduction
- sourceChar.StatMod("damage")  // Stat modifications for bonus damage
- sourceChar.HasBuffFlag(buffs.Accuracy)  // Buff effects on combat
```

### Event System Integration
```go
// Combat results trigger various game events
- user.WimpyCheck()  // Automatic flee on low health
- mob.Character.TrackPlayerDamage()  // Damage tracking for loot distribution
- user.PlaySound()  // Audio feedback for combat actions
- sourceChar.SetAggro()  // Aggression state management
```

### Item System Integration
```go
// Weapons provide combat capabilities and messaging
- weapon.GetDiceRoll()  // Damage calculation from weapon stats
- weapon.StatMod()  // Racial bonuses and special modifiers
- items.GetAttackMessage()  // Dynamic combat messaging
- weapon.DisplayName()  // Weapon identification in messages
```

## Usage Examples

### Basic Combat Initiation
```go
// Player attacks mob
user := users.GetByUserId(userId)
mob := mobs.GetInstance(mobInstanceId)

if user != nil && mob != nil {
    result := combat.AttackPlayerVsMob(user, mob)
    
    // Send messages to all participants
    for _, msg := range result.MessagesToSource {
        user.SendText(msg)
    }
    
    // Check if mob died
    if mob.Character.Health <= 0 {
        handleMobDeath(mob, user)
    }
}
```

### PvP Combat with Consequences
```go
// Player vs player combat
attacker := users.GetByUserId(attackerId)
defender := users.GetByUserId(defenderId)

result := combat.AttackPlayerVsPlayer(attacker, defender)

// Calculate alignment change
alignmentChange := combat.AlignmentChange(attacker.Character.Alignment, defender.Character.Alignment)
attacker.Character.Alignment += int8(alignmentChange)

// Handle death consequences
if defender.Character.Health <= 0 {
    handlePlayerDeath(defender, attacker)
}
```

### Combat Assessment
```go
// Assess relative combat power
player := users.GetByUserId(userId)
mob := mobs.GetInstance(mobInstanceId)

powerRatio := combat.PowerRanking(player.Character, mob.Character)

if powerRatio > 1.5 {
    player.SendText("This should be an easy fight.")
} else if powerRatio < 0.5 {
    player.SendText("This looks very dangerous!")
} else {
    player.SendText("This should be a fair fight.")
}
```

## Dependencies

- `internal/characters` - Character stats, equipment, and abilities
- `internal/items` - Weapon specifications and combat messaging
- `internal/users` - Player character management and state
- `internal/mobs` - NPC character management and AI integration
- `internal/buffs` - Status effects that modify combat
- `internal/skills` - Skill system for dual wielding and combat abilities
- `internal/races` - Racial bonuses and unarmed combat specifications
- `internal/rooms` - Room management for cross-room combat
- `internal/util` - Dice rolling and random number generation
- `internal/configs` - Configuration for combat behavior and messaging

This comprehensive combat system provides sophisticated turn-based combat mechanics with support for all character types, advanced weapon systems, detailed messaging, and seamless integration with all other game systems.