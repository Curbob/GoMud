# Plugin System Context

## Overview

The `internal/plugins` package provides a comprehensive plugin architecture for the GoMud game engine. It enables extending the game with custom functionality through self-contained modules that can add commands, handle events, provide web interfaces, and integrate with all game systems without modifying the core codebase.

## Key Components

### Core Files
- **plugins.go**: Main plugin system and registry management
- **plugincallbacks.go**: Plugin callback system for event handling
- **pluginconfig.go**: Plugin-specific configuration management
- **pluginfiles.go**: Plugin file system integration and asset management
- **webconfig.go**: Web interface integration for plugins

### Key Structures

#### Plugin
```go
type Plugin struct {
    Name         string
    Version      string
    Description  string
    Dependencies []dependency
    FileSystem   fs.ReadFileFS
    Callbacks    PluginCallbacks
    Config       PluginConfig
    WebConfig    WebConfig
}
```
Main plugin structure containing all plugin metadata, functionality, and integration points.

#### PluginCallbacks
```go
type PluginCallbacks struct {
    LoadCallback     func() error
    SaveCallback     func() error
    CommandCallback  func(string) usercommands.UserCommand
    MobCommandCallback func(string) mobcommands.MobCommand
    IACCallback      func([]byte, connections.ConnectionId) bool
    NewConnectionCallback func(connections.ConnectionId)
    ScriptCallback   func(string) any
}
```
Callback functions for plugin integration with various game systems.

#### dependency
```go
type dependency struct {
    name    string
    version string
}
```
Plugin dependency specification for version management and compatibility.

### Global State
- **registry**: `pluginRegistry` - Global registry of all loaded plugins
- **registrationOpen**: `bool` - Controls whether new plugin registration is allowed
- **txtCleanRegex**: Regex for cleaning text input in plugin operations

## Core Functions

### Plugin Registration
- **Register(p *Plugin) error**: Registers new plugin with the system
  - Validates plugin structure and dependencies
  - Adds plugin to global registry
  - Integrates plugin callbacks with game systems
  - Handles plugin file system registration

### Plugin Discovery
- **GetAllPlugins() []*Plugin**: Returns all registered plugins
- **GetPlugin(name string) *Plugin**: Retrieves specific plugin by name
- **HasPlugin(name string) bool**: Checks if plugin is registered

### Command Integration
- **GetUserCommand(cmdName string) usercommands.UserCommand**: Retrieves user command from plugins
- **GetMobCommand(cmdName string) mobcommands.MobCommand**: Retrieves mob command from plugins
- **RegisterCommands()**: Registers all plugin commands with command systems

### File System Integration
- **ReadFile(name string) ([]byte, error)**: Reads files from plugin file systems
- **AllFileSubSystems(yield func(fs.ReadFileFS) bool)**: Iterates over all plugin file systems
- **WriteFile(name string, data []byte) error**: Writes files to plugin directories

## Plugin Features

### Modular Architecture
- **Self-Contained**: Plugins are complete, independent modules
- **Hot Loading**: Plugins can be loaded without server restart (where supported)
- **Dependency Management**: Automatic dependency resolution and validation
- **Version Control**: Plugin versioning and compatibility checking

### Command System Integration
- **User Commands**: Plugins can add new player commands
- **Mob Commands**: Plugins can add new NPC AI commands
- **Command Override**: Plugins can override existing commands
- **Dynamic Registration**: Commands registered automatically on plugin load

### Event System Integration
- **Lifecycle Events**: Load and save callbacks for plugin state management
- **Network Events**: Handle new connections and network protocols
- **IAC Processing**: Telnet protocol command handling
- **Custom Events**: Plugins can define and handle custom events

### File System Support
- **Embedded Assets**: Plugins can embed files using Go's embed.FS
- **Virtual File System**: Plugin files accessible through standard interfaces
- **Asset Serving**: Automatic serving of plugin assets through web interface
- **Data Overlays**: Plugin data files can override core game data

### Web Interface Integration
- **Custom Pages**: Plugins can add new web pages to admin interface
- **Navigation Integration**: Automatic menu integration for plugin pages
- **Template System**: Use game's template system for plugin web content
- **Asset Management**: Serve CSS, JavaScript, and images from plugins

## Dependencies

### Internal Dependencies
- `internal/configs`: For plugin configuration management
- `internal/mobcommands`: For mob command registration
- `internal/mudlog`: For plugin operation logging
- `internal/scripting`: For JavaScript integration
- `internal/usercommands`: For user command registration
- `internal/util`: For utility functions and file operations

### External Dependencies
- `gopkg.in/yaml.v2`: For plugin configuration parsing
- Standard library: `embed`, `fmt`, `io/fs`, `maps`, `net/http`, `os`, `path/filepath`, `reflect`, `regexp`, `strings`

## Usage Patterns

### Plugin Development
```go
// Define plugin structure
plugin := &Plugin{
    Name:        "MyPlugin",
    Version:     "1.0.0",
    Description: "Example plugin functionality",
    FileSystem:  myPluginFS, // embed.FS or other fs.ReadFileFS
    Callbacks: PluginCallbacks{
        LoadCallback: func() error {
            // Initialize plugin
            return nil
        },
        CommandCallback: func(cmdName string) usercommands.UserCommand {
            // Return custom commands
            return myCommands[cmdName]
        },
    },
}

// Register plugin
err := plugins.Register(plugin)
```

### Plugin File System
```go
//go:embed assets/*
var pluginAssets embed.FS

// Plugin can serve files from embedded file system
plugin := &Plugin{
    FileSystem: pluginAssets,
    // ... other configuration
}
```

### Web Integration
```go
// Add web pages to plugin
plugin.WebConfig = WebConfig{
    Pages: []WebPage{
        {
            Path:     "/admin/myplugin",
            Template: "myplugin/admin.html",
            Handler:  myAdminHandler,
        },
    },
    Navigation: []NavItem{
        {
            Text: "My Plugin",
            URL:  "/admin/myplugin",
        },
    },
}
```

## Integration Points

### Command Systems
- **User Commands**: Seamless integration with player command processing
- **Mob AI**: Integration with NPC command systems and AI behaviors
- **Command Discovery**: Automatic discovery and registration of plugin commands
- **Help Integration**: Plugin commands automatically included in help system

### Game Engine
- **Event Handling**: Plugin callbacks integrated with game event system
- **Data Access**: Plugins have access to game data through standard APIs
- **State Management**: Plugin state persistence through save/load callbacks
- **Resource Management**: Efficient resource sharing between plugins and core

### Web Interface
- **Admin Interface**: Plugin pages integrated into administrative web interface
- **Asset Serving**: Automatic serving of plugin web assets
- **Authentication**: Plugin pages inherit authentication and authorization
- **Template Integration**: Plugin templates use game's template system

### Scripting System
- **JavaScript Integration**: Plugin functions exposed to JavaScript runtime
- **Script Callbacks**: Plugins can provide functions for use in game scripts
- **Dynamic Functionality**: Runtime access to plugin functionality from scripts

## Performance Considerations

### Loading Efficiency
- **Lazy Loading**: Plugins loaded only when needed
- **Dependency Caching**: Efficient dependency resolution and caching
- **Resource Sharing**: Shared resources between plugins and core systems
- **Memory Management**: Efficient memory usage for plugin assets and data

### Runtime Performance
- **Command Lookup**: Optimized command lookup and execution
- **File System Access**: Efficient access to plugin file systems
- **Callback Performance**: Minimal overhead for plugin callback execution
- **Resource Pooling**: Pooled resources for frequently used plugin operations

### Scalability
- **Multiple Plugins**: Efficient handling of many simultaneous plugins
- **Concurrent Access**: Thread-safe plugin operations and resource access
- **Resource Limits**: Configurable limits to prevent resource exhaustion
- **Load Balancing**: Balanced resource usage across plugins

## Security Considerations

### Plugin Validation
- **Code Validation**: Validation of plugin code and functionality
- **Dependency Checking**: Verification of plugin dependencies and versions
- **Permission Management**: Plugin-specific permission and access control
- **Resource Limits**: Limits on plugin resource usage and capabilities

### Isolation and Safety
- **Sandboxing**: Plugin execution in controlled environments where possible
- **API Restrictions**: Limited API access based on plugin permissions
- **Data Protection**: Protection of sensitive game data from plugins
- **Error Isolation**: Plugin errors don't crash core game systems

### Access Control
- **Authentication**: Plugin access to authenticated functionality
- **Authorization**: Role-based access control for plugin operations
- **Audit Logging**: Comprehensive logging of plugin activities
- **Security Updates**: Mechanism for security updates and patches

## Future Enhancements

### Advanced Plugin Features
- **Hot Reloading**: Dynamic plugin reloading without server restart
- **Plugin Marketplace**: Centralized plugin distribution and management
- **Automatic Updates**: Automatic plugin updates and dependency management
- **Plugin Analytics**: Usage analytics and performance monitoring for plugins

### Enhanced Integration
- **Database Access**: Direct database access for plugins with proper permissions
- **Network Protocols**: Plugin-defined network protocols and handlers
- **Custom Events**: Advanced custom event system for inter-plugin communication
- **Resource Sharing**: Advanced resource sharing and communication between plugins

### Development Tools
- **Plugin SDK**: Comprehensive software development kit for plugin development
- **Testing Framework**: Testing tools and framework for plugin development
- **Documentation Tools**: Automatic documentation generation for plugins
- **Debug Support**: Advanced debugging tools for plugin development

### Management Features
- **Plugin Manager**: Web-based plugin management interface
- **Configuration UI**: Dynamic configuration interfaces for plugins
- **Monitoring Dashboard**: Real-time monitoring of plugin performance and health
- **Backup and Recovery**: Plugin-specific backup and recovery mechanisms

## Administrative Features

### Plugin Management
- **Installation**: Tools for installing and configuring plugins
- **Updates**: Plugin update management and version control
- **Dependencies**: Dependency management and conflict resolution
- **Removal**: Safe plugin removal and cleanup

### Monitoring and Analytics
- **Performance Monitoring**: Real-time monitoring of plugin performance
- **Usage Analytics**: Analysis of plugin usage patterns and effectiveness
- **Error Tracking**: Comprehensive error tracking and reporting
- **Resource Usage**: Monitoring of plugin resource consumption

### Configuration Management
- **Dynamic Configuration**: Runtime configuration changes for plugins
- **Configuration Validation**: Validation of plugin configurations
- **Backup and Restore**: Configuration backup and restoration
- **Template Configurations**: Template-based plugin configuration

## Development Guidelines

### Best Practices
- **Modular Design**: Design plugins as self-contained, modular components
- **Error Handling**: Comprehensive error handling and graceful degradation
- **Documentation**: Thorough documentation of plugin functionality and APIs
- **Testing**: Comprehensive testing of plugin functionality and integration

### Code Organization
- **Clear Structure**: Well-organized plugin code structure and architecture
- **API Design**: Clean, well-designed APIs for plugin functionality
- **Resource Management**: Efficient resource management and cleanup
- **Version Management**: Proper versioning and compatibility management