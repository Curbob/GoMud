package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/mudbot/internal/leaderboard"
	"github.com/bwmarrin/discordgo"
)

// CommandHandler handles Discord slash commands
type CommandHandler struct {
	dataProvider *leaderboard.DataProvider
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(dataProvider *leaderboard.DataProvider) *CommandHandler {
	return &CommandHandler{
		dataProvider: dataProvider,
	}
}

// HandleServerCommand responds to /server command
func (ch *CommandHandler) HandleServerCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	connInfo := ch.dataProvider.GetConnectionInfo()
	
	var portsStr []string
	for _, port := range connInfo.Ports {
		portsStr = append(portsStr, fmt.Sprintf("%d", port))
	}
	
	embed := &discordgo.MessageEmbed{
		Title:       "🌐 CackalackyCon MUD Server",
		Description: fmt.Sprintf("**Host:** %s\n**Ports:** %s", connInfo.Host, strings.Join(portsStr, ", ")),
		Color:       0x00ff00, // Green
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "How to Connect",
				Value:  strings.Join(connInfo.Instructions, "\n"),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "MudBot • CackalackyCon 2026",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	
	if err != nil {
		fmt.Printf("Error responding to server command: %v\n", err)
	}
}

// HandleStatusCommand responds to /status command
func (ch *CommandHandler) HandleStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	status, err := ch.dataProvider.GetServerStatus()
	if err != nil {
		ch.respondWithError(s, i, "Failed to get server status", err)
		return
	}

	statusIcon := "🟢" // Green circle for online
	if status.Status != "online" {
		statusIcon = "🔴" // Red circle for offline
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s CackalackyCon MUD Status", statusIcon),
		Description: fmt.Sprintf("**Status:** %s", strings.Title(status.Status)),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Players",
				Value:  fmt.Sprintf("%d / %d online", status.PlayersOnline, status.MaxPlayers),
				Inline: true,
			},
			{
				Name:   "Uptime",
				Value:  status.Uptime,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "MudBot • Updates every 5 minutes",
		},
		Timestamp: status.LastUpdated.Format(time.RFC3339),
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	
	if err != nil {
		fmt.Printf("Error responding to status command: %v\n", err)
	}
}

// HandleHelpCommand responds to /help command
func (ch *CommandHandler) HandleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 MudBot Commands",
		Description: "Discord bot for CackalackyCon MUD Server",
		Color:       0x5865F2, // Discord blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "/mudserver",
				Value:  "Get server connection information and setup instructions",
				Inline: false,
			},
			{
				Name:   "/mudstatus",
				Value:  "Check current server status and player count",
				Inline: false,
			},
			{
				Name:   "/mudwho",
				Value:  "Show who is currently in the MUD",
				Inline: false,
			},
			{
				Name:   "/mudwhere",
				Value:  "Show who is online and where they are",
				Inline: false,
			},
			{
				Name:   "/mudleaderboard <type>",
				Value:  "View player rankings:\n• `level` - Highest level players\n• `gold` - Richest players\n• `kills` - Most dangerous players",
				Inline: false,
			},
			{
				Name:   "/mudhelp",
				Value:  "Show this help message",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "MudBot • CackalackyCon 2026 • Secure & Read-Only",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	
	if err != nil {
		fmt.Printf("Error responding to help command: %v\n", err)
	}
}

// HandleWhoCommand responds to /who command
func (ch *CommandHandler) HandleWhoCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	status, err := ch.dataProvider.GetServerStatus()
	if err != nil {
		ch.respondWithError(s, i, "Failed to get online players", err)
		return
	}

	var body strings.Builder
	if len(status.OnlinePlayers) == 0 {
		body.WriteString("*Nobody is in the MUD right now.*")
	} else {
		for _, player := range status.OnlinePlayers {
			body.WriteString(fmt.Sprintf("• **%s**\n", player.CharacterName))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "👥 Who's in the MUD",
		Description: fmt.Sprintf("**%d player(s) online**", status.PlayersOnline),
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{{
			Name:   "Players",
			Value:  body.String(),
			Inline: false,
		}},
		Footer: &discordgo.MessageEmbedFooter{Text: "MudBot • Read-only live view"},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	if err != nil {
		fmt.Printf("Error responding to who command: %v\n", err)
	}
}

// HandleWhereCommand responds to /where command
func (ch *CommandHandler) HandleWhereCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	status, err := ch.dataProvider.GetServerStatus()
	if err != nil {
		ch.respondWithError(s, i, "Failed to get player locations", err)
		return
	}

	var body strings.Builder
	if len(status.OnlinePlayers) == 0 {
		body.WriteString("*Nobody is in the MUD right now.*")
	} else {
		for _, player := range status.OnlinePlayers {
			body.WriteString(fmt.Sprintf("• **%s** — %s\n", player.CharacterName, player.Location))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📍 Where players are",
		Description: fmt.Sprintf("**%d player(s) online**", status.PlayersOnline),
		Color:       0x00c853,
		Fields: []*discordgo.MessageEmbedField{{
			Name:   "Locations",
			Value:  body.String(),
			Inline: false,
		}},
		Footer: &discordgo.MessageEmbedFooter{Text: "MudBot • Event visibility mode"},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
	if err != nil {
		fmt.Printf("Error responding to where command: %v\n", err)
	}
}

// HandleLeaderboardCommand responds to /leaderboard command
func (ch *CommandHandler) HandleLeaderboardCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get the leaderboard type from command options
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		ch.respondWithError(s, i, "Missing leaderboard type", fmt.Errorf("please specify a leaderboard type"))
		return
	}

	lbTypeStr := options[0].StringValue()
	lbType := leaderboard.LeaderboardType(lbTypeStr)
	
	// Validate leaderboard type
	validTypes := map[leaderboard.LeaderboardType]bool{
		leaderboard.LeaderboardLevel: true,
		leaderboard.LeaderboardGold:  true,
		leaderboard.LeaderboardKills: true,
		// TODO: Enable when implemented
		// leaderboard.LeaderboardFishing:  true,
		// leaderboard.LeaderboardGambling: true,
	}
	
	if !validTypes[lbType] {
		ch.respondWithError(s, i, "Invalid leaderboard type", 
			fmt.Errorf("valid types: level, gold, kills"))
		return
	}

	lb, err := ch.dataProvider.GetLeaderboard(lbType, 10) // Top 10
	if err != nil {
		ch.respondWithError(s, i, "Failed to get leaderboard", err)
		return
	}

	// Build leaderboard display
	var leaderboardText strings.Builder
	if len(lb.Entries) == 0 {
		leaderboardText.WriteString("*No data available*")
	} else {
		for _, entry := range lb.Entries {
			leaderboardText.WriteString(fmt.Sprintf("%d. **%s** - %s\n", 
				entry.Rank, entry.CharacterName, entry.FormattedValue))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       lb.Title,
		Description: lb.Description,
		Color:       0xffd700, // Gold
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Rankings",
				Value:  leaderboardText.String(),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("MudBot • %d total players • Updated %s", 
				lb.TotalPlayers, lb.LastUpdated.Format("15:04:05")),
		},
		Timestamp: lb.LastUpdated.Format(time.RFC3339),
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	
	if err != nil {
		fmt.Printf("Error responding to leaderboard command: %v\n", err)
	}
}

// respondWithError sends an error response to Discord
func (ch *CommandHandler) respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string, err error) {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Error",
		Description: message,
		Color:       0xff0000, // Red
		Footer: &discordgo.MessageEmbedFooter{
			Text: "MudBot",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	respErr := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral, // Only visible to the user
		},
	})
	
	if respErr != nil {
		fmt.Printf("Error sending error response: %v\n", respErr)
	}
	
	fmt.Printf("Command error: %s - %v\n", message, err)
}