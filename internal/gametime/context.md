# GoMud GameTime System Context

## Overview

The GoMud gametime system provides a comprehensive fantasy calendar and time management framework with day/night cycles, custom month names, zodiac systems, and time-based event scheduling. It features round-based time progression, period calculations, visual time representation, and integration with game events and mechanics.

## Architecture

The gametime system is built around several key components:

### Core Components

**GameDate Structure:**
- Round-based time tracking with configurable day/night cycles
- 12-month fantasy calendar with custom month names
- Hour/minute precision with AM/PM formatting
- Visual time representation with day/night symbols and colors

**Time Period System:**
- String-based period definitions (e.g., "1 day", "3 hours", "2 weeks")
- Automatic period calculation and conversion to rounds
- Timer expiration tracking and validation
- Integration with buff durations and event scheduling

**Zodiac System:**
- 228 unique zodiac animals for year identification
- Seeded randomization for consistent server-wide zodiac order
- Fantasy-themed creatures including mythical beings
- Year-based zodiac calculation with modular cycling

**Day/Night Mechanics:**
- Configurable day/night cycle lengths
- Sunrise/sunset event timing
- Visual indicators for time of day (symbols and colors)
- Dusk transition periods for atmospheric effects

## Key Features

### 1. **Fantasy Calendar System**
- **12 Custom Months**: Arvalon, Beldris, Celmara, Durelin, Esmira, Ferulan, Glimar, Hestara, Irinel, Jorenth, Keldris, Luneth
- **Round-Based Progression**: Time advances based on game rounds rather than real time
- **Configurable Cycles**: Adjustable day/night cycle lengths and timing
- **Precise Time Tracking**: Hour and minute precision with floating-point accuracy

### 2. **Visual Time Representation**
- **Day/Night Symbols**: Sun (☀️) for day, moon (☾) for night
- **Color-Coded Display**: Different colors for day, dusk, and night periods
- **Atmospheric Transitions**: Dusk period detection for gradual transitions
- **Compact Display**: Symbol-only mode for space-constrained interfaces

### 3. **Advanced Period Calculations**
- **Flexible Period Strings**: Support for "1 day", "3 hours", "2 weeks" format
- **Automatic Conversion**: Period strings converted to round counts
- **Timer Management**: Expiration tracking for time-based events
- **Integration Ready**: Seamless integration with buff and event systems

### 4. **Zodiac and Lore System**
- **Rich Creature List**: 228 unique animals including fantasy creatures
- **Seeded Randomization**: Consistent zodiac order across server restarts
- **Year-Based Cycling**: Predictable zodiac progression for lore consistency
- **Fantasy Integration**: Dragons, phoenixes, griffins, and other mythical beings

## GameDate Structure

### Core GameDate Properties
```go
type GameDate struct {
    RoundNumber      uint64  // The round this date represents
    RoundsPerDay     int     // Configurable day length
    NightHoursPerDay int     // Hours of night per day
    
    // Calendar Components
    Year        int     // Game year
    Month       int     // Month (1-12)
    Week        int     // Week of year
    Day         int     // Day of month
    
    // Time Components
    Hour        int     // 12-hour format hour
    Hour24      int     // 24-hour format hour
    Minute      int     // Minute (0-59)
    MinuteFloat float64 // Precise minute with fractional seconds
    AmPm        string  // "AM" or "PM"
    
    // Day/Night Cycle
    Night       bool    // Is it currently night?
    DayStart    int     // Hour when day begins
    NightStart  int     // Hour when night begins
}
```

### Time Display and Formatting
```go
// Standard time display with color coding
func (gd GameDate) String(symbolOnly ...bool) string {
    dayNight := "day"
    if gd.Night {
        dayNight = "night"
    } else {
        // Check for dusk transition
        hoursLeft := int(math.Abs(float64(gd.Hour24) - float64(gd.NightStart)))
        if hoursLeft < 3 {
            dayNight = "day-dusk"
        }
    }
    
    // Symbol-only display for compact representation
    if len(symbolOnly) > 0 && symbolOnly[0] {
        if gd.Night {
            return `<ansi fg="night">☾</ansi>`
        }
        return fmt.Sprintf(`<ansi fg="%s">☀️</ansi>`, dayNight)
    }
    
    // Full time display with color coding
    return fmt.Sprintf("<ansi fg=\"%s\">%d:%02d%s</ansi>", 
                      dayNight, gd.Hour, gd.Minute, gd.AmPm)
}
```

## Time Period System

### RoundTimer for Event Scheduling
```go
type RoundTimer struct {
    RoundStart uint64 // Starting round for the timer
    Period     string // Period duration (e.g., "1 day", "3 hours")
    gd         GameDate // Cached GameDate for efficiency
}

// Check if timer has expired
func (r RoundTimer) Expired() bool {
    if r.Period == "" || r.RoundStart == 0 {
        return true
    }
    
    // Lazy load GameDate if not cached
    if r.gd.RoundNumber == 0 {
        r.gd = GetDate(r.RoundStart)
    }
    
    // Check if current round exceeds timer expiration
    return r.gd.AddPeriod(r.Period) < util.GetRoundCount()
}
```

### Period String Processing
```go
// Add time period to current date
func (gd GameDate) AddPeriod(period string) uint64 {
    // Parse period strings like:
    // "1 day", "3 hours", "2 weeks", "30 minutes"
    
    // Implementation converts period to rounds based on:
    // - Game configuration for rounds per day/hour
    // - Calendar structure for days/weeks/months
    // - Returns new round number after period addition
    
    return newRoundNumber
}

// Get the last occurrence of a specific period
func GetLastPeriod(period string, currentRound uint64) uint64 {
    // Find the most recent round when specified period occurred
    // Useful for events like "last sunset", "last midnight"
    return lastOccurrenceRound
}
```

## Calendar and Month System

### Fantasy Month Names
```go
var monthNames = []string{
    "Arvalon",   // Month 1 - Beginning of year
    "Beldris",   // Month 2
    "Celmara",   // Month 3
    "Durelin",   // Month 4
    "Esmira",    // Month 5
    "Ferulan",   // Month 6
    "Glimar",    // Month 7
    "Hestara",   // Month 8
    "Irinel",    // Month 9
    "Jorenth",   // Month 10
    "Keldris",   // Month 11
    "Luneth",    // Month 12 - End of year
}

// Get month name by number (1-12)
func MonthName(month int) string {
    month-- // Convert to 0-based index
    return monthNames[month%len(monthNames)]
}
```

### Date Calculation and Caching
```go
var roundDateCache = map[uint64]GameDate{}

// Get GameDate for specific round with caching
func GetDate(roundNumber uint64) GameDate {
    if cached, exists := roundDateCache[roundNumber]; exists {
        return cached
    }
    
    // Calculate new GameDate
    gd := calculateGameDate(roundNumber)
    
    // Cache for future lookups
    roundDateCache[roundNumber] = gd
    
    return gd
}

func calculateGameDate(roundNumber uint64) GameDate {
    // Complex calculation involving:
    // - Rounds per day configuration
    // - Calendar math for year/month/day
    // - Time of day calculation
    // - Day/night cycle determination
    
    return gameDate
}
```

## Zodiac System

### Fantasy Creature Zodiac
```go
var zodiacAnimals = []string{
    // Real animals
    "Aardvark", "Albatross", "Alligator", "Alpaca", "Ant", "Antelope",
    "Baboon", "Badger", "Bear", "Beaver", "Bee", "Beetle", "Bison",
    "Boar", "Buffalo", "Butterfly", "Camel", "Cat", "Cheetah", "Chicken",
    
    // Fantasy creatures
    "Amphiptere", "Basilisk", "Centaur", "Cerberus", "Chimera", "Cockatrice",
    "Cyclops", "Dragon", "Dryad", "Fairy", "Firebird", "Gargoyle", "Griffin",
    "Harpy", "Hippogriff", "Hydra", "Ifrit", "Jabberwocky", "Kraken", "Lamia",
    "Leviathan", "Manticore", "Medusa", "Mermaid", "Minotaur", "Naga", "Nymph",
    "Ogre", "Pegasus", "Phoenix", "Pixie", "Poltergeist", "Questing Beast", "Roc",
    
    // And 190+ more creatures...
}

// Get zodiac animal for specific year
func GetZodiac(year int) string {
    if !randomized {
        randomizeZodiac()
    }
    return zodiacAnimals[year%len(zodiacAnimals)]
}

// Seeded randomization for consistent server-wide zodiac order
func randomizeZodiac() {
    r := rand.New(rand.NewSource(configs.GetConfig().SeedInt()))
    r.Shuffle(len(zodiacAnimals), func(i, j int) { 
        zodiacAnimals[i], zodiacAnimals[j] = zodiacAnimals[j], zodiacAnimals[i] 
    })
    randomized = true
}
```

## Time Manipulation and Events

### Day/Night Cycle Control
```go
// Jump to next night period
func SetToNight(roundAdjustment ...int) {
    // Find last sunset
    dayRound := GetLastPeriod("sunset", util.GetRoundCount())
    
    // Apply optional round adjustment
    if len(roundAdjustment) > 0 {
        if roundAdjustment[0] < 0 {
            dayRound -= uint64(-1 * roundAdjustment[0])
        } else {
            dayRound += uint64(roundAdjustment[0])
        }
    }
    
    // Advance to next night and update game time
    gd := GetDate(dayRound).Add(0, 1, 0) // Add 1 day
    util.SetRoundCount(gd.RoundNumber)
}

// Jump to next day period
func SetToDay(roundAdjustment ...int) {
    // Find last sunrise
    nightRound := GetLastPeriod("sunrise", util.GetRoundCount())
    
    // Apply adjustment and advance time
    if len(roundAdjustment) > 0 {
        nightRound += uint64(roundAdjustment[0])
    }
    
    gd := GetDate(nightRound).Add(0, 1, 0)
    util.SetRoundCount(gd.RoundNumber)
}
```

### Time-Based Event Integration
```go
// Check for day/night transitions
func CheckDayNightTransition(previousRound, currentRound uint64) (bool, string) {
    prevDate := GetDate(previousRound)
    currDate := GetDate(currentRound)
    
    // Detect transitions
    if !prevDate.Night && currDate.Night {
        return true, "sunset"
    } else if prevDate.Night && !currDate.Night {
        return true, "sunrise"
    }
    
    return false, ""
}

// Schedule time-based events
func ScheduleTimeEvent(eventType string, targetTime GameDate) {
    events.AddToQueue(events.TimeEvent{
        EventType:   eventType,
        TargetRound: targetTime.RoundNumber,
        GameDate:    targetTime,
    })
}
```

## Integration Patterns

### Buff System Integration
```go
// Time-based buff durations
buffTimer := RoundTimer{
    RoundStart: util.GetRoundCount(),
    Period:     "30 minutes", // Buff lasts 30 minutes
}

// Check expiration in buff processing
if buffTimer.Expired() {
    character.Buffs.RemoveBuff(buffId)
}
```

### Event System Integration
```go
// Day/night cycle events
currentDate := GetDate(util.GetRoundCount())
if currentDate.Night != previousNight {
    if currentDate.Night {
        events.AddToQueue(events.DayNightCycle{
            EventType: "sunset",
            GameDate:  currentDate,
        })
    } else {
        events.AddToQueue(events.DayNightCycle{
            EventType: "sunrise", 
            GameDate:  currentDate,
        })
    }
}
```

### Configuration Integration
```go
// Time system uses game configuration
timeConfig := configs.GetTimeConfig()
gameDate := GameDate{
    RoundsPerDay:     timeConfig.RoundsPerDay,
    NightHoursPerDay: timeConfig.NightHours,
    DayStart:         timeConfig.DayStartHour,
    NightStart:       timeConfig.NightStartHour,
}
```

## Usage Examples

### Basic Time Display
```go
// Get current game time
currentRound := util.GetRoundCount()
gameDate := gametime.GetDate(currentRound)

// Display time in chat
timeString := gameDate.String()
user.SendText(fmt.Sprintf("Current time: %s", timeString))

// Display compact time symbol
symbolOnly := gameDate.String(true)
user.SendText(fmt.Sprintf("Time: %s", symbolOnly))
```

### Date and Calendar Information
```go
// Get full date information
gameDate := gametime.GetDate(util.GetRoundCount())

user.SendText(fmt.Sprintf("Date: %s %d, Year %d (%s)", 
    gametime.MonthName(gameDate.Month),
    gameDate.Day,
    gameDate.Year,
    gametime.GetZodiac(gameDate.Year)))

// Example output: "Date: Arvalon 15, Year 1247 (Dragon)"
```

### Time-Based Scheduling
```go
// Schedule event for specific time
targetDate := gametime.GetDate(util.GetRoundCount())
futureDate := targetDate.Add(1, 0, 0) // Add 1 day

timer := gametime.RoundTimer{
    RoundStart: util.GetRoundCount(),
    Period:     "1 day",
}

// Check later if event should trigger
if timer.Expired() {
    triggerScheduledEvent()
}
```

### Day/Night Cycle Usage
```go
// Check current time of day
gameDate := gametime.GetDate(util.GetRoundCount())

if gameDate.Night {
    // Night-time activities
    spawnNightCreatures()
    user.SendText("The creatures of the night begin to stir...")
} else {
    // Day-time activities
    if gameDate.Hour24 >= gameDate.NightStart - 2 {
        user.SendText("Dusk approaches, and shadows grow long...")
    }
}
```

### Administrative Time Control
```go
// Admin command to set time
func AdminSetNight() {
    gametime.SetToNight()
    broadcastMessage("An administrator has brought forth the night!")
}

func AdminSetDay() {
    gametime.SetToDay()
    broadcastMessage("An administrator has summoned the dawn!")
}

// Set specific time with adjustment
func AdminSetTime(adjustment int) {
    gametime.SetToNight(adjustment) // Set to night with round offset
}
```

## Dependencies

- `internal/configs` - Configuration management for time system settings
- `internal/util` - Round counting and game timing utilities
- `internal/mudlog` - Logging system for debugging time calculations

This comprehensive gametime system provides immersive fantasy calendar functionality with sophisticated time management, visual representation, and seamless integration with game events and mechanics while maintaining performance through intelligent caching and efficient calculations.