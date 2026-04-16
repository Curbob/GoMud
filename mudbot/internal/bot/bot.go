package bot

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/GoMudEngine/GoMud/mudbot/internal/leaderboard"
	"github.com/bwmarrin/discordgo"
)

// Bot represents the Discord bot
type Bot struct {
	session             *discordgo.Session
	cmdHandler          *CommandHandler
	dataProvider        *leaderboard.DataProvider
	guildID             string
	allowedChannelID    string
	sayInterval         time.Duration
	quietHoursStart     int
	quietHoursEnd       int
	lastAnnouncementIdx int
	randSource          *rand.Rand
	sayings             []string
}

// NewBot creates a new Discord bot instance
func NewBot(token, usersPath, guildID, allowedChannelID string, sayInterval time.Duration, quietHoursStart, quietHoursEnd int) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %v", err)
	}

	dataProvider := leaderboard.NewDataProvider(usersPath)
	cmdHandler := NewCommandHandler(dataProvider)

	bot := &Bot{
		session:             session,
		cmdHandler:          cmdHandler,
		dataProvider:        dataProvider,
		guildID:             guildID,
		allowedChannelID:    allowedChannelID,
		sayInterval:         sayInterval,
		quietHoursStart:     quietHoursStart,
		quietHoursEnd:       quietHoursEnd,
		lastAnnouncementIdx: -1,
		randSource:          rand.New(rand.NewSource(time.Now().UnixNano())),
		sayings: []string{
			"The terminal hums to life. Type /mudhelp for commands.",
			"MudBot is awake and listening for packets. Type /mudhelp for commands.",
			"Need a quick look at the realm? Type /mudhelp for commands.",
			"Status boards flicker in the dark. Type /mudhelp for commands.",
			"The dungeon wire is hot tonight. Type /mudhelp for commands.",
			"Looking for players, rankings, or server info? Type /mudhelp for commands.",
			"CackalackyCon bytes are flowing. Type /mudhelp for commands.",
			"MudBot is on watch in #gomud. Type /mudhelp for commands.",
		},
	}

	// Register event handlers
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onInteractionCreate)

	return bot, nil
}

// Start starts the bot and blocks until shutdown
func (b *Bot) Start() error {
	// Open Discord connection
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("failed to open Discord connection: %v", err)
	}
	defer b.session.Close()

	fmt.Println("MudBot is running! Press Ctrl+C to stop.")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("MudBot shutting down...")
	return nil
}

// onReady handles the bot ready event
func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Printf("MudBot is ready! Logged in as %s\n", event.User.Username)

	// Set bot status
	err := s.UpdateGameStatus(0, "CackalackyCon MUD")
	if err != nil {
		log.Printf("Error setting status: %v", err)
	}

	// Register slash commands
	err = b.registerCommands()
	if err != nil {
		log.Printf("Error registering commands: %v", err)
	}

	// Load initial data
	err = b.dataProvider.LoadUsers()
	if err != nil {
		log.Printf("Error loading initial user data: %v", err)
	}

	// Start periodic data refresh
	go b.dataRefreshLoop()

	if b.allowedChannelID != "" && b.sayInterval > 0 {
		go b.announcementLoop()
	}
}

// onInteractionCreate handles slash command interactions
func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if b.allowedChannelID != "" && i.ChannelID != b.allowedChannelID {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "MudBot only responds in the configured GoMUD channel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	commandName := i.ApplicationCommandData().Name

	switch commandName {
	case "mudhelp":
		b.cmdHandler.HandleHelpCommand(s, i)
	case "mudserver":
		b.cmdHandler.HandleServerCommand(s, i)
	case "mudstatus":
		b.cmdHandler.HandleStatusCommand(s, i)
	case "mudwho":
		b.cmdHandler.HandleWhoCommand(s, i)
	case "mudwhere":
		b.cmdHandler.HandleWhereCommand(s, i)
	case "mudleaderboard":
		b.cmdHandler.HandleLeaderboardCommand(s, i)
	default:
		fmt.Printf("Unknown command: %s\n", commandName)
	}
}

// registerCommands registers all slash commands with Discord
func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "mudhelp",
			Description: "Show available MudBot commands and help information",
		},
		{
			Name:        "mudserver",
			Description: "Get MUD server connection information",
		},
		{
			Name:        "mudstatus",
			Description: "Check MUD server status and player count",
		},
		{
			Name:        "mudwho",
			Description: "Show who is currently in the MUD",
		},
		{
			Name:        "mudwhere",
			Description: "Show who is online and where they are",
		},
		{
			Name:        "mudleaderboard",
			Description: "View player leaderboards",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "type",
					Description: "Type of leaderboard to view",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Level (Highest Level Players)",
							Value: "level",
						},
						{
							Name:  "Gold (Richest Players)",
							Value: "gold",
						},
						{
							Name:  "Kills (Most Dangerous Players)",
							Value: "kills",
						},
					},
				},
			},
		},
	}

	for _, command := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.guildID, command)
		if err != nil {
			return fmt.Errorf("failed to create command %s: %v", command.Name, err)
		}
	}

	fmt.Printf("Registered %d slash commands\n", len(commands))
	return nil
}

// dataRefreshLoop periodically refreshes user data
func (b *Bot) dataRefreshLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := b.dataProvider.LoadUsers()
			if err != nil {
				log.Printf("Error refreshing user data: %v", err)
			} else {
				fmt.Printf("User data refreshed at %s\n", time.Now().Format("15:04:05"))
			}
		}
	}
}

func (b *Bot) announcementLoop() {
	ticker := time.NewTicker(b.sayInterval)
	defer ticker.Stop()

	for range ticker.C {
		if b.inQuietHours(time.Now()) {
			continue
		}

		message := b.randomSaying()
		if message == "" {
			continue
		}

		_, err := b.session.ChannelMessageSend(b.allowedChannelID, message)
		if err != nil {
			log.Printf("Error sending scheduled announcement: %v", err)
		} else {
			log.Printf("Scheduled announcement sent to channel %s", b.allowedChannelID)
		}
	}
}

func (b *Bot) inQuietHours(now time.Time) bool {
	hour := now.Hour()
	if b.quietHoursStart == b.quietHoursEnd {
		return false
	}
	if b.quietHoursStart < b.quietHoursEnd {
		return hour >= b.quietHoursStart && hour < b.quietHoursEnd
	}
	return hour >= b.quietHoursStart || hour < b.quietHoursEnd
}

func (b *Bot) randomSaying() string {
	if len(b.sayings) == 0 {
		return ""
	}
	if len(b.sayings) == 1 {
		b.lastAnnouncementIdx = 0
		return strings.TrimSpace(b.sayings[0])
	}

	idx := b.randSource.Intn(len(b.sayings))
	for idx == b.lastAnnouncementIdx {
		idx = b.randSource.Intn(len(b.sayings))
	}
	b.lastAnnouncementIdx = idx
	return strings.TrimSpace(b.sayings[idx])
}
