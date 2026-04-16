# MudBot Deployment Guide

## Pre-Deployment Checklist

### 1. Discord Bot Setup
- [ ] Create Discord Application at https://discord.com/developers/applications
- [ ] Create bot user and copy token
- [ ] Invite bot to server with required permissions:
  - Send Messages
  - Use Slash Commands
  - Embed Links
- [ ] Get Discord Guild ID (right-click server → Copy ID with Developer Mode enabled)

### 2. Server Preparation  
- [ ] GoMud server running with user data in YAML files
- [ ] File system access to GoMud users directory
- [ ] Network access for Discord API (HTTPS outbound)

## Local Testing

1. **Build MudBot**
   ```bash
   cd mudbot
   ./scripts/build.sh
   ```

2. **Test with your data**
   ```bash
   export DISCORD_TOKEN=your_bot_token_here
   export DISCORD_GUILD_ID=your_guild_id_here  
   ./bin/mudbot -users /path/to/gomud/users
   ```

3. **Verify commands work in Discord**
   - `/server` - Should show connection info
   - `/status` - Should show current player count
   - `/leaderboard level` - Should show player rankings

## VPS Deployment

### Option 1: Direct Binary Deployment

1. **Create user and directories**
   ```bash
   sudo useradd -r -s /bin/false mudbot
   sudo mkdir -p /opt/mudbot/{bin,logs}
   sudo chown mudbot:mudbot /opt/mudbot
   ```

2. **Upload and install binary**
   ```bash
   # Build for Linux (if building on Mac/Windows)
   GOOS=linux GOARCH=amd64 go build -o bin/mudbot-linux ./cmd/mudbot
   
   # Upload to VPS
   scp bin/mudbot-linux user@vps:/opt/mudbot/bin/mudbot
   ssh user@vps "sudo chown mudbot:mudbot /opt/mudbot/bin/mudbot"
   ssh user@vps "sudo chmod +x /opt/mudbot/bin/mudbot"
   ```

3. **Install systemd service**
   ```bash
   sudo cp scripts/mudbot.service /etc/systemd/system/
   sudo systemctl edit mudbot.service  # Set your DISCORD_TOKEN
   sudo systemctl enable mudbot
   sudo systemctl start mudbot
   ```

### Option 2: Docker Deployment

1. **Create Dockerfile**
   ```dockerfile
   FROM golang:1.19-alpine AS builder
   WORKDIR /app
   COPY . .
   RUN go mod tidy && go build -o mudbot ./cmd/mudbot
   
   FROM alpine:latest
   RUN apk --no-cache add ca-certificates tzdata
   WORKDIR /root/
   COPY --from=builder /app/mudbot .
   CMD ["./mudbot"]
   ```

2. **Build and run**
   ```bash
   docker build -t mudbot .
   docker run -d \
     --name mudbot \
     -e DISCORD_TOKEN=your_token \
     -e DISCORD_GUILD_ID=your_guild \
     -v /opt/gomud/users:/data/users:ro \
     --restart unless-stopped \
     mudbot
   ```

## Security Configuration

### File System Permissions
```bash
# Read-only access to GoMud users directory
sudo chmod -R 644 /opt/gomud/_datafiles/world/default/users
sudo chown -R gomud:mudbot /opt/gomud/_datafiles/world/default/users
```

### Firewall Rules
```bash
# Only allow outbound HTTPS (for Discord API)
sudo ufw deny incoming
sudo ufw allow outgoing 443/tcp
sudo ufw allow ssh
sudo ufw enable
```

### Rate Limiting (nginx proxy)
```nginx
location /health {
    limit_req zone=api burst=5 nodelay;
    # Health check endpoint if added later
}
```

## Monitoring & Logs

### View logs
```bash
# Systemd
journalctl -u mudbot -f

# Docker  
docker logs -f mudbot
```

### Health monitoring script
```bash
#!/bin/bash
# /opt/mudbot/scripts/health_check.sh

if ! systemctl is-active --quiet mudbot; then
    echo "MudBot is down! Restarting..."
    systemctl restart mudbot
    
    # Send alert (webhook, email, etc.)
    curl -X POST "https://hooks.slack.com/..." \
         -d '{"text":"MudBot restarted on VPS"}'
fi
```

### Cron job for health checks
```bash
# Check every 5 minutes
*/5 * * * * /opt/mudbot/scripts/health_check.sh
```

## Backup & Recovery

### Config backup
```bash
# Backup bot configuration
tar -czf mudbot-config-$(date +%Y%m%d).tar.gz \
    /etc/systemd/system/mudbot.service \
    /opt/mudbot/
```

### Quick recovery
```bash
# If bot goes down, check:
1. systemctl status mudbot
2. journalctl -u mudbot -n 50
3. Check Discord token hasn't expired
4. Verify GoMud users directory access
5. Test network connectivity to Discord
```

## Performance Tuning

### Memory usage
- Expect ~10-20MB base memory usage
- Memory grows with number of users (~1KB per user)
- No database connections or heavy processing

### Disk I/O
- Bot scans users directory every 5 minutes
- With 1000 users: ~5MB of YAML files read
- Consider SSD storage for better responsiveness

### Network
- Minimal Discord API calls
- Only sends messages in response to commands
- Typical bandwidth: <1MB/day for moderate usage

## Troubleshooting

### Common Issues

1. **Bot doesn't respond to commands**
   - Check Discord token and permissions
   - Verify guild ID is correct
   - Check bot is online in Discord

2. **"Failed to load users" error**
   - Verify users directory path
   - Check file permissions
   - Ensure YAML files are valid

3. **Leaderboard shows no data**
   - Check if admin users are excluded (correct)
   - Verify user YAML files have required fields
   - Check for YAML parsing errors in logs

4. **High memory usage**
   - Check number of user files
   - Look for memory leaks in logs
   - Restart bot if necessary

### Debug Mode
```bash
# Enable verbose logging (if implemented)
export DEBUG=true
./bin/mudbot
```

### Test Commands Manually
```bash
# Test data loading without Discord
cd mudbot
go run test_data.go  # Creates temporary test
```

## Updates & Maintenance

### Updating MudBot
```bash
# Pull updates
git pull

# Rebuild
cd mudbot && ./scripts/build.sh

# Deploy new binary
sudo systemctl stop mudbot
sudo cp bin/mudbot /opt/mudbot/bin/mudbot
sudo systemctl start mudbot

# Verify
sudo systemctl status mudbot
```

### Log Rotation
```bash
# Add to /etc/logrotate.d/mudbot
/var/log/mudbot/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        systemctl reload mudbot || true
    endscript
}
```