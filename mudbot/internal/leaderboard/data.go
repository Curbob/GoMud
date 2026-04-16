package leaderboard

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var roomNameMap = map[int]string{
	2003: "Con Floor",
	2004: "North Hallway",
	2005: "Main Talk Track",
	2010: "Bar",
	2011: "CTF Arena",
	2020: "Lockpick Village",
	2022: "Hardware Hacking",
	2024: "Village Hall",
	2028: "SAV Lounge",
	2032: "Trivia",
	2033: "Wireless Shootout",
	2034: "Chaos Workshop",
	2050: "Service Hallway",
	2051: "Server Room",
}

// UserData represents the YAML structure from GoMud user files
type UserData struct {
	UserID   int    `yaml:"userid"`
	Role     string `yaml:"role"`
	Username string `yaml:"username"`
	Joined   time.Time `yaml:"joined"`
	Character struct {
		Name        string `yaml:"name"`
		Level       int    `yaml:"level"`
		Experience  int    `yaml:"experience"`
		Gold        int    `yaml:"gold"`
		Bank        int    `yaml:"bank"`
		RoomID      int    `yaml:"roomid"`
		Zone        string `yaml:"zone"`
		KillDeath   struct {
			TotalKills int `yaml:"totalkills"`
			TotalDeaths int `yaml:"totaldeaths"`
		} `yaml:"kd"`
	} `yaml:"character"`
	Created time.Time `yaml:"created"`
}

// DataProvider handles reading GoMud user data
type DataProvider struct {
	usersPath string
	cache     map[int]*UserData
	lastLoad  time.Time
}

// NewDataProvider creates a new data provider for the given users directory
func NewDataProvider(usersPath string) *DataProvider {
	return &DataProvider{
		usersPath: usersPath,
		cache:     make(map[int]*UserData),
	}
}

// LoadUsers reads all user files from the GoMud users directory
func (dp *DataProvider) LoadUsers() error {
	files, err := ioutil.ReadDir(dp.usersPath)
	if err != nil {
		return fmt.Errorf("failed to read users directory %s: %v", dp.usersPath, err)
	}

	newCache := make(map[int]*UserData)
	
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		// Extract user ID from filename (e.g., "1.yaml" -> 1)
		userIdStr := strings.TrimSuffix(file.Name(), ".yaml")
		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			continue // Skip non-numeric filenames
		}

		// Read user file
		filePath := filepath.Join(dp.usersPath, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		// Parse YAML
		var userData UserData
		err = yaml.Unmarshal(data, &userData)
		if err != nil {
			continue // Skip malformed files
		}

		// Skip admin accounts for leaderboards
		if userData.Role == "admin" {
			continue
		}

		newCache[userId] = &userData
	}

	dp.cache = newCache
	dp.lastLoad = time.Now()
	return nil
}

// GetLeaderboard generates a leaderboard of the specified type
func (dp *DataProvider) GetLeaderboard(lbType LeaderboardType, limit int) (*LeaderboardResponse, error) {
	// Ensure data is loaded
	if time.Since(dp.lastLoad) > 5*time.Minute {
		if err := dp.LoadUsers(); err != nil {
			return nil, fmt.Errorf("failed to load user data: %v", err)
		}
	}

	var entries []LeaderboardEntry
	
	for _, userData := range dp.cache {
		var value int
		var formattedValue string

		switch lbType {
		case LeaderboardLevel:
			value = userData.Character.Level
			formattedValue = fmt.Sprintf("Level %d", value)
			
		case LeaderboardGold:
			value = userData.Character.Gold + userData.Character.Bank
			formattedValue = fmt.Sprintf("%d gold", value)
			
		case LeaderboardKills:
			value = userData.Character.KillDeath.TotalKills
			formattedValue = fmt.Sprintf("%d kills", value)
			
		case LeaderboardFishing:
			// TODO: Need to track fishing stats in user data
			// For now, skip fishing leaderboard
			continue
			
		case LeaderboardGambling:
			// TODO: Need to track gambling stats in user data
			// For now, skip gambling leaderboard
			continue
			
		default:
			return nil, fmt.Errorf("unknown leaderboard type: %s", lbType)
		}

		// Skip zero values for most leaderboards (except level, which starts at 1)
		if value == 0 && lbType != LeaderboardLevel {
			continue
		}

		entries = append(entries, LeaderboardEntry{
			CharacterName:  userData.Character.Name,
			Value:          value,
			FormattedValue: formattedValue,
		})
	}

	// Sort by value (highest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value > entries[j].Value
	})

	// Limit results and assign ranks
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	
	for i := range entries {
		entries[i].Rank = i + 1
	}

	// Create response
	response := &LeaderboardResponse{
		Type:         lbType,
		Entries:      entries,
		LastUpdated:  time.Now(),
		TotalPlayers: len(dp.cache),
	}

	// Set title and description based on type
	switch lbType {
	case LeaderboardLevel:
		response.Title = "🏆 Top Players by Level"
		response.Description = "Highest level characters in the MUD"
	case LeaderboardGold:
		response.Title = "💰 Richest Players"
		response.Description = "Players with the most gold (including bank)"
	case LeaderboardKills:
		response.Title = "⚔️ Most Dangerous Players"
		response.Description = "Players with the most kills"
	case LeaderboardFishing:
		response.Title = "🎣 Master Anglers"
		response.Description = "Top fishing achievements"
	case LeaderboardGambling:
		response.Title = "🎰 Casino High Rollers"
		response.Description = "Biggest gambling winners"
	}

	return response, nil
}

// GetOnlinePlayers returns the currently loaded players and their locations.
func (dp *DataProvider) GetOnlinePlayers() ([]OnlinePlayer, error) {
	if time.Since(dp.lastLoad) > 5*time.Minute {
		if err := dp.LoadUsers(); err != nil {
			return nil, fmt.Errorf("failed to load user data: %v", err)
		}
	}

	players := make([]OnlinePlayer, 0, len(dp.cache))
	for _, userData := range dp.cache {
		location := userData.Character.Zone
		if roomName, ok := roomNameMap[userData.Character.RoomID]; ok {
			location = roomName
		}

		players = append(players, OnlinePlayer{
			Username:      userData.Username,
			CharacterName: userData.Character.Name,
			Zone:          userData.Character.Zone,
			RoomID:        userData.Character.RoomID,
			Location:      location,
		})
	}

	 sort.Slice(players, func(i, j int) bool {
		return strings.ToLower(players[i].CharacterName) < strings.ToLower(players[j].CharacterName)
	})

	return players, nil
}

// GetServerStatus returns current server status.
func (dp *DataProvider) GetServerStatus() (*ServerStatus, error) {
	onlinePlayers, err := dp.GetOnlinePlayers()
	if err != nil {
		return nil, err
	}

	return &ServerStatus{
		Status:          "online",
		PlayersOnline:   len(onlinePlayers),
		MaxPlayers:      200,
		Uptime:          "Unknown",
		ServerStartTime: time.Now().Add(-24 * time.Hour),
		LastUpdated:     time.Now(),
		OnlinePlayers:   onlinePlayers,
	}, nil
}

// GetConnectionInfo returns server connection details
func (dp *DataProvider) GetConnectionInfo() *ConnectionInfo {
	return &ConnectionInfo{
		Host:     "mud.cackalackycon.org", // TODO: Make configurable
		Ports:    []int{33333, 44444},
		GameType: "Multi-User Dungeon (MUD)",
		Instructions: []string{
			"Use any MUD client (recommended: Mudlet)",
			"Or connect via telnet: `telnet mud.cackalackycon.org 33333`",
			"Create a new character and start exploring!",
			"Type 'help newbie' in-game for getting started tips",
		},
	}
}