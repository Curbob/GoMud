# Mutators System Context

## Overview

The `internal/mutators` package provides a dynamic world modification system for the GoMud game engine. It manages temporary and permanent changes to game world elements such as room descriptions, item properties, and environmental conditions through configurable mutator specifications that can spawn, evolve, and decay over time.

## Key Components

### Core Files
- **mutators.go**: Complete mutator system implementation and management

### Key Structures

#### MutatorSpec
```go
type MutatorSpec struct {
    MutatorId     string
    Name          string
    Description   string
    Duration      string
    DecayRate     float64
    TextModifiers map[string]TextModifier
    ExitModifiers map[string]ExitModifier
    SpawnChance   float64
    Requirements  []string
}
```
Defines the specification and behavior of a mutator type, including its effects, duration, and spawning conditions.

#### Mutator
```go
type Mutator struct {
    MutatorId      string `yaml:"mutatorid,omitempty"`
    SpawnedRound   uint64 `yaml:"spawnedround,omitempty"`
    DespawnedRound uint64 `yaml:"despawnedround,omitempty"`
}
```
Runtime instance of a mutator applied to a specific game element, tracking its lifecycle and current state.

#### TextModifier
```go
type TextModifier struct {
    Behavior     TextBehavior `yaml:"behavior,omitempty"`
    Text         string       `yaml:"text,omitempty"`
    ColorPattern string       `yaml:"colorpattern,omitempty"`
}
```
Defines how text elements (descriptions, names) are modified by the mutator.

#### TextBehavior
```go
type TextBehavior string

const (
    TextPrepend TextBehavior = "prepend"
    TextAppend  TextBehavior = "append"
    TextReplace TextBehavior = "replace"
    TextDefault TextBehavior = TextReplace
)
```
Enumeration defining how text modifications are applied.

#### ExitModifier
```go
type ExitModifier struct {
    Blocked     bool
    NewExit     *exit.RoomExit
    Temporary   bool
    Description string
}
```
Defines modifications to room exits, including blocking passages or creating temporary connections.

### Global State
- **allMutators**: `map[string]*MutatorSpec` - Registry of all loaded mutator specifications

## Core Functions

### Mutator Management
- **LoadMutators()**: Loads mutator specifications from configuration files
  - Reads YAML files containing mutator definitions
  - Validates mutator specifications and requirements
  - Populates global mutator registry
  - Handles loading errors and malformed specifications

- **GetMutatorSpec(mutatorId string) *MutatorSpec**: Retrieves mutator specification
  - Returns mutator specification by ID
  - Used for spawning new mutator instances
  - Provides access to mutator properties and effects

### Mutator Application
- **ApplyMutator(target interface{}, mutatorId string) *Mutator**: Applies mutator to target
  - Creates new mutator instance
  - Applies specified modifications to target object
  - Records spawn time and initial state
  - Returns mutator instance for tracking

- **RemoveMutator(target interface{}, mutator *Mutator)**: Removes mutator from target
  - Reverses mutator effects on target object
  - Records despawn time for lifecycle tracking
  - Cleans up mutator state and resources

### Text Modification
- **ModifyText(original string, modifier TextModifier) string**: Applies text modifications
  - Supports prepend, append, and replace behaviors
  - Applies color patterns to modified text
  - Handles complex text transformation rules
  - Maintains text formatting and structure

### Lifecycle Management
- **UpdateMutators(targetList []interface{})**: Updates all active mutators
  - Processes decay and evolution of active mutators
  - Handles mutator expiration and removal
  - Applies time-based changes to mutator effects
  - Manages mutator lifecycle transitions

## Mutator Features

### Dynamic World Changes
- **Environmental Effects**: Weather, seasonal changes, and atmospheric conditions
- **Temporary Modifications**: Time-limited changes to world elements
- **Progressive Changes**: Gradual evolution of world state over time
- **Conditional Effects**: Changes triggered by specific conditions or events

### Text System Integration
- **Description Modification**: Dynamic changes to room and object descriptions
- **Name Changes**: Temporary or permanent name modifications
- **Color Application**: Dynamic color pattern application to text
- **Formatting Control**: Advanced text formatting and presentation control

### Exit System Integration
- **Passage Blocking**: Temporary or permanent blocking of exits
- **New Connections**: Creation of temporary passages and shortcuts
- **Exit Descriptions**: Dynamic modification of exit descriptions
- **Access Control**: Conditional access to passages based on mutator state

### Time-Based Evolution
- **Decay Systems**: Gradual weakening or strengthening of mutator effects
- **Lifecycle Stages**: Multi-stage mutator evolution over time
- **Duration Control**: Configurable duration for mutator effects
- **Automatic Cleanup**: Automatic removal of expired mutators

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing configuration file paths
- `internal/exit`: For exit modification and management
- `internal/fileloader`: For loading mutator specification files
- `internal/gametime`: For time-based mutator calculations
- `internal/mudlog`: For logging mutator operations
- `internal/util`: For utility functions and file operations

### External Dependencies
- `gopkg.in/yaml.v2`: For YAML mutator specification parsing
- Standard library: `fmt`, `os`, `strings`, `time`

## Usage Patterns

### Mutator Specification
```yaml
# Example mutator specification file
dusty:
  name: "Dusty Condition"
  description: "Adds dust and age to room descriptions"
  duration: "2 hours"
  decay_rate: 0.1
  text_modifiers:
    room_description:
      behavior: "prepend"
      text: "A thick layer of dust covers everything here. "
      color_pattern: "dusty"
  spawn_chance: 0.05
  requirements: ["abandoned", "indoor"]
```

### Runtime Application
```go
// Apply mutator to room
mutator := mutators.ApplyMutator(room, "dusty")
if mutator != nil {
    // Mutator successfully applied
    room.AddMutator(mutator)
}

// Update all mutators in area
mutators.UpdateMutators(roomList)

// Remove specific mutator
mutators.RemoveMutator(room, mutator)
```

### Text Modification
```go
// Modify text with mutator
modifier := TextModifier{
    Behavior:     TextPrepend,
    Text:         "Frost covers ",
    ColorPattern: "ice",
}

modifiedText := mutators.ModifyText(originalText, modifier)
```

## Integration Points

### Room System
- **Description Changes**: Dynamic modification of room descriptions
- **Environmental Effects**: Weather and atmospheric condition simulation
- **Temporary Features**: Addition of temporary room features and elements
- **Exit Modifications**: Dynamic changes to room connections and passages

### Item System
- **Item Descriptions**: Dynamic modification of item descriptions and properties
- **Condition Effects**: Simulation of item wear, damage, and aging
- **Temporary Enhancements**: Time-limited item modifications and effects
- **Visual Changes**: Dynamic changes to item appearance and presentation

### Event System
- **Trigger Events**: Mutators can be triggered by game events
- **State Changes**: Mutator changes can trigger additional game events
- **Condition Monitoring**: Integration with condition-based event systems
- **Automatic Responses**: Automated responses to mutator lifecycle events

### Time System
- **Duration Tracking**: Integration with game time for mutator duration
- **Scheduled Changes**: Time-based mutator spawning and evolution
- **Seasonal Effects**: Seasonal mutator applications and changes
- **Decay Calculations**: Time-based decay and evolution calculations

## Performance Considerations

### Efficient Processing
- **Batch Updates**: Efficient batch processing of mutator updates
- **Selective Application**: Apply mutators only where needed
- **Caching**: Cache frequently used mutator specifications
- **Lazy Evaluation**: Evaluate mutator effects only when accessed

### Memory Management
- **Lightweight Instances**: Minimal memory footprint for mutator instances
- **Automatic Cleanup**: Automatic cleanup of expired mutators
- **Resource Pooling**: Pooled resources for frequently used mutators
- **Garbage Collection**: Efficient garbage collection of mutator data

### Scalability
- **Large Worlds**: Efficient handling of mutators across large game worlds
- **Concurrent Processing**: Thread-safe mutator operations
- **Distributed Updates**: Support for distributed mutator processing
- **Load Balancing**: Balanced processing of mutator updates

## Future Enhancements

### Advanced Mutator Types
- **Interactive Mutators**: Mutators that respond to player actions
- **Conditional Mutators**: Complex condition-based mutator behavior
- **Chained Mutators**: Mutators that trigger other mutators
- **Probabilistic Effects**: Mutators with probability-based effects

### Enhanced Integration
- **Quest Integration**: Mutators that interact with quest systems
- **Combat Effects**: Mutators that affect combat and gameplay
- **Economic Impact**: Mutators that influence game economy
- **Social Effects**: Mutators that affect player interactions

### Visual and Audio
- **Visual Effects**: Integration with visual effect systems
- **Audio Changes**: Dynamic audio modifications through mutators
- **Animation Support**: Animated mutator effects and transitions
- **Particle Systems**: Integration with particle effect systems

### Administrative Tools
- **Mutator Editor**: Visual tools for creating and editing mutators
- **Testing Tools**: Tools for testing mutator behavior and effects
- **Analytics**: Analysis of mutator usage and effectiveness
- **Debugging**: Advanced debugging tools for mutator development

## Security and Validation

### Input Validation
- **Specification Validation**: Comprehensive validation of mutator specifications
- **Parameter Checking**: Validation of mutator parameters and settings
- **Safety Limits**: Limits on mutator effects to prevent abuse
- **Error Handling**: Robust error handling for malformed specifications

### System Protection
- **Resource Limits**: Limits on mutator resource usage
- **Performance Protection**: Protection against performance-impacting mutators
- **State Integrity**: Maintenance of game state integrity during mutations
- **Rollback Capability**: Ability to rollback problematic mutator changes

## Administrative Features

### Mutator Management
- **Dynamic Loading**: Hot-loading of new mutator specifications
- **Configuration Updates**: Runtime updates to mutator configurations
- **Batch Operations**: Batch application and removal of mutators
- **State Monitoring**: Real-time monitoring of mutator states

### Analytics and Reporting
- **Usage Statistics**: Analysis of mutator usage patterns
- **Performance Metrics**: Monitoring of mutator performance impact
- **Effect Analysis**: Analysis of mutator effects on gameplay
- **Player Feedback**: Integration with player feedback systems

### Development Support
- **Testing Framework**: Framework for testing mutator behavior
- **Simulation Tools**: Tools for simulating mutator effects
- **Documentation**: Automatic documentation generation for mutators
- **Version Control**: Version management for mutator specifications

## Error Handling and Recovery

### Robust Operation
- **Graceful Degradation**: Graceful handling of mutator failures
- **Error Recovery**: Automatic recovery from mutator errors
- **State Consistency**: Maintenance of consistent game state
- **Rollback Mechanisms**: Rollback capabilities for failed mutations

### Debugging and Monitoring
- **Comprehensive Logging**: Detailed logging of mutator operations
- **Error Tracking**: Tracking and analysis of mutator errors
- **Performance Monitoring**: Monitoring of mutator performance impact
- **State Validation**: Validation of mutator state consistency