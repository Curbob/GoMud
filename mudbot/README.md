# MudBot - Discord Bot for GoMud

MudBot provides Discord integration for GoMud servers, offering server status, leaderboards, and connection information through slash commands.

## Features

- **Server Status** - Real-time player count and server information
- **Leaderboards** - Player rankings by level, gold, kills, and more
- **Connection Info** - Easy access to server connection details
- **Secure** - Read-only access with rate limiting and input validation

## Commands

- `/server` - Get server connection information
- `/status` - Check server status and player count  
- `/leaderboard <type>` - View player rankings
  - `level` - Highest level characters
  - `gold` - Richest players (including bank)
  - `kills` - Most dangerous players

## Quick Start

1. **Create Discord Bot**
   - Go to https://discord.com/developers/applications
   - Create new application and bot
   - Copy bot token

2. **Build MudBot**
   ```bash
   cd mudbot
   ./scripts/build.sh
   ```

3. **Configure**
   ```bash
   cp config.example.env .env
   # Edit .env with your bot token and guild ID
   ```

4. **Run**
   ```bash
   export DISCORD_TOKEN=your_bot_token_here
   export DISCORD_GUILD_ID=your_guild_id_here
   ./bin/mudbot
   ```

## Configuration

### Environment Variables

- `DISCORD_TOKEN` - Discord bot token (required)
- `DISCORD_GUILD_ID` - Discord server ID for faster command registration (optional)
- `GOMUD_USERS_PATH` - Path to GoMud users directory (auto-detected if not set)

### Command Line Options

- `-token` - Discord bot token
- `-users` - Path to GoMud users directory  
- `-guild` - Discord guild ID

## Security

MudBot is designed with security in mind:

- **Read-only** - Never modifies game data
- **Rate limited** - Commands are rate-limited per user
- **Input validation** - All inputs are sanitized
- **Minimal permissions** - Bot only needs basic message permissions
- **No personal data** - Only shows aggregated, public game statistics

## Architecture

```
Discord Bot ↔ File System Reader ↔ GoMud YAML Files
```

- Bot reads user data directly from GoMud's YAML files
- No database connections or network access to GoMud server
- Data refreshed every 5 minutes automatically
- Completely isolated from game server

## Development

### File Structure

```
mudbot/
├── cmd/mudbot/          # Main application entry point
├── internal/
│   ├── bot/            # Discord bot logic
│   └── leaderboard/    # Data processing and leaderboards
├── scripts/            # Build and deployment scripts
├── config.example.env  # Example configuration
└── README.md
```

### Adding New Commands

1. Add command definition to `internal/bot/bot.go` in `registerCommands()`
2. Add handler to `internal/bot/commands.go`
3. Update command router in `onInteractionCreate()`

### Adding New Leaderboards

1. Update `LeaderboardType` constants in `internal/leaderboard/types.go`
2. Add case in `GetLeaderboard()` method in `internal/leaderboard/data.go`
3. Add choice to leaderboard command options in `registerCommands()`

## Deployment

For production deployment:

1. Build for target platform
2. Set up systemd service or Docker container
3. Configure firewall (bot only needs outbound HTTPS to Discord)
4. Set up log rotation
5. Monitor bot health

Example systemd service file included in `scripts/mudbot.service`.

## License

Same as GoMud - see parent project for license details.