# Audio System Context

## Overview

The `internal/audio` package provides a simple audio configuration management system for the GoMud game engine. It handles loading and retrieving audio file configurations that can be used throughout the game for sound effects and music.

## Key Components

### Core Files
- **audio.go**: Main audio configuration management functionality

### Key Structures

#### AudioConfig
```go
type AudioConfig struct {
    FilePath string `yaml:"filepath,omitempty"`
    Volume   int    `yaml:"volume,omitempty"`
}
```
Represents an audio file configuration with file path and volume settings.

### Key Functions

#### Configuration Loading
- **LoadAudioConfig()**: Loads audio configurations from `_datafiles/audio.yaml` file
  - Clears existing configurations and reloads from file
  - Uses YAML unmarshaling to populate the audioLookup map
  - Logs loading time and count of loaded configurations
  - Panics on file read or YAML parsing errors

#### Audio Retrieval
- **GetFile(identifier string) AudioConfig**: Retrieves audio configuration by identifier
  - Returns AudioConfig if found, empty AudioConfig if not found
  - Thread-safe lookup from the audioLookup map

### Global State
- **audioLookup**: `map[string]AudioConfig` - In-memory cache of all loaded audio configurations

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing file path configurations
- `internal/mudlog`: For logging audio loading operations

### External Dependencies
- `gopkg.in/yaml.v2`: For YAML configuration parsing
- `github.com/pkg/errors`: For error wrapping
- Standard library: `os`, `time`

## Usage Patterns

### Loading Audio Configurations
```go
// Load all audio configurations from file
audio.LoadAudioConfig()
```

### Retrieving Audio Files
```go
// Get audio configuration for a specific sound
audioConfig := audio.GetFile("sword_clang")
if audioConfig.FilePath != "" {
    // Use the audio file
    playSound(audioConfig.FilePath, audioConfig.Volume)
}
```

## Configuration File Format

The system expects an `audio.yaml` file in the data files directory with the following structure:
```yaml
sound_identifier:
  filepath: "path/to/audio/file.wav"
  volume: 75
another_sound:
  filepath: "path/to/another/file.mp3"
  volume: 50
```

## Error Handling

- **File Loading Errors**: Panics if the audio.yaml file cannot be read or parsed
- **Missing Configurations**: Returns empty AudioConfig for non-existent identifiers
- **Thread Safety**: Uses map lookups which are safe for concurrent reads

## Performance Considerations

- **Memory Usage**: All audio configurations are loaded into memory at startup
- **Lookup Performance**: O(1) map lookups for audio configuration retrieval
- **Reload Strategy**: Complete replacement of configuration map on reload

## Integration Points

### Game Engine Integration
- Used by sound effect systems to get audio file paths and volume settings
- Integrated with MSP (MUD Sound Protocol) for client audio playback
- Referenced by scripting system for dynamic audio playback

### Configuration Management
- Follows standard GoMud configuration loading patterns
- Uses centralized file path configuration system
- Supports hot-reloading of audio configurations

## Future Considerations

- Could be extended to support audio format validation
- Potential for audio streaming or caching mechanisms
- Integration with client-side audio capabilities
- Support for audio playlists or sequences