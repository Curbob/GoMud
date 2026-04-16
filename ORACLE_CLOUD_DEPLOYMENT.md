# 🚀 Oracle Cloud ARM Deployment Guide for GoMud + MudBot

**Complete step-by-step guide to deploy your CackalackyCon MUD on Oracle Cloud's free ARM instances.**

---

## 📋 Phase 1: Oracle Cloud Setup (15 minutes)

### 1.1 Create Oracle Cloud Account
1. Go to https://cloud.oracle.com/
2. Click "Start for free" 
3. Sign up with your email
4. **Important**: Choose your home region wisely (can't change later)
   - **US East (Ashburn)** - Good for East Coast
   - **US West (Phoenix)** - Good for West Coast
5. Complete phone verification
6. Add credit card (won't be charged, required for verification)

### 1.2 Create ARM Compute Instance
1. **Navigate**: Compute → Instances → Create Instance
2. **Name**: `cackalackycon-mud`
3. **Image**: Ubuntu 22.04 (ARM64)
4. **Shape**: 
   - Click "Change Shape"
   - Select "Ampere" (ARM-based)
   - **OCPU**: 2 (or all 4 if you want)
   - **Memory**: 12GB (or all 24GB)
5. **Networking**:
   - Create new VCN: `mud-network`
   - Create new subnet: `mud-subnet`  
   - ✅ Assign public IPv4 address
6. **SSH Keys**:
   - Generate new key pair
   - **Save both files**: `ssh-key-*.key` (private) and `ssh-key-*.pub` (public)
   - Store them safely!
7. Click **Create**

Wait 2-3 minutes for instance to start. Note the **public IP address**.

### 1.3 Configure Firewall Rules
1. **Navigate**: Networking → Virtual Cloud Networks → mud-network
2. Click **Security Lists** → Default Security List
3. **Add Ingress Rules**:

   ```
   Rule 1: SSH
   Source: 0.0.0.0/0
   Port: 22
   
   Rule 2: GoMud Telnet
   Source: 0.0.0.0/0  
   Port: 33333
   
   Rule 3: GoMud SSL
   Source: 0.0.0.0/0
   Port: 44444
   
   Rule 4: HTTP (optional - for web admin)
   Source: 0.0.0.0/0
   Port: 80
   
   Rule 5: HTTPS (optional - for web admin) 
   Source: 0.0.0.0/0
   Port: 443
   ```

---

## 📋 Phase 2: Server Setup (20 minutes)

### 2.1 Connect to Your Server
```bash
# Make your SSH key usable
chmod 600 ~/Downloads/ssh-key-*.key

# Connect (replace YOUR_PUBLIC_IP)
ssh -i ~/Downloads/ssh-key-*.key ubuntu@YOUR_PUBLIC_IP
```

### 2.2 Initial Server Configuration
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install essential tools
sudo apt install -y curl wget unzip git build-essential

# Install Go (required for GoMud)
cd /tmp
wget https://go.dev/dl/go1.21.6.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify Go installation
go version  # Should show: go version go1.21.6 linux/arm64

# Create directory structure
sudo mkdir -p /opt/{gomud,mudbot}
sudo chown -R ubuntu:ubuntu /opt/{gomud,mudbot}
```

### 2.3 Configure Firewall (Ubuntu level)
```bash
# Enable Ubuntu firewall
sudo ufw enable

# Allow SSH
sudo ufw allow 22/tcp

# Allow GoMud ports
sudo ufw allow 33333/tcp
sudo ufw allow 44444/tcp

# Optional: Web ports
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Check status
sudo ufw status
```

---

## 📋 Phase 3: Deploy GoMud (15 minutes)

### 3.1 Upload GoMud Files
**On your local machine:**
```bash
# Create deployment package
cd ~/projects/GoMUD
tar -czf gomud-deploy.tar.gz \
  --exclude='*.git*' \
  --exclude='mudbot' \
  --exclude='logs/*' \
  .

# Upload to server (replace YOUR_PUBLIC_IP)
scp -i ~/Downloads/ssh-key-*.key \
  gomud-deploy.tar.gz \
  ubuntu@YOUR_PUBLIC_IP:/opt/gomud/
```

### 3.2 Install GoMud on Server
**Back on the server (SSH session):**
```bash
cd /opt/gomud

# Extract files
tar -xzf gomud-deploy.tar.gz
rm gomud-deploy.tar.gz

# Build GoMud for ARM
go mod tidy
go build -o gomud .

# Make executable
chmod +x gomud

# Test configuration
./gomud --help
```

### 3.3 Configure GoMud for Production
```bash
# Edit configuration for production
nano _datafiles/config.yaml
```

**Key changes to make:**
```yaml
# Set your domain/IP
Host: "YOUR_PUBLIC_IP"  # or mud.cackalackycon.org

# Increase connection limits  
MaxTelnetConnections: 200

# Enable logging
LogLevel: "info"

# Production mode
Debug: false
```

### 3.4 Create GoMud Service
```bash
sudo tee /etc/systemd/system/gomud.service > /dev/null <<EOF
[Unit]
Description=CackalackyCon GoMUD Server
After=network.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/opt/gomud
ExecStart=/opt/gomud/gomud
Restart=always
RestartSec=5

# Security
NoNewPrivileges=yes
ProtectSystem=strict
ReadWritePaths=/opt/gomud

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=gomud

[Install]
WantedBy=multi-user.target
EOF

# Enable and start GoMud
sudo systemctl daemon-reload
sudo systemctl enable gomud
sudo systemctl start gomud

# Check status
sudo systemctl status gomud
```

---

## 📋 Phase 4: Deploy MudBot (10 minutes)

### 4.1 Build MudBot for ARM
**On your local machine:**
```bash
cd ~/projects/GoMUD/mudbot

# Cross-compile for ARM64
GOOS=linux GOARCH=arm64 go build -o mudbot-arm64 ./cmd/mudbot

# Upload to server
scp -i ~/Downloads/ssh-key-*.key \
  mudbot-arm64 \
  ubuntu@YOUR_PUBLIC_IP:/opt/mudbot/mudbot

scp -i ~/Downloads/ssh-key-*.key \
  config.example.env \
  ubuntu@YOUR_PUBLIC_IP:/opt/mudbot/
```

### 4.2 Configure MudBot
**On the server:**
```bash
cd /opt/mudbot

# Make executable
chmod +x mudbot

# Create configuration
cp config.example.env .env
nano .env
```

**Edit `.env` with your Discord bot details:**
```bash
# Get these from Discord Developer Portal
DISCORD_TOKEN=your_bot_token_here
DISCORD_GUILD_ID=your_discord_server_id_here

# GoMud data path
GOMUD_USERS_PATH=/opt/gomud/_datafiles/world/default/users
```

### 4.3 Create MudBot Service
```bash
sudo tee /etc/systemd/system/mudbot.service > /dev/null <<EOF
[Unit]
Description=MudBot Discord Integration
After=network.target gomud.service

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/opt/mudbot
ExecStart=/opt/mudbot/mudbot
Restart=always
RestartSec=10
EnvironmentFile=/opt/mudbot/.env

# Security
NoNewPrivileges=yes
ProtectSystem=strict
ReadOnlyPaths=/opt/gomud/_datafiles/world/default/users

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mudbot

[Install]
WantedBy=multi-user.target
EOF

# Enable (but don't start until you have Discord token)
sudo systemctl daemon-reload
sudo systemctl enable mudbot
```

---

## 📋 Phase 5: Discord Bot Setup (5 minutes)

### 5.1 Create Discord Application
1. Go to https://discord.com/developers/applications
2. Click "New Application"
3. Name: "CackalackyCon MudBot"
4. Go to **Bot** section
5. Click "Add Bot"
6. **Copy the Bot Token** (keep this secret!)
7. Under "Privileged Gateway Intents":
   - ✅ Message Content Intent (if you want to read messages)

### 5.2 Add Bot to Your Server
1. Go to **OAuth2** → **URL Generator**
2. **Scopes**: ✅ `bot` ✅ `applications.commands`
3. **Bot Permissions**:
   - ✅ Send Messages
   - ✅ Use Slash Commands  
   - ✅ Embed Links
4. Copy the generated URL
5. Visit URL in browser → Add to your Discord server

### 5.3 Get Discord Server ID
1. In Discord, enable Developer Mode (User Settings → Advanced)
2. Right-click your server name → "Copy ID"
3. This is your Guild ID

### 5.4 Start MudBot
**On your server:**
```bash
# Update configuration with real values
nano /opt/mudbot/.env

# Start MudBot
sudo systemctl start mudbot

# Check it's working
sudo systemctl status mudbot

# View logs
sudo journalctl -u mudbot -f
```

---

## 📋 Phase 6: Testing & Verification (5 minutes)

### 6.1 Test GoMud Connection
```bash
# From your local machine
telnet YOUR_PUBLIC_IP 33333

# Or test SSL
openssl s_client -connect YOUR_PUBLIC_IP:44444
```

### 6.2 Test Discord Commands
In your Discord server, try:
- `/help` - Should show command list
- `/server` - Should show connection info
- `/status` - Should show player count
- `/leaderboard level` - Should show rankings

### 6.3 Health Monitoring
```bash
# Test health endpoint
curl http://YOUR_PUBLIC_IP:8080/health

# Should return JSON status
```

---

## 📋 Phase 7: DNS & Domain (Optional - 10 minutes)

### 7.1 If You Have a Domain
1. **Add A Records** in your DNS provider:
   ```
   mud.cackalackycon.org → YOUR_PUBLIC_IP
   ```

2. **Update GoMud config:**
   ```bash
   nano /opt/gomud/_datafiles/config.yaml
   # Change Host: "mud.cackalackycon.org"
   
   sudo systemctl restart gomud
   ```

---

## 🎛️ Management Commands (Reference)

### View Logs
```bash
# GoMud logs
sudo journalctl -u gomud -f

# MudBot logs  
sudo journalctl -u mudbot -f

# System logs
sudo journalctl -f
```

### Restart Services
```bash
# Restart GoMud
sudo systemctl restart gomud

# Restart MudBot
sudo systemctl restart mudbot

# Restart both
sudo systemctl restart gomud mudbot
```

### Update Services
```bash
# Update GoMud
cd /opt/gomud
git pull  # if using git
go build -o gomud .
sudo systemctl restart gomud

# Update MudBot - upload new binary from local machine
sudo systemctl restart mudbot
```

### Monitor Resource Usage
```bash
# Check memory/CPU
htop

# Check disk space
df -h

# Check network connections
netstat -tuln
```

---

## 🚨 Troubleshooting

### GoMud Won't Start
```bash
# Check logs
sudo journalctl -u gomud -n 50

# Test manually
cd /opt/gomud
./gomud

# Check ports
sudo netstat -tuln | grep -E ":(33333|44444)"
```

### MudBot Won't Connect to Discord
```bash
# Check configuration
cat /opt/mudbot/.env

# Test manually
cd /opt/mudbot  
./mudbot

# Common issues:
# - Wrong Discord token
# - Bot not added to server
# - Network connectivity
```

### Can't Connect from Outside
```bash
# Check Oracle Cloud Security Lists
# Check Ubuntu firewall
sudo ufw status

# Check if services are listening
sudo ss -tuln
```

### High Memory Usage
```bash
# Check what's using memory
ps aux --sort=-%mem | head -10

# Restart services if needed
sudo systemctl restart gomud mudbot
```

---

## 📊 Expected Resource Usage

**Normal Operation:**
- **GoMud**: 200-500MB RAM, ~5% CPU
- **MudBot**: 20-50MB RAM, ~1% CPU  
- **System**: 200-400MB RAM
- **Total**: ~1GB RAM used of 12-24GB available

**With 50 players:**
- **GoMud**: 400-800MB RAM, ~10-20% CPU
- Network: ~1-5 Mbps

**You have PLENTY of headroom!** 🚀

---

## ✅ Success Checklist

After deployment, verify:
- [ ] GoMud server starts and listens on ports 33333/44444
- [ ] Can connect via telnet/MUD client
- [ ] MudBot connects to Discord successfully  
- [ ] Discord slash commands work (`/help`, `/server`, `/status`)
- [ ] Leaderboards show real player data
- [ ] Health endpoint responds at `:8080/health`
- [ ] Services automatically restart on reboot
- [ ] Logs are being generated properly

---

**🎉 That's it! Your CackalackyCon MUD is now live on Oracle Cloud!**

**Total cost: $0/month (Oracle's Always Free tier)**  
**Capacity: 200+ concurrent players**  
**Features: Full MUD + Discord integration**

Have a great time at work, and ping me if you hit any snags tonight! 🛠️