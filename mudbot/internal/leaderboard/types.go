package leaderboard

import "time"

// LeaderboardEntry represents a single entry in any leaderboard
type LeaderboardEntry struct {
	Rank         int    `json:"rank"`
	CharacterName string `json:"character_name"`
	Value        int    `json:"value"`
	FormattedValue string `json:"formatted_value,omitempty"`
}

// ServerStatus represents current server information
type ServerStatus struct {
	Status           string    `json:"status"`
	PlayersOnline    int       `json:"players_online"`
	MaxPlayers       int       `json:"max_players"`
	Uptime           string    `json:"uptime"`
	ServerStartTime  time.Time `json:"server_start_time"`
	LastUpdated      time.Time `json:"last_updated"`
	OnlinePlayers    []OnlinePlayer `json:"online_players,omitempty"`
}

// OnlinePlayer represents a currently online character and their location.
type OnlinePlayer struct {
	Username      string `json:"username"`
	CharacterName string `json:"character_name"`
	Zone          string `json:"zone"`
	RoomID        int    `json:"room_id"`
	Location      string `json:"location"`
}

// ConnectionInfo provides server connection details
type ConnectionInfo struct {
	Host        string   `json:"host"`
	Ports       []int    `json:"ports"`
	GameType    string   `json:"game_type"`
	Instructions []string `json:"instructions"`
}

// LeaderboardType represents different types of leaderboards
type LeaderboardType string

const (
	LeaderboardLevel    LeaderboardType = "level"
	LeaderboardGold     LeaderboardType = "gold"
	LeaderboardFishing  LeaderboardType = "fishing"
	LeaderboardGambling LeaderboardType = "gambling"
	LeaderboardKills    LeaderboardType = "kills"
)

// LeaderboardResponse contains the full leaderboard data
type LeaderboardResponse struct {
	Type        LeaderboardType     `json:"type"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Entries     []LeaderboardEntry `json:"entries"`
	LastUpdated time.Time          `json:"last_updated"`
	TotalPlayers int               `json:"total_players"`
}