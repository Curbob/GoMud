package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/mudbot/internal/api"
	"github.com/GoMudEngine/GoMud/mudbot/internal/bot"
	"github.com/GoMudEngine/GoMud/mudbot/internal/leaderboard"
)

func main() {
	var (
		token           = flag.String("token", "", "Discord bot token (or set DISCORD_TOKEN env var)")
		usersPath       = flag.String("users", "", "Path to GoMud users directory")
		guildID         = flag.String("guild", "", "Discord guild ID for slash commands")
		healthPort      = flag.Int("health-port", 8080, "Port for health check HTTP server")
		allowedChannel  = flag.String("channel", "", "Allowed Discord channel ID for bot responses and announcements")
		sayIntervalMins = flag.Int("say-interval", 0, "Minutes between scheduled random sayings (0 disables)")
		quietStart      = flag.Int("quiet-start", 2, "Quiet hours start, 24h local time")
		quietEnd        = flag.Int("quiet-end", 9, "Quiet hours end, 24h local time")
	)
	flag.Parse()

	// Get token from environment if not provided
	if *token == "" {
		*token = os.Getenv("DISCORD_TOKEN")
	}
	if *token == "" {
		log.Fatal("Discord bot token is required (use -token flag or DISCORD_TOKEN env var)")
	}

	// Default users path to GoMud installation
	if *usersPath == "" {
		// Try to find GoMud users directory relative to this binary
		execPath, err := os.Executable()
		if err != nil {
			log.Fatal("Failed to get executable path:", err)
		}

		// Assume mudbot is in GoMud/mudbot/cmd/mudbot/
		goMudRoot := filepath.Join(filepath.Dir(execPath), "..", "..", "..")
		*usersPath = filepath.Join(goMudRoot, "_datafiles", "world", "default", "users")
	}

	// Validate users path exists
	if _, err := os.Stat(*usersPath); os.IsNotExist(err) {
		log.Fatalf("Users directory does not exist: %s", *usersPath)
	}

	// Guild ID is optional for testing
	if *guildID == "" {
		*guildID = os.Getenv("DISCORD_GUILD_ID")
		if *guildID == "" {
			fmt.Println("Warning: No guild ID specified, commands will be registered globally (slower)")
		}
	}

	if *allowedChannel == "" {
		*allowedChannel = os.Getenv("DISCORD_CHANNEL_ID")
	}

	if *sayIntervalMins == 0 {
		if raw := os.Getenv("MUDBOT_SAY_INTERVAL_MINUTES"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				*sayIntervalMins = parsed
			}
		}
	}

	if raw := os.Getenv("MUDBOT_QUIET_START_HOUR"); raw != "" && *quietStart == 2 {
		if parsed, err := strconv.Atoi(raw); err == nil {
			*quietStart = parsed
		}
	}

	if raw := os.Getenv("MUDBOT_QUIET_END_HOUR"); raw != "" && *quietEnd == 9 {
		if parsed, err := strconv.Atoi(raw); err == nil {
			*quietEnd = parsed
		}
	}

	sayInterval := time.Duration(*sayIntervalMins) * time.Minute

	fmt.Printf("MudBot starting...\n")
	fmt.Printf("Users path: %s\n", *usersPath)
	fmt.Printf("Health port: %d\n", *healthPort)
	if *guildID != "" {
		fmt.Printf("Guild ID: %s\n", *guildID)
	}
	if *allowedChannel != "" {
		fmt.Printf("Allowed channel: %s\n", *allowedChannel)
	}
	if sayInterval > 0 {
		fmt.Printf("Scheduled sayings: every %d minutes (quiet %02d:00-%02d:00 local time)\n", *sayIntervalMins, *quietStart, *quietEnd)
	}

	// Create shared data provider
	dataProvider := leaderboard.NewDataProvider(*usersPath)

	// Start health check server in background
	healthServer := api.NewHealthServer(dataProvider, *healthPort)
	go func() {
		if err := healthServer.Start(); err != nil {
			log.Printf("Health server error: %v", err)
		}
	}()

	// Create and start bot
	mudBot, err := bot.NewBot(*token, *usersPath, *guildID, *allowedChannel, sayInterval, *quietStart, *quietEnd)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	err = mudBot.Start()
	if err != nil {
		log.Fatal("Bot error:", err)
	}
}
