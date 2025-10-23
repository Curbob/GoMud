# GoMud Skills System Context

## Overview

The GoMud skills system provides a profession-based character development framework with 15 distinct skills organized into 10 professions. It features a 4-level progression system, experience-based titles, and synergistic skill combinations that emphasize capability expansion over raw power increases.

## Architecture

The skills system is built around several key components:

### Core Components

**Skill Definition System:**
- String-based skill tags with hierarchical subtag support
- 15 core skills with specific training locations and requirements
- Level-based progression from 1-4 with increasing costs
- Capability-focused design philosophy

**Profession Framework:**
- 10 distinct professions with unique skill combinations
- Dynamic profession ranking based on skill investment
- Experience titles reflecting overall mastery level
- Multi-profession specialization support

**Progression Mechanics:**
- Exponential skill point costs (1+2+3+4 = 10 points for max level)
- Completion percentage calculations for profession ranking
- Flexible specialization allowing deep or broad development

## Key Features

### 1. **Comprehensive Skill Set**
- **Combat Skills**: Dual Wield, Brawling, Protection
- **Magic Skills**: Cast, Enchant, Scribe
- **Exploration Skills**: Map, Portal, Search, Track
- **Utility Skills**: Peep, Inspect, Skulduggery, Tame, Trading

### 2. **Profession System**
- 10 distinct career paths with thematic skill groupings
- Dynamic profession assignment based on skill investment
- Multi-profession mastery recognition ("demigod" status)
- Experience-based titles (scrub → novice → apprentice → journeyman → expert)

### 3. **Flexible Progression**
- Choice between specialization (deep) and diversification (wide)
- Skill synergies at maximum levels
- Capability-based advancement rather than power scaling
- Equipment integration through stat modifications

## Skill Structure

### Skill Tag System
```go
type SkillTag string

// Hierarchical skill identification with subtag support
func (s SkillTag) String(subtag ...string) string {
    result := string(s)
    if len(subtag) > 0 {
        result += ":" + strings.Join(subtag, ":")
    }
    return result
}

// Create skill subtags for specialized abilities
func (s SkillTag) Sub(subtag string) SkillTag {
    return SkillTag(string(s) + subtag)
}
```

### Core Skills Enumeration
```go
const (
    // Magic and Spellcasting
    Cast        SkillTag = "cast"        // Magic Academy training
    Enchant     SkillTag = "enchant"     // Item enhancement
    Scribe      SkillTag = "scribe"      // Dark Acolyte's Chamber
    
    // Combat and Weapons
    DualWield   SkillTag = "dual-wield"  // Fisherman's house
    Brawling    SkillTag = "brawling"    // Soldiers Training Yard
    Protection  SkillTag = "protection"  // Defensive abilities
    
    // Exploration and Navigation
    Map         SkillTag = "map"         // Frostwarden Rangers
    Portal      SkillTag = "portal"      // Obelisk activation
    Search      SkillTag = "search"      // Frostwarden Rangers
    Track       SkillTag = "track"       // Frostwarden Rangers
    
    // Utility and Social
    Peep        SkillTag = "peep"        // Information gathering
    Inspect     SkillTag = "inspect"     // Item analysis
    Skulduggery SkillTag = "skulduggery" // Thieves Den
    Tame        SkillTag = "tame"        // Monster taming
    Trading     SkillTag = "trading"     // Commerce abilities
)
```

## Profession System

### Profession Definitions
```go
var Professions = map[string][]SkillTag{
    "treasure hunter": {Map, Search, Peep, Inspect, Trading},
    "assassin":        {Skulduggery, DualWield, Track},
    "explorer":        {Map, Portal, Scribe},
    "arcane scholar":  {Enchant, Scribe, Inspect},
    "warrior":         {Brawling, DualWield},
    "paladin":         {Protection, Brawling},
    "ranger":          {Map, Search, Track},
    "monster hunter":  {Tame, Track},
    "sorcerer":        {Cast, Enchant},
    "merchant":        {Peep, Trading},
}
```

### Profession Ranking System
```go
type ProfessionRank struct {
    Profession       string   // Profession name
    ExperienceTitle  string   // Experience level title
    TotalPointsSpent float64  // Points invested in profession
    PointsToMax      float64  // Maximum possible points
    Completion       float64  // Completion percentage (0.0-1.0)
    Skills           []string // Skills in this profession
}

// Calculate profession rankings based on skill investments
func GetProfessionRanks(allRanks map[string]int) []ProfessionRank {
    professionList := []ProfessionRank{}
    
    for professionName, skills := range Professions {
        ranking := ProfessionRank{Profession: professionName}
        
        for _, skillName := range skills {
            skillLevel := 0
            if rankVal, ok := allRanks[string(skillName)]; ok {
                skillLevel = rankVal
            }
            
            // Cap at level 4
            if skillLevel > 4 {
                skillLevel = 4
            }
            
            // Calculate cumulative cost: 1+2+3+4 = 10 points max per skill
            totalSkill := (skillLevel * (skillLevel + 1)) / 2
            
            ranking.PointsToMax += 10.0 // Maximum 10 points per skill
            ranking.TotalPointsSpent += float64(totalSkill)
            ranking.Skills = append(ranking.Skills, string(skillName))
        }
        
        ranking.Completion = ranking.TotalPointsSpent / ranking.PointsToMax
        ranking.ExperienceTitle = GetExperienceLevel(ranking.Completion)
        
        professionList = append(professionList, ranking)
    }
    
    return professionList
}
```

### Dynamic Profession Assignment
```go
// Determine primary profession based on highest completion
func GetProfession(allRanks map[string]int) string {
    rankData := GetProfessionRanks(allRanks)
    
    var highestCompletion float64 = 0
    chosenProfessions := []string{}
    experienceName := ""
    
    // Find highest completion percentage
    for _, pRank := range rankData {
        if pRank.Completion == 0 {
            continue
        }
        
        if pRank.Completion > highestCompletion {
            highestCompletion = pRank.Completion
            chosenProfessions = []string{}
        }
        
        if pRank.Completion == highestCompletion {
            experienceName = pRank.ExperienceTitle
            chosenProfessions = append(chosenProfessions, pRank.Profession)
        }
    }
    
    // Handle special cases
    if len(chosenProfessions) < 1 {
        return "scrub" // No skill investment
    }
    
    if len(experienceName) > 0 {
        experienceName = experienceName + " "
    }
    
    // Demigod status for mastering all professions
    if len(chosenProfessions) == len(Professions) {
        return experienceName + "demigod"
    }
    
    // Limit display to 3 professions
    extra := ""
    if len(chosenProfessions) > 3 {
        chosenProfessions = chosenProfessions[0:3]
        extra = " (and more)"
    }
    
    return experienceName + strings.Join(chosenProfessions, "/") + extra
}
```

## Experience Level System

### Experience Title Calculation
```go
// Convert completion percentage to experience title
func GetExperienceLevel(percentage float64) string {
    if percentage >= 0.9 { // ~90% completion (avg level 4)
        return "expert"
    }
    
    if percentage >= 0.6 { // ~60% completion (avg level 3)
        return "journeyman"
    }
    
    if percentage >= 0.3 { // ~30% completion (avg level 2)
        return "apprentice"
    }
    
    if percentage >= 0.1 { // ~10% completion (avg level 1)
        return "novice"
    }
    
    return "scrub" // No meaningful investment
}
```

### Progression Philosophy
```
Skill Level Progression Costs:
- Level 1: 1 skill point  (Total: 1 point)
- Level 2: 2 skill points (Total: 3 points)
- Level 3: 3 skill points (Total: 6 points)
- Level 4: 4 skill points (Total: 10 points)

This creates meaningful choices:
- 10 points = 1 maxed skill OR 10 different level-1 skills
- Encourages specialization vs diversification strategies
- Maximum profession completion requires 50+ skill points
```

## Skill Management Operations

### Skill Validation
```go
// Check if skill exists in the system
func SkillExists(sk string) bool {
    for _, skTag := range allSkillNames {
        if sk == string(skTag) {
            return true
        }
    }
    return false
}

// Get all available skill names
func GetAllSkillNames() []SkillTag {
    return append([]SkillTag{}, allSkillNames...)
}
```

### Skill Discovery and Initialization
```go
// Automatic skill discovery from profession definitions
func init() {
    skillNameSet := map[SkillTag]struct{}{}
    
    // Extract unique skills from all professions
    for _, skills := range Professions {
        for _, skillName := range skills {
            if _, ok := skillNameSet[skillName]; ok {
                continue // Skip duplicates
            }
            
            skillNameSet[skillName] = struct{}{}
            allSkillNames = append(allSkillNames, skillName)
        }
    }
}
```

## Skill Training Locations

### Training Centers and Requirements
```go
// Documented training locations for each skill:

Cast:        // Magic Academy - Room 879 (Levels 1-4)
DualWield:   // Fisherman's House - Room 758 (Levels 1-4)
Map:         // Frostwarden Rangers - Room 74 (Levels 1-4)
Portal:      // Touch Obelisk - Room 871 (Level 1 only)
Search:      // Frostwarden Rangers - Room 74 (Levels 1-4)
Track:       // Frostwarden Rangers - Room 74 (Levels 1-4)
Skulduggery: // Thieves Den - Room 491 (Levels 1-4)
Brawling:    // Soldiers Training Yard - Room 829 (Levels 1-4)
Scribe:      // Dark Acolyte's Chamber - Room 160 (Levels 1-4)
Tame:        // Give mushroom to fairy - Room 558, train Room 830 (Levels 1-4)

// TODO: Training locations for:
// Enchant, Peep, Inspect, Protection, Trading
```

## Design Philosophy and Balance

### Capability vs Power Design
```
Core Design Principles:

1. Skill Levels (1-4) determine CAPABILITIES:
   - New abilities and features unlocked
   - Access to advanced techniques
   - Synergy combinations at max level

2. Stat Points (0-200+) determine EFFECTIVENESS:
   - Success rates and reliability
   - Duration and magnitude of effects
   - Equipment can modify stats, not skill levels

3. Specialization Benefits:
   - Level 4 skills unlock synergies with other skills
   - Deep investment provides unique capabilities
   - Broad investment provides versatility

4. Balance Considerations:
   - Max level skills don't create power imbalance
   - Focus on horizontal progression (new abilities)
   - Vertical progression (effectiveness) through stats
```

### Skill Synergies
```go
// Example synergies at maximum skill levels:

// Track (Level 4) + Map = Mark enemy positions on map
// Cast (Level 4) + Enchant = Advanced spell enchantments
// Skulduggery (Level 4) + Peep = Advanced information gathering
// Tame (Level 4) + Track = Tamed creature tracking abilities
// Protection (Level 4) + Brawling = Defensive combat techniques
```

## Integration Patterns

### Character System Integration
```go
// Skills integrate with character progression
- character.SkillPoints           // Available points for training
- character.Skills[skillName]     // Current skill levels
- character.GetSkillLevel()       // Skill level lookup
- character.Stats                 // Effectiveness modifiers
```

### Combat System Integration
```go
// Skills modify combat behavior
if character.GetSkillLevel(skills.DualWield) >= 3 {
    // Allow dual wielding without penalty
    maxWeapons = 2
}

if character.GetSkillLevel(skills.Brawling) >= 2 {
    // Improved unarmed combat damage
    unarmedBonus += 5
}
```

### Equipment System Integration
```go
// Equipment can enhance skill effectiveness but not levels
- item.StatMod("cast-bonus")      // Improves spell success rate
- item.StatMod("dual-wield-speed") // Faster dual wield attacks
- item.StatMod("search-range")     // Extended search radius
```

## Usage Examples

### Profession Analysis
```go
// Analyze character's profession development
playerSkills := map[string]int{
    "cast":       3,
    "enchant":    2,
    "scribe":     1,
    "dual-wield": 1,
}

// Get profession rankings
rankings := skills.GetProfessionRanks(playerSkills)
for _, rank := range rankings {
    if rank.Completion > 0 {
        fmt.Printf("%s %s: %.1f%% complete\n", 
                  rank.ExperienceTitle, 
                  rank.Profession, 
                  rank.Completion*100)
    }
}

// Get primary profession
profession := skills.GetProfession(playerSkills)
fmt.Printf("Primary profession: %s\n", profession)
// Output: "apprentice sorcerer"
```

### Skill Validation and Discovery
```go
// Validate skill names
if skills.SkillExists("dual-wield") {
    fmt.Println("Dual wield is a valid skill")
}

// Get all available skills
allSkills := skills.GetAllSkillNames()
fmt.Printf("Available skills: %v\n", allSkills)

// Skill tag manipulation
baseSkill := skills.Cast
specializedSkill := baseSkill.Sub(":fireball")
fmt.Printf("Specialized skill: %s\n", specializedSkill.String())
// Output: "cast:fireball"
```

### Training Cost Calculation
```go
// Calculate cost to train skill to specific level
func calculateTrainingCost(currentLevel, targetLevel int) int {
    cost := 0
    for level := currentLevel + 1; level <= targetLevel; level++ {
        cost += level // Level 1=1pt, Level 2=2pts, etc.
    }
    return cost
}

// Examples:
// Level 0 → 1: 1 point
// Level 0 → 2: 3 points (1+2)
// Level 0 → 4: 10 points (1+2+3+4)
// Level 2 → 4: 7 points (3+4)
```

### Profession Specialization Strategies
```go
// Deep specialization example (Sorcerer focus)
deepStrategy := map[string]int{
    "cast":    4, // 10 points
    "enchant": 4, // 10 points
    // Total: 20 points for 100% sorcerer completion
}

// Broad diversification example
broadStrategy := map[string]int{
    "cast":        1, // 1 point
    "dual-wield":  1, // 1 point
    "map":         1, // 1 point
    "search":      1, // 1 point
    "skulduggery": 1, // 1 point
    "brawling":    1, // 1 point
    // Total: 6 points across multiple professions
}
```

## Dependencies

- `strings` - String manipulation for skill tags and profession names

This comprehensive skills system provides flexible character development with meaningful choices between specialization and diversification, thematic profession groupings, and capability-focused progression that integrates seamlessly with the character and combat systems.