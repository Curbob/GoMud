# GoMud Util System Context

## Overview

The GoMud util system provides essential utility functions and infrastructure services including game timing management, string processing, file operations, cryptographic functions, memory monitoring, performance tracking, and cross-platform compatibility utilities. It serves as the foundational layer supporting all other game systems with thread-safe operations and comprehensive debugging tools.

## Architecture

The util system is built around several key categories:

### Core Components

**Game Timing System:**
- Turn and round counting with persistent state
- High-level mutex synchronization for game data
- Performance tracking and accumulator system
- Time-based event coordination

**String Processing:**
- Text matching and fuzzy search algorithms
- Color code processing and ANSI conversion
- Filename sanitization and path utilities
- Multi-language text processing (CJK support)

**Cryptographic Services:**
- Password hashing with SHA-256 and MD5 support
- Secure random number generation
- Base64 encoding/decoding utilities
- Hash-based data integrity

**Memory Management:**
- Memory usage tracking and reporting
- Performance monitoring and statistics
- Resource cleanup and optimization
- System health monitoring

## Key Features

### 1. **Game Timing and Synchronization**
- **Turn/Round Counting**: Persistent game time tracking
- **Thread Safety**: High-level mutex for game data synchronization
- **Performance Tracking**: Accumulator system for timing analysis
- **State Persistence**: Round count persistence across server restarts

### 2. **Advanced String Processing**
- **Fuzzy Matching**: Sophisticated text matching algorithms
- **Color Processing**: ANSI color code handling and conversion
- **Filename Sanitization**: Cross-platform safe filename generation
- **Multi-language Support**: Unicode and CJK character handling

### 3. **Comprehensive Utilities**
- **File Operations**: Path utilities and file manipulation
- **Random Generation**: Seeded random numbers and dice rolling
- **Data Compression**: Gzip compression for data storage
- **Network Utilities**: HTTP helpers and address management

### 4. **System Monitoring**
- **Memory Tracking**: Detailed memory usage reporting
- **Performance Metrics**: Execution time tracking and analysis
- **Resource Monitoring**: System resource usage statistics
- **Debug Support**: Comprehensive logging and debugging utilities

## Game Timing System

### Turn and Round Management
```go
var (
    turnCount  uint64 = 0
    roundCount uint64 = RoundCountMinimum // Start at 1314000 for stability
)

const (
    RoundCountMinimum  = 1314000        // ~4 years offset for delta safety
    RoundCountFilename = ".roundcount"  // Persistence file
)

// Thread-safe turn counting
func IncrementTurnCount() uint64 {
    turnCount++
    return turnCount
}

func GetTurnCount() uint64 {
    return turnCount
}

// Thread-safe round counting with persistence
func IncrementRoundCount() uint64 {
    roundCount++
    return roundCount
}

func GetRoundCount() uint64 {
    return roundCount
}

func SetRoundCount(newRoundCount uint64) {
    roundCount = newRoundCount
}
```

### High-Level Synchronization
```go
var mudLock = sync.RWMutex{}

// Exclusive lock for game data modifications
func LockMud() {
    mudLock.Lock()
}

func UnlockMud() {
    mudLock.Unlock()
}

// Shared lock for game data reading
func RLockMud() {
    mudLock.RLock()
}

func RUnlockMud() {
    mudLock.RUnlock()
}
```

### Performance Tracking
```go
type Accumulator struct {
    Name    string    // Tracker name
    Total   float64   // Total time accumulated
    Lowest  float64   // Fastest execution time
    Highest float64   // Slowest execution time
    Count   float64   // Number of samples
    Average float64   // Calculated average
    Start   time.Time // Tracker creation time
}

var timeTrackers = map[string]*Accumulator{}

// Track execution time for performance analysis
func TrackTime(name string, timePassed float64) {
    if _, ok := timeTrackers[name]; !ok {
        timeTrackers[name] = &Accumulator{
            Name:  name,
            Start: time.Now(),
        }
    }
    timeTrackers[name].Record(timePassed)
}

// Get all performance tracking data
func GetTimeTrackers() []Accumulator {
    result := []Accumulator{}
    for _, t := range timeTrackers {
        result = append(result, *t)
    }
    return result
}
```

## String Processing System

### Text Matching and Search
```go
// Sophisticated fuzzy matching algorithm
func FindMatchIn(searchFor string, searchIn ...string) (closeMatch string, exactMatch string) {
    searchFor = strings.ToLower(strings.TrimSpace(searchFor))
    
    if searchFor == "" {
        return "", ""
    }
    
    var bestPartialMatch string
    var bestPartialScore int
    
    for _, candidate := range searchIn {
        candidateLower := strings.ToLower(candidate)
        
        // Exact match takes priority
        if candidateLower == searchFor {
            return candidate, candidate
        }
        
        // Prefix matching
        if strings.HasPrefix(candidateLower, searchFor) {
            if bestPartialMatch == "" || len(candidate) < len(bestPartialMatch) {
                bestPartialMatch = candidate
            }
        }
        
        // Substring matching with scoring
        if strings.Contains(candidateLower, searchFor) {
            score := calculateMatchScore(searchFor, candidateLower)
            if score > bestPartialScore {
                bestPartialScore = score
                bestPartialMatch = candidate
            }
        }
    }
    
    return bestPartialMatch, ""
}

// Strip common prepositions for better matching
var strippablePrepositions = []string{
    "onto", "into", "over", "to", "toward", "towards",
    "from", "in", "under", "upon", "with", "the", "my",
}

func StripPrepositions(input string) string {
    words := strings.Fields(strings.ToLower(input))
    filtered := []string{}
    
    for _, word := range words {
        isPreposition := false
        for _, prep := range strippablePrepositions {
            if word == prep {
                isPreposition = true
                break
            }
        }
        if !isPreposition {
            filtered = append(filtered, word)
        }
    }
    
    return strings.Join(filtered, " ")
}
```

### Color Code Processing
```go
var colorShortTagRegex = regexp.MustCompile(`\{(\d*)(?::)?(\d*)?\}`)

// Convert short color tags to full ANSI codes
func ConvertColorShortTags(input string) string {
    return colorShortTagRegex.ReplaceAllStringFunc(input, func(match string) string {
        // Extract color codes and convert to ANSI
        return convertToAnsiColor(match)
    })
}

// Strip all color codes for plain text output
func StripColorCodes(input string) string {
    // Remove ANSI escape sequences
    ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
    stripped := ansiRegex.ReplaceAllString(input, "")
    
    // Remove custom color tags
    stripped = colorShortTagRegex.ReplaceAllString(stripped, "")
    
    return stripped
}
```

### Filename Sanitization
```go
// Convert text to safe filename across all platforms
func ConvertForFilename(input string) string {
    // Remove/replace unsafe characters
    unsafe := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
    safe := unsafe.ReplaceAllString(input, "_")
    
    // Handle reserved names on Windows
    reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", 
                         "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", 
                         "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", 
                         "LPT7", "LPT8", "LPT9"}
    
    upper := strings.ToUpper(safe)
    for _, name := range reserved {
        if upper == name {
            safe = "_" + safe
            break
        }
    }
    
    // Trim spaces and dots
    safe = strings.Trim(safe, " .")
    
    // Ensure not empty
    if safe == "" {
        safe = "unnamed"
    }
    
    return safe
}

// Cross-platform file path construction
func FilePath(parts ...string) string {
    return filepath.Join(parts...)
}
```

## Cryptographic Services

### Password Hashing
```go
// Secure password hashing with SHA-256
func Hash(input string) string {
    hasher := sha256.New()
    hasher.Write([]byte(input))
    return hex.EncodeToString(hasher.Sum(nil))
}

// Legacy MD5 support for compatibility
func Md5Hash(input string) string {
    hasher := md5.New()
    hasher.Write([]byte(input))
    return hex.EncodeToString(hasher.Sum(nil))
}

// Base64 encoding utilities
func EncodeBase64(data []byte) string {
    return base64.StdEncoding.EncodeToString(data)
}

func DecodeBase64(encoded string) ([]byte, error) {
    return base64.StdEncoding.DecodeString(encoded)
}
```

### Random Number Generation
```go
// Seeded random number generation
func Rand(max int) int {
    if max <= 0 {
        return 0
    }
    return rand.Intn(max)
}

// Dice rolling simulation
func RollDice(count, sides int) int {
    if count <= 0 || sides <= 0 {
        return 0
    }
    
    total := 0
    for i := 0; i < count; i++ {
        total += rand.Intn(sides) + 1
    }
    return total
}

// Percentage chance evaluation
func RollPercent(chance int) bool {
    if chance <= 0 {
        return false
    }
    if chance >= 100 {
        return true
    }
    return rand.Intn(100) < chance
}
```

## Memory Management System

### Memory Tracking
```go
type MemReport func() map[string]MemoryResult

type MemoryResult struct {
    Memory uint64 // Memory usage in bytes
    Count  int    // Number of items
}

var (
    memoryTrackerNames []string
    memoryTrackers     []MemReport
)

// Register memory tracking for a system
func AddMemoryReporter(name string, reporter MemReport) {
    memoryTrackerNames = append(memoryTrackerNames, name)
    memoryTrackers = append(memoryTrackers, reporter)
}

// Get comprehensive memory report
func GetMemoryReport() (names []string, trackedResults []map[string]MemoryResult) {
    names = append([]string{}, memoryTrackerNames...)
    trackedResults = []map[string]MemoryResult{}
    
    for _, reporter := range memoryTrackers {
        trackedResults = append(trackedResults, reporter())
    }
    
    return names, trackedResults
}

// Calculate memory usage of any data structure
func MemoryUsage(v interface{}) uint64 {
    return calculateMemoryUsage(reflect.ValueOf(v), make(map[uintptr]bool))
}

func calculateMemoryUsage(v reflect.Value, visited map[uintptr]bool) uint64 {
    if !v.IsValid() {
        return 0
    }
    
    var size uint64
    
    switch v.Kind() {
    case reflect.Ptr, reflect.Interface:
        if v.IsNil() {
            return 0
        }
        
        ptr := v.Pointer()
        if visited[ptr] {
            return 0 // Avoid infinite loops
        }
        visited[ptr] = true
        
        size += 8 // Pointer size
        size += calculateMemoryUsage(v.Elem(), visited)
        
    case reflect.Slice:
        size += 24 // Slice header
        for i := 0; i < v.Len(); i++ {
            size += calculateMemoryUsage(v.Index(i), visited)
        }
        
    case reflect.Map:
        size += 8 // Map header
        for _, key := range v.MapKeys() {
            size += calculateMemoryUsage(key, visited)
            size += calculateMemoryUsage(v.MapIndex(key), visited)
        }
        
    case reflect.String:
        size += uint64(v.Len())
        
    case reflect.Struct:
        for i := 0; i < v.NumField(); i++ {
            size += calculateMemoryUsage(v.Field(i), visited)
        }
        
    default:
        size += uint64(v.Type().Size())
    }
    
    return size
}
```

### System Statistics
```go
// Get comprehensive server statistics
func ServerStats() string {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    return fmt.Sprintf(
        "Heap: %dMB  Largest Heap: %dMB\n"+
        "Stack: %dMB\n"+
        "Total Mem: %dMB\n"+
        "GC Count: %d\n"+
        "NumCPU: %d\n",
        bToMb(m.HeapInuse),
        bToMb(m.HeapSys),
        bToMb(m.StackSys),
        bToMb(m.Sys),
        m.NumGC,
        runtime.NumCPU(),
    )
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}
```

## File and Data Operations

### File Utilities
```go
// Check if file exists
func FileExists(filename string) bool {
    _, err := os.Stat(filename)
    return err == nil
}

// Create directory if it doesn't exist
func EnsureDirectory(path string) error {
    return os.MkdirAll(path, 0755)
}

// Copy file contents
func CopyFile(src, dst string) error {
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()
    
    destFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destFile.Close()
    
    _, err = io.Copy(destFile, sourceFile)
    return err
}
```

### Data Compression
```go
// Compress data using gzip
func CompressData(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    writer := gzip.NewWriter(&buf)
    
    _, err := writer.Write(data)
    if err != nil {
        return nil, err
    }
    
    err = writer.Close()
    if err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}

// Decompress gzip data
func DecompressData(data []byte) ([]byte, error) {
    reader, err := gzip.NewReader(bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    return io.ReadAll(reader)
}
```

## Text Processing and Internationalization

### Multi-language Support
```go
// Regular expressions for different text types
var (
    // CJK character support (Chinese, Japanese, Korean)
    wordRegex = regexp.MustCompile(`([\p{Han}\p{Hiragana}\p{Katakana}\p{Hangul}]|\w+|[\p{P}\p{S}\s]+)`)
    punctuationRegex = regexp.MustCompile(`[\p{P}]+`)
)

// Count visual width accounting for CJK characters
func VisualWidth(text string) int {
    return runewidth.StringWidth(text)
}

// Word wrapping with CJK support
func WordWrap(text string, width int) []string {
    if width <= 0 {
        return []string{text}
    }
    
    words := wordRegex.FindAllString(text, -1)
    lines := []string{}
    currentLine := ""
    currentWidth := 0
    
    for _, word := range words {
        wordWidth := runewidth.StringWidth(word)
        
        if currentWidth+wordWidth > width && currentLine != "" {
            lines = append(lines, currentLine)
            currentLine = word
            currentWidth = wordWidth
        } else {
            currentLine += word
            currentWidth += wordWidth
        }
    }
    
    if currentLine != "" {
        lines = append(lines, currentLine)
    }
    
    return lines
}
```

## Network and Server Utilities

### Server Management
```go
var serverAddr string = "Unknown"

// Set server address for identification
func SetServerAddress(addr string) {
    serverAddr = addr
}

func GetServerAddress() string {
    return serverAddr
}

// HTTP utility functions
func IsValidURL(url string) bool {
    _, err := http.Get(url)
    return err == nil
}

// Network address validation
func ValidateIPAddress(ip string) bool {
    return net.ParseIP(ip) != nil
}
```

## Integration Patterns

### Performance Monitoring Integration
```go
// Track function execution time
func TrackExecutionTime(name string, fn func()) {
    start := time.Now()
    fn()
    duration := time.Since(start)
    TrackTime(name, duration.Seconds())
}

// Memory usage monitoring
func MonitorMemoryUsage(system string, data interface{}) {
    usage := MemoryUsage(data)
    mudlog.Debug("Memory Usage", "system", system, "bytes", usage, "mb", usage/1024/1024)
}
```

### Thread Safety Integration
```go
// Safe game data access pattern
func SafeGameOperation(operation func()) {
    LockMud()
    defer UnlockMud()
    operation()
}

// Safe game data reading pattern
func SafeGameRead(reader func()) {
    RLockMud()
    defer RUnlockMud()
    reader()
}
```

## Usage Examples

### Performance Tracking
```go
// Track command execution time
start := time.Now()
executePlayerCommand(user, command)
duration := time.Since(start)
util.TrackTime("player_commands", duration.Seconds())

// Get performance report
trackers := util.GetTimeTrackers()
for _, tracker := range trackers {
    fmt.Printf("%s: avg=%.3fs, min=%.3fs, max=%.3fs, count=%.0f\n",
        tracker.Name, tracker.Average, tracker.Lowest, tracker.Highest, tracker.Count)
}
```

### Text Processing
```go
// Fuzzy item search
itemName := "rusty sword"
items := []string{"Rusty Iron Sword", "Sharp Blade", "Old Rusty Dagger"}

closeMatch, exactMatch := util.FindMatchIn(itemName, items...)
if exactMatch != "" {
    fmt.Printf("Found exact match: %s\n", exactMatch)
} else if closeMatch != "" {
    fmt.Printf("Found close match: %s\n", closeMatch)
}

// Filename sanitization
playerName := "Player/Name<With>Bad:Characters"
safeFilename := util.ConvertForFilename(playerName)
// Result: "Player_Name_With_Bad_Characters"
```

### Memory Monitoring
```go
// Register memory reporter for a system
util.AddMemoryReporter("Users", func() map[string]util.MemoryResult {
    return map[string]util.MemoryResult{
        "active_users": {util.MemoryUsage(activeUsers), len(activeUsers)},
        "user_cache":   {util.MemoryUsage(userCache), len(userCache)},
    }
})

// Get memory report
names, results := util.GetMemoryReport()
for i, name := range names {
    fmt.Printf("System: %s\n", name)
    for component, result := range results[i] {
        fmt.Printf("  %s: %d bytes (%d items)\n", 
            component, result.Memory, result.Count)
    }
}
```

## Dependencies

- `crypto/sha256` - Secure password hashing
- `crypto/md5` - Legacy hash support
- `compress/gzip` - Data compression utilities
- `github.com/mattn/go-runewidth` - CJK character width calculation
- `internal/mudlog` - Logging system integration
- `internal/term` - Terminal control and ANSI codes

This comprehensive util system provides the essential foundation for all GoMud operations with thread-safe utilities, performance monitoring, advanced text processing, and robust system management capabilities.