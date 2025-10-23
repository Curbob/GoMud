# Characters Package Context

## Overview
The `internal/characters` package is the core character system for GoMud, handling both player characters (PCs) and non-player characters (NPCs/mobs). It provides a comprehensive character model with stats, equipment, skills, combat mechanics, and various character states.

## Key Components

### Core Character Structure (`character.go`)
- **Character struct**: The main character entity containing all character data
- **Character creation and management**: Factory functions and lifecycle management
- **Stat calculations**: Dynamic stat computation with buffs, equipment, and racial modifiers
- **Experience and leveling**: Level progression and TNL (To Next Level) calculations
- **Persistence**: Character data serialization/deserialization

### Character Statistics System
- **Six core stats**: Strength, Speed, Smarts, Vitality, Mysticism, Perception
- **Stat scaling**: Stats over 100 use `SQRT(overage)*2` formula for diminishing returns
- **Dynamic modifiers**: Equipment, buffs, and racial bonuses affect final stats
- **Stat points**: Manual allocation points gained per level

### Equipment System (`worn.go`)
- **Equipment slots**: Weapon, Offhand, Head, Neck, Body, Belt, Gloves, Ring, Legs, Feet
- **Stat modifications**: Equipment provides stat bonuses aggregated across all slots
- **Item management**: Worn item tracking and validation

### Character States and Modifiers
- **Alignment system** (`alignment.go`): Good/neutral/evil alignment with numeric values (-100 to +100)
- **Aggro system** (`aggro.go`): Combat targeting and threat management
- **Buffs integration**: Status effects that modify character capabilities
- **Cooldowns** (`cooldowns.go`): Time-based ability restrictions

### Combat and Interaction Systems
- **Kill/Death statistics** (`kdstats.go`): PvP and PvE combat tracking
- **Charm system** (`charminfo.go`): Mind control and pet mechanics
- **Mob mastery** (`mobmastery.go`): Character proficiency with specific creature types
- **Shop system** (`shop.go`): NPC merchant capabilities with restocking mechanics

### Character Presentation
- **Formatted names** (`formattedname.go`): Rich text rendering with adjectives and color coding
- **Adjectives system**: Visual indicators for character states (sleeping, charmed, poisoned, etc.)
- **Quest indicators**: Visual markers for quest-relevant NPCs

## Key Features

### Character Persistence
- YAML-based character data storage
- Automatic saving with configurable intervals
- Character creation timestamps and history tracking
- Room history for movement tracking

### Dynamic Stat System
- Base stats from race definitions
- Equipment stat modifications
- Buff/debuff effects
- Manual stat point allocation
- Calculated maximums for Health, Mana, and Action Points

### Social and Economic Systems
- Gold and banking system
- Player shops and merchant NPCs
- Clan membership support
- Pet ownership and management
- Quest progress tracking

### Combat Integration
- Aggro management for targeting
- Damage tracking between players
- Combat state management
- Weapon and armor effectiveness

### Scripting Integration
- JavaScript-accessible character properties
- Event-driven character updates
- Scriptable character behaviors for NPCs

## Dependencies
- `internal/stats`: Core statistics definitions
- `internal/items`: Item system integration
- `internal/buffs`: Status effect system
- `internal/races`: Character race definitions
- `internal/skills`: Skill system integration
- `internal/spells`: Magic system integration
- `internal/quests`: Quest system integration
- `internal/pets`: Pet system integration
- `internal/gametime`: Time-based mechanics
- `internal/colorpatterns`: Text formatting and colors

## Usage Patterns
- Character creation through factory functions
- Stat calculations via getter methods that apply all modifiers
- Equipment management through worn item slots
- State management through adjectives and flags
- Combat integration through aggro and damage tracking

## Testing
Comprehensive test coverage in `*_test.go` files covering:
- Character creation and initialization
- Stat calculation accuracy
- Equipment stat aggregation
- Alignment system functionality
- Shop mechanics and restocking
- Kill/death tracking
- Cooldown management

This package serves as the foundation for all character-related functionality in GoMud, providing a rich and flexible character model that supports both player and NPC needs.