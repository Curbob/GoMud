# Conversations System Context

## Overview

The `internal/conversations` package provides a dynamic NPC conversation system for the GoMud game engine. It manages scripted dialogues between NPCs, supports conversation selection based on participant names, tracks conversation usage for variety, and provides turn-based conversation execution with automatic cleanup.

## Key Components

### Core Files
- **conversations.go**: Complete conversation management and execution system
- **conversation_datafile.go**: Data structure definitions for conversation files

### Key Structures

#### Conversation
```go
type Conversation struct {
    Id             int
    MobInstanceId1 int
    MobInstanceId2 int
    StartRound     uint64
    LastRound      uint64
    Position       int
    ActionList     [][]string
}
```
Runtime conversation instance containing:
- **Participant IDs**: Two mob instance IDs for conversation participants
- **Timing**: Start and last activity rounds for lifecycle management
- **Progress**: Current position in conversation action sequence
- **Actions**: List of command sequences to execute

#### ConversationData
```go
type ConversationData struct {
    Supported    map[string][]string `yaml:"Supported"`
    Conversation [][]string          `yaml:"Conversation"`
}
```
YAML-loadable conversation definition:
- **Supported**: Map of initiator names to allowed participant names
- **Conversation**: Sequence of command lists for conversation execution

### Global State
- **conversations**: `map[int]*Conversation` - Active conversation instances
- **conversationCounter**: `map[string]int` - Usage tracking for conversation variety
- **converseCheckCache**: `map[string]bool` - File existence cache for performance
- **conversationUniqueId**: Global ID counter for conversation instances

## Core Functions

### Conversation Initiation
- **AttemptConversation(initiatorMobId, initiatorInstanceId int, initiatorName string, participantInstanceId int, participantName string, zone string, forceIndex ...int) int**: Main conversation starter
  - Loads conversation definitions from zone-specific YAML files
  - Matches participants against supported name combinations
  - Selects least-used conversation for variety
  - Creates new conversation instance and returns unique ID
  - Returns 0 if no suitable conversation found

### Conversation Management
- **IsComplete(conversationId int) bool**: Checks if conversation is finished
  - Returns true if conversation doesn't exist or has reached end
  - Automatically destroys completed conversations
  - Used for conversation lifecycle management

- **Destroy(conversationId int)**: Manually destroys conversation instance
  - Removes conversation from active conversations map
  - Used for cleanup and forced conversation termination

### Action Execution
- **GetNextActions(convId int) (mob1, mob2 int, actions []string)**: Retrieves next conversation actions
  - Returns participant mob IDs and command list for current turn
  - Advances conversation position automatically
  - Prevents duplicate action execution within same round
  - Returns empty actions if conversation is complete

### File System Integration
- **HasConverseFile(mobId int, zone string) bool**: Checks for conversation file existence
  - Caches file existence checks for performance
  - Uses zone-specific file organization
  - Sanitizes zone names for consistent file paths

### Utility Functions
- **ZoneNameSanitize(zone string) string**: Normalizes zone names for file paths
  - Converts spaces to underscores
  - Converts to lowercase
  - Ensures consistent file naming

## Conversation Features

### Dynamic Selection
- **Name Matching**: Conversations selected based on participant names
- **Wildcard Support**: `*` wildcard for universal participant matching
- **Usage Balancing**: Least-used conversations selected for variety
- **Forced Selection**: Optional manual conversation index selection

### File Organization
- **Zone-Based**: Conversations organized by game zones
- **Mob-Specific**: Each mob can have its own conversation file
- **YAML Format**: Human-readable conversation definitions
- **Hierarchical Structure**: `conversations/zone/mobid.yaml` organization

### Execution Control
- **Turn-Based**: Conversations progress one step per game round
- **Duplicate Prevention**: Prevents multiple actions in same round
- **Automatic Cleanup**: Inactive conversations automatically removed
- **Position Tracking**: Maintains current conversation position

### Performance Optimization
- **File Caching**: Conversation file existence cached for performance
- **Lazy Loading**: Conversations loaded only when needed
- **Automatic Cleanup**: Periodic cleanup of stale conversations (2% chance per access)
- **Memory Management**: Efficient storage and cleanup of conversation data

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing configuration file paths
- `internal/mudlog`: For logging conversation errors and operations
- `internal/util`: For utility functions, file paths, and random number generation

### External Dependencies
- `gopkg.in/yaml.v2`: For YAML conversation file parsing
- Standard library: `fmt`, `os`, `strconv`, `strings`

## File Format

### Conversation YAML Structure
```yaml
# Example conversation file: conversations/frostfang/123.yaml
- Supported:
    "guard": ["player", "visitor"]
    "*": ["*"]  # Universal conversation
  Conversation:
    - ["#1", "say Hello there, traveler!"]
    - ["#2", "say Greetings, guard."]
    - ["#1", "say Welcome to our town."]
    - ["#2", "emote nods politely"]

- Supported:
    "guard": ["merchant"]
  Conversation:
    - ["#1", "say Any goods to declare?"]
    - ["#2", "say Just the usual wares."]
```

### Command Format
- **Participant Prefix**: `#1` (initiator) or `#2` (participant)
- **Command Structure**: `["#1", "command arguments"]`
- **Action Types**: Any valid mob command (say, emote, look, etc.)

## Usage Patterns

### Basic Conversation Initiation
```go
// Attempt to start conversation between NPCs
conversationId := conversations.AttemptConversation(
    guardMobId,           // Initiator mob ID
    guardInstanceId,      // Initiator instance ID
    "guard",              // Initiator name
    visitorInstanceId,    // Participant instance ID
    "visitor",            // Participant name
    "frostfang",          // Zone name
)

if conversationId > 0 {
    // Conversation successfully started
}
```

### Conversation Execution Loop
```go
// In game loop, process active conversations
for conversationId := range activeConversations {
    if conversations.IsComplete(conversationId) {
        continue // Conversation finished
    }
    
    mob1, mob2, actions := conversations.GetNextActions(conversationId)
    if len(actions) > 0 {
        // Execute actions for appropriate mob
        executeMobActions(mob1, mob2, actions)
    }
}
```

### File Existence Checking
```go
// Check if mob has conversation file before attempting
if conversations.HasConverseFile(mobId, zoneName) {
    // Mob has conversations available
    conversationId := conversations.AttemptConversation(...)
}
```

## Integration Points

### NPC AI System
- **Behavior Triggers**: Conversations triggered by NPC AI decisions
- **Social Interactions**: NPCs can initiate conversations with players or other NPCs
- **Contextual Responses**: Conversations based on game state and relationships
- **Dynamic Storytelling**: NPCs tell stories through scripted conversations

### Event System
- **Conversation Events**: Events triggered by conversation start/end
- **Action Integration**: Conversation actions integrate with mob command system
- **State Changes**: Conversations can trigger game state changes
- **Quest Integration**: Conversations can advance quest objectives

### Zone Management
- **Zone-Specific Content**: Conversations tailored to specific game areas
- **Cultural Context**: Zone-appropriate dialogue and interactions
- **Immersion**: Rich, contextual NPC interactions enhance world building
- **Scalable Content**: Easy addition of new conversations per zone

## Performance Considerations

### Caching Strategy
- **File Existence**: Cache file existence checks to reduce filesystem access
- **Conversation Loading**: Load conversations only when needed
- **Memory Cleanup**: Automatic cleanup of inactive conversations
- **Usage Tracking**: Efficient tracking of conversation usage patterns

### Memory Management
- **Lightweight Storage**: Minimal memory footprint per active conversation
- **Automatic Cleanup**: Periodic removal of stale conversations
- **Efficient Indexing**: Fast lookup of active conversations by ID
- **Resource Limits**: Implicit limits through automatic cleanup

### File System Optimization
- **Organized Structure**: Hierarchical file organization for efficient access
- **Lazy Loading**: Conversations loaded only when participants match
- **Error Handling**: Graceful handling of missing or malformed files
- **Path Sanitization**: Consistent file path generation

## Future Enhancements

### Advanced Features
- **Branching Conversations**: Multiple conversation paths based on conditions
- **Dynamic Content**: Conversations that change based on game state
- **Emotional States**: NPC mood affecting conversation selection
- **Relationship Tracking**: Conversation history affecting future interactions

### Enhanced Selection
- **Weighted Selection**: Probability-based conversation selection
- **Conditional Conversations**: Conversations requiring specific conditions
- **Time-Based Conversations**: Conversations available at certain times
- **Reputation-Based**: Conversations based on player reputation

### Content Management
- **Conversation Editor**: Visual tool for creating conversations
- **Validation Tools**: Tools for validating conversation syntax
- **Import/Export**: Tools for sharing conversations between servers
- **Version Control**: Track changes to conversation content

### Performance Improvements
- **Preloading**: Preload frequently used conversations
- **Compression**: Compressed storage for large conversation sets
- **Database Integration**: Optional database storage for conversations
- **Streaming**: Stream large conversations for memory efficiency

## Security and Validation

### Input Validation
- **Name Sanitization**: Ensure participant names are properly sanitized
- **Command Validation**: Validate conversation commands for safety
- **Path Security**: Prevent path traversal attacks through zone names
- **Resource Limits**: Prevent resource exhaustion through conversation abuse

### Content Safety
- **Command Filtering**: Filter dangerous or inappropriate commands
- **Content Validation**: Validate conversation content for appropriateness
- **Error Handling**: Safe handling of malformed conversation files
- **Access Control**: Ensure only authorized conversations are accessible

## Administrative Features

### Monitoring and Analytics
- **Usage Statistics**: Track conversation usage patterns
- **Performance Metrics**: Monitor conversation system performance
- **Error Tracking**: Log and track conversation-related errors
- **Content Analysis**: Analyze conversation effectiveness and popularity

### Content Management
- **Dynamic Loading**: Hot-reload conversations without server restart
- **Content Validation**: Validate conversation files before deployment
- **Backup and Recovery**: Backup and restore conversation content
- **Version Management**: Track conversation content versions