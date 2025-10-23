# Modules System Context

## Overview
The GoMud modules system provides a powerful plugin architecture that allows for extending the game engine with custom functionality without modifying the core codebase. Modules are self-contained packages that can add commands, handle events, provide web interfaces, and integrate deeply with all game systems through a comprehensive plugin API.

## Architecture Components

### Plugin Infrastructure (`internal/plugins`)

#### **Core Plugin System** (`plugins.go`)
- **Plugin struct**: Central plugin definition with callbacks, configuration, and file system integration
- **Plugin registry**: Global registry managing all loaded plugins with automatic discovery
- **Function export system**: Allows plugins to expose functions to other systems and JavaScript scripting
- **Embedded file system**: Each plugin can embed its own files (templates, web assets, data files)
- **Dependency management**: Plugin dependency resolution and version tracking

#### **Plugin Callbacks** (`plugincallbacks.go`)
- **Command registration**: Add new user commands and mob AI commands
- **Event handling**: IAC (telnet protocol) command processing
- **Network callbacks**: Handle new connection events
- **Lifecycle hooks**: Load and save callbacks for plugin state management
- **Script integration**: Expose plugin functions to JavaScript runtime

#### **Configuration System** (`pluginconfig.go`)
- **Plugin-specific config**: Each plugin gets its own configuration namespace
- **Dynamic configuration**: Runtime configuration changes through the main config system
- **Persistent settings**: Plugin configurations are saved with the main game configuration

#### **Web Integration** (`webconfig.go`)
- **Custom web pages**: Plugins can add new web pages to the admin interface
- **Navigation integration**: Add menu items to the web interface
- **Template system**: Use custom HTML templates with dynamic data
- **Asset serving**: Serve static files (CSS, JS, images) from plugin file systems

#### **File System** (`pluginfiles.go`)
- **Embedded files**: Go embed.FS integration for packaging plugin assets
- **Virtual file system**: Plugins provide files through fs.ReadFileFS interface
- **File overlay system**: Plugin files can override core game files
- **Data file integration**: Plugin data files are merged with core data files

### Module Implementations (`modules/`)

#### **GMCP Module** (`modules/gmcp/`)
**Generic MUD Communication Protocol implementation**
- **Multi-component architecture**: Separate components for Char, Comm, Game, Mudlet, Party, Room
- **Real-time data sync**: Character stats, room information, party status to MUD clients
- **Client integration**: Mudlet-specific features including mapping and UI components
- **Telnet protocol handling**: IAC command processing for GMCP negotiation
- **Discord integration**: Status updates and message bridging
- **Event-driven updates**: Responds to character changes, room changes, party updates

#### **Auctions Module** (`modules/auctions/`)
**Player auction system**
- **Auction management**: Start, bid, and end auction functionality
- **Event-driven processing**: Uses NewRound events for auction timing
- **Template system**: Custom templates for auction notifications
- **State persistence**: Saves auction history and current auction state
- **User command integration**: Adds `auction` command for player interaction
- **Broadcast integration**: Auction updates sent to all players

#### **Follow Module** (`modules/follow/`)
**Player and mob following mechanics**
- **Follow relationships**: Track who follows whom with limits and restrictions
- **Event integration**: Responds to room changes, player/mob deaths, party updates
- **AI integration**: Idle mob handlers for following behavior
- **Scripting functions**: Exposes follow functions to JavaScript runtime
- **State management**: Persistent follow relationships across server restarts
- **Party integration**: Following behavior integrates with party system

#### **Leaderboards Module** (`modules/leaderboards/`)
**Player statistics and rankings**
- **Web interface**: Custom web page showing player rankings
- **Data collection**: Tracks various player statistics over time
- **Periodic updates**: Uses NewRound events to update statistics
- **Persistent storage**: Saves leaderboard data between server restarts
- **Navigation integration**: Adds leaderboards link to web interface

#### **Time Module** (`modules/time/`)
**Game time display functionality**
- **Simple command**: Adds `time` command to display current game time
- **Template integration**: Uses help system for command documentation
- **Minimal example**: Demonstrates basic plugin functionality

#### **Cleanup Module** (`modules/cleanup/`)
**World cleanup and maintenance**
- **Cleanup commands**: `bury` and `trash` commands for removing items/corpses
- **World maintenance**: Automated cleanup of abandoned items and corpses
- **Configuration**: Configurable cleanup intervals and rules

#### **Web Help Module** (`modules/webhelp/`)
**Web-based help system**
- **Help browser**: Web interface for browsing game help files
- **Search functionality**: Search through help topics via web interface
- **Template system**: Custom HTML templates for help display

## Event System Integration

### **Event-Driven Architecture**
Modules extensively use the event system (`internal/events`) to integrate with game mechanics:

#### **Core Event Types**
- **NewRound**: Periodic processing (auctions, leaderboards, follow mechanics)
- **NewTurn**: Turn-based updates and maintenance
- **PlayerSpawn/PlayerDespawn**: Player login/logout handling
- **RoomChange**: Movement tracking and following behavior
- **PartyUpdated**: Party system integration
- **Communication**: Chat and communication system integration
- **CharacterVitalsChanged**: Character stat updates for GMCP
- **EquipmentChange**: Equipment updates for client sync
- **ItemOwnership**: Item tracking and quest integration

#### **Event Registration**
```go
events.RegisterListener(events.NewRound{}, module.handleNewRound)
events.RegisterListener(events.RoomChange{}, module.handleRoomChange)
```

#### **Event Priority System**
- **events.First**: High priority event handling
- **events.Last**: Final event processing
- **Default priority**: Standard event processing order

### **Custom Event Types**
Modules can define and emit custom events:
```go
type GMCPOut struct {
    ConnectionId uint64
    Data         []byte
}
```

## Plugin Capabilities

### **Command System Integration**
```go
// Add user commands
plugin.AddUserCommand("auction", auctionCommand, allowWhenDowned, adminOnly)

// Add mob AI commands  
plugin.AddMobCommand("customai", aiCommand, allowWhenDowned)
```

### **Scripting Integration**
```go
// Expose functions to JavaScript
plugin.AddScriptingFunction("GetFollowers", getFollowersFunc)

// Available in scripts as:
// modules.follow.GetFollowers()
```

### **Web System Integration**
```go
// Add web pages
plugin.Web.WebPage("Leaderboards", "/leaderboards", "leaderboards.html", true, dataFunc)

// Add navigation links
plugin.Web.NavLink("Auctions", "/auctions")
```

### **Configuration Management**
```go
// Plugin-specific configuration
plugin.Config.Set("maxAuctions", 10)
value := plugin.Config.Get("maxAuctions")
```

### **File System Integration**
```go
//go:embed files/*
var files embed.FS

// Attach to plugin
plugin.AttachFileSystem(files)

// Files available as overlays to core system
```

## Data File Integration

### **File Overlay System**
Modules can provide files that override or extend core game data:

#### **Data Overlays** (`files/data-overlays/`)
- **config.yaml**: Module-specific configuration additions
- **keywords.yaml**: Help system keyword additions
- **ansi-aliases.yaml**: Color scheme additions

#### **Data Files** (`files/datafiles/`)
- **templates/**: Custom message templates
- **html/**: Web interface files
- **help/**: Help system documentation

### **Template System Integration**
Modules can provide custom templates for:
- **Auction notifications**: Bid updates, auction start/end messages
- **Help documentation**: Command help and feature documentation  
- **Web interfaces**: Custom HTML pages with dynamic data

## Module Development Patterns

### **Basic Module Structure**
```go
package mymodule

import (
    "embed"
    "github.com/GoMudEngine/GoMud/internal/plugins"
    "github.com/GoMudEngine/GoMud/internal/events"
)

//go:embed files/*
var files embed.FS

func init() {
    plugin := plugins.New("mymodule", "1.0")
    
    // Attach file system
    plugin.AttachFileSystem(files)
    
    // Register commands
    plugin.AddUserCommand("mycommand", myCommand, false, false)
    
    // Register event listeners
    events.RegisterListener(events.NewRound{}, handleNewRound)
    
    // Set up callbacks
    plugin.Callbacks.SetOnLoad(onLoad)
    plugin.Callbacks.SetOnSave(onSave)
}
```

### **State Management**
```go
type ModuleData struct {
    SomeState map[string]interface{} `json:"somestate"`
}

func (m *MyModule) save() {
    data := ModuleData{SomeState: m.state}
    m.plugin.WriteBytes("state.json", data)
}

func (m *MyModule) load() {
    var data ModuleData
    m.plugin.ReadBytes("state.json", &data)
    m.state = data.SomeState
}
```

### **Event Handling**
```go
func (m *MyModule) handleNewRound(e events.Event) events.ListenerReturn {
    if evt, ok := e.(events.NewRound); ok {
        // Process round-based logic
        m.processRoundLogic(evt.RoundNumber)
    }
    return events.Continue
}
```

## Integration Points

### **Core System Hooks**
The engine provides numerous hooks for module integration:

#### **Auto-Save Integration**
- Modules are automatically saved during system save operations
- `plugins.Save()` called from `internal/hooks/NewTurn_AutoSave.go`

#### **Command System**
- User commands integrated into `internal/usercommands` system
- Mob commands integrated into `internal/mobcommands` system
- Full access to command infrastructure and permissions

#### **Event System**
- Full access to all game events
- Ability to register listeners with priority control
- Custom event types supported

#### **Web System**
- Integration with admin web interface
- Custom pages and navigation
- Template system access

### **JavaScript Runtime Integration**
- Module functions exposed to scripting system
- Available in scripts as `modules.modulename.functionname()`
- Full integration with game scripting capabilities

## Performance Considerations

### **Event Processing**
- Event listeners are called synchronously
- Heavy processing should be deferred or optimized
- Event priority system allows control over execution order

### **File System**
- Embedded files are loaded at compile time
- File overlay system provides efficient file serving
- Plugin files cached in memory for performance

### **State Management**
- Plugin state saved/loaded with main game state
- JSON serialization for complex data structures
- Efficient binary serialization available

## Security and Isolation

### **Sandboxed Execution**
- Plugins run in the same process but with controlled access
- No direct file system access outside of embedded files
- Configuration changes go through main config system

### **API Boundaries**
- Well-defined interfaces for all plugin interactions
- Type-safe event system
- Controlled access to core game systems

## Module Ecosystem

### **Official Modules**
- **GMCP**: Essential for modern MUD clients
- **Auctions**: Player economy features
- **Follow**: Social and AI mechanics
- **Leaderboards**: Player engagement and competition
- **Time**: Basic utility functionality
- **Cleanup**: World maintenance
- **Web Help**: Enhanced help system

### **Module Discovery**
- Automatic module loading via `modules/all-modules.go`
- Code generation for module imports
- Compile-time module inclusion

This comprehensive plugin architecture allows GoMud to be extended with sophisticated functionality while maintaining clean separation between core engine and optional features. The event-driven design ensures modules can integrate deeply with game mechanics while remaining modular and maintainable.