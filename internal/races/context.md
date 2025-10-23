# Race System Context

## Overview

The `internal/races` package provides a comprehensive character race system for the GoMud game engine. It manages racial characteristics, abilities, stat bonuses, size classifications, and behavioral traits that define the fundamental nature of player characters and NPCs.

## Key Components

### Core Files
- **races.go**: Complete race management and definition system

### Key Structures

#### Race
```go
type Race struct {
    RaceId           int
    Name             string
    Description      string
    DefaultAlignment int8
    BuffIds          []int
    Size             Size
    TNLScale         float32
    UnarmedName      string
    Tameable         bool
    Damage           items.Damage
    Selectable       bool
    AngryCommands    []string
    KnowsFirstAid    bool
    Stats            stats.Statistics
    DisabledSlots    []string `yaml:"disabledslots,omitempty"`
}
```
Comprehensive race definition including:
- **Identity**: ID, name, and description for race identification
- **Alignment**: Default moral/ethical alignment for the race
- **Abilities**: Permanent buffs and special capabilities
- **Physical**: Size classification and combat characteristics
- **Progression**: TNL (To Next Level) scaling for experience requirements
- **Behavior**: AI commands and behavioral patterns
- **Equipment**: Disabled equipment slots for anatomical restrictions

#### Size Enumeration
```go
type Size string

const (
    Small  Size = "small"  // Mouse, dog-sized creatures
    Medium Size = "medium" // Human-sized creatures
    Large  Size = "large"  // Troll, ogre, dragon-sized creatures
)
```
Physical size classification affecting combat, equipment, and interactions.

### Global State
- **races**: `map[int]*Race` - Registry of all available races indexed by race ID

## Core Functions

### Race Management
- **GetRaces() []Race**: Returns slice of all available races
  - Provides complete race list for character creation
  - Used for administrative and display purposes
  - Returns copies to prevent modification of originals

- **GetRace(raceId int) *Race**: Retrieves specific race by ID
  - Returns race pointer for direct access
  - Used for character creation and validation
  - Returns nil if race doesn't exist

- **LoadRaces()**: Loads race definitions from configuration files
  - Reads race YAML files from data directory
  - Validates race data and populates race registry
  - Logs loading progress and statistics
  - Handles loading errors and validation failures

### Race Properties
- **GetSize() Size**: Returns race size classification
- **GetStats() stats.Statistics**: Returns base racial statistics
- **GetBuffs() []int**: Returns permanent racial buff IDs
- **IsSelectable() bool**: Determines if race is available for player selection
- **IsTameable() bool**: Determines if race can be tamed as pet/mount

## Race Features

### Physical Characteristics
- **Size Classification**: Small, medium, or large size categories
- **Equipment Restrictions**: Disabled equipment slots based on anatomy
- **Combat Capabilities**: Unarmed combat names and damage values
- **Physical Limitations**: Size-based restrictions and capabilities

### Stat System Integration
- **Base Statistics**: Racial starting stats for all attributes
- **Stat Modifiers**: Racial bonuses and penalties to specific stats
- **Progression Scaling**: TNL scale affects experience requirements
- **Balanced Design**: Races balanced through stat distribution

### Buff and Ability System
- **Permanent Buffs**: Always-active racial abilities
- **Special Abilities**: Unique racial capabilities and traits
- **Passive Effects**: Ongoing racial bonuses and modifications
- **Active Powers**: Triggered racial abilities (implementation dependent)

### Behavioral System
- **AI Commands**: Predefined commands for NPC behavior when angry
- **Personality Traits**: Racial behavioral characteristics
- **Social Interactions**: Race-specific interaction patterns
- **Combat Behavior**: Racial combat preferences and tactics

### Character Creation
- **Selectable Races**: Races available for player character creation
- **NPC Races**: Races restricted to NPC use
- **Balanced Options**: Multiple viable character creation choices
- **Unique Gameplay**: Each race offers distinct gameplay experience

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing configuration file paths
- `internal/fileloader`: For loading race definition files
- `internal/items`: For damage system integration
- `internal/mudlog`: For logging race loading operations
- `internal/stats`: For racial statistics system
- `internal/util`: For utility functions and file operations

### External Dependencies
- `gopkg.in/yaml.v2`: For YAML race definition parsing
- Standard library: `errors`, `fmt`, `os`, `strings`, `time`

## Data Persistence

### YAML Configuration
- **Race Definitions**: Races defined in YAML configuration files
- **Modular Loading**: Each race can be defined in separate files
- **Validation**: Comprehensive validation during loading process
- **Error Handling**: Graceful handling of malformed race data

### File Operations
- **Batch Loading**: Efficient loading of all race definitions
- **Incremental Updates**: Support for adding new races without restart
- **Validation**: Data integrity checking during load operations
- **Error Reporting**: Detailed error reporting for configuration issues

## Usage Patterns

### Character Creation
```go
// Get available races for character creation
availableRaces := races.GetRaces()
selectableRaces := []Race{}
for _, race := range availableRaces {
    if race.Selectable {
        selectableRaces = append(selectableRaces, race)
    }
}
```

### Race Information Retrieval
```go
// Get specific race information
race := races.GetRace(raceId)
if race != nil {
    baseStats := race.GetStats()
    racialBuffs := race.GetBuffs()
    size := race.GetSize()
}
```

### Gameplay Integration
```go
// Apply racial characteristics
character.ApplyRacialStats(race.Stats)
character.ApplyRacialBuffs(race.BuffIds)
character.SetSize(race.Size)
character.SetTNLScale(race.TNLScale)
```

## Integration Points

### Character System
- **Stat Foundation**: Provides base statistics for character creation
- **Buff Application**: Applies permanent racial buffs to characters
- **Size Effects**: Size affects combat, equipment, and movement
- **Progression**: TNL scaling affects character advancement

### Combat System
- **Unarmed Combat**: Racial unarmed attack names and damage
- **Size Modifiers**: Size affects combat calculations and targeting
- **Racial Abilities**: Combat-related racial buffs and abilities
- **Behavioral AI**: Angry commands for NPC combat behavior

### Equipment System
- **Slot Restrictions**: Disabled slots prevent certain equipment usage
- **Size Compatibility**: Equipment compatibility based on race size
- **Anatomical Limits**: Equipment restrictions based on race anatomy
- **Balance Considerations**: Equipment restrictions balance racial advantages

### Experience System
- **TNL Scaling**: Racial experience multipliers for progression balance
- **Advancement Rate**: Different races advance at different rates
- **Balance Mechanism**: Prevents overpowered race combinations
- **Progression Variety**: Creates different character development paths

## Performance Considerations

### Memory Efficiency
- **Static Data**: Race definitions loaded once at startup
- **Efficient Storage**: Compact race data structures
- **Reference Sharing**: Shared race references across characters
- **Minimal Allocation**: Race access without additional allocations

### Loading Optimization
- **Batch Loading**: All races loaded efficiently at startup
- **Validation Caching**: Validation results cached for performance
- **Error Aggregation**: Efficient error collection and reporting
- **Parallel Processing**: Concurrent race loading where applicable

## Balance and Design

### Racial Balance
- **Stat Distribution**: Balanced stat allocation across races
- **Advantage/Disadvantage**: Each race has strengths and weaknesses
- **Niche Specialization**: Races excel in different areas
- **Viable Options**: All selectable races provide viable gameplay

### Progression Balance
- **TNL Scaling**: Experience requirements balance racial advantages
- **Long-term Balance**: Racial differences remain relevant at all levels
- **Power Curves**: Racial power progression carefully tuned
- **Endgame Viability**: All races remain competitive at high levels

## Future Enhancements

### Advanced Racial Features
- **Racial Evolution**: Races that change over time or through actions
- **Hybrid Races**: Mixed-race characters with combined traits
- **Cultural Variants**: Sub-races with cultural differences
- **Racial Prestige**: Advanced racial abilities unlocked through play

### Enhanced Customization
- **Racial Talents**: Selectable racial abilities and specializations
- **Cultural Background**: Additional customization beyond base race
- **Appearance Options**: Visual customization within racial bounds
- **Personality Traits**: Behavioral customization options

### Social Systems
- **Racial Relations**: Inter-racial relationships and conflicts
- **Racial Territories**: Race-specific areas and settlements
- **Cultural Events**: Race-specific holidays and celebrations
- **Racial Politics**: Political systems and governance structures

### Gameplay Integration
- **Racial Quests**: Race-specific quest lines and content
- **Environmental Adaptation**: Racial bonuses in specific environments
- **Social Interactions**: Race affects NPC interactions and dialogue
- **Economic Systems**: Racial preferences in trade and commerce

## Security and Validation

### Data Integrity
- **Race Validation**: Ensures race data consistency and completeness
- **Stat Validation**: Validates racial stat distributions and limits
- **Buff Validation**: Ensures racial buffs exist and are valid
- **Configuration Validation**: Comprehensive validation of race definitions

### Balance Protection
- **Stat Limits**: Prevents overpowered racial stat combinations
- **Progression Limits**: Ensures balanced experience progression
- **Ability Limits**: Prevents exploitation of racial abilities
- **Equipment Balance**: Equipment restrictions maintain game balance

## Administrative Features

### Race Management
- **Race Statistics**: Analytics on race selection and performance
- **Balance Monitoring**: Tracking racial balance and player preferences
- **Performance Analysis**: Race effectiveness in different game areas
- **Population Tracking**: Distribution of races across player base

### Development Tools
- **Race Editor**: Tools for creating and modifying race definitions
- **Balance Testing**: Automated testing of racial balance
- **Validation Tools**: Tools for validating race configurations
- **Import/Export**: Tools for sharing race definitions between servers