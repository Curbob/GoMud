# 📋 Tonight's Deployment Checklist

**Rich, here's your quick reference for deploying to Oracle Cloud tonight:**

---

## 🎯 Before You Start (5 min prep)

**Files you'll need:**
- ✅ `~/projects/GoMUD/ORACLE_CLOUD_DEPLOYMENT.md` - Complete guide
- ✅ `~/projects/GoMUD/mudbot/bin/mudbot-arm64` - ARM binary ready
- ✅ `~/projects/GoMUD/mudbot/config.example.env` - Bot config template

**Accounts you'll need:**
- [ ] Oracle Cloud account (sign up first - it's free)
- [ ] Discord Developer account (https://discord.com/developers/applications)

---

## ⚡ Quick Steps (Total: ~60 minutes)

### 1. Oracle Cloud (15 min)
- [ ] Sign up at https://cloud.oracle.com/
- [ ] Create ARM instance (2-4 CPU, 12-24GB RAM)
- [ ] Download SSH keys (SAVE THEM!)
- [ ] Configure firewall rules (ports 22, 33333, 44444)

### 2. Discord Bot (5 min)  
- [ ] Create Discord app/bot
- [ ] Copy bot token (keep secret!)
- [ ] Add bot to your Discord server
- [ ] Get your Discord server ID

### 3. Server Setup (20 min)
- [ ] SSH into Oracle instance
- [ ] Install Go + basic tools
- [ ] Upload GoMud files
- [ ] Build and configure GoMud

### 4. MudBot Deploy (10 min)
- [ ] Upload `mudbot-arm64` binary
- [ ] Configure with Discord token
- [ ] Set up systemd service

### 5. Testing (10 min)
- [ ] Test MUD connection: `telnet YOUR_IP 33333`
- [ ] Test Discord commands: `/help`, `/server`, `/status`
- [ ] Verify leaderboards work

---

## 🚨 If You Get Stuck

**Most common issues:**
1. **Can't SSH**: Check Oracle firewall rules, verify SSH key path
2. **GoMud won't start**: Check config.yaml, view logs with `sudo journalctl -u gomud -f`
3. **Bot won't connect**: Verify Discord token in `/opt/mudbot/.env`
4. **Commands don't work**: Make sure bot has permissions in Discord

**Emergency contacts:**
- Oracle docs: https://docs.oracle.com/en-us/iaas/
- Discord dev docs: https://discord.com/developers/docs/
- Or just ping me! I'll help debug.

---

## 💡 Pro Tips

- **Save your SSH key files** - you'll need them every time
- **Use `screen` or `tmux`** if your connection is flaky
- **Test locally first** - make sure GoMud builds on your Mac
- **Take notes** - write down your public IP, Discord IDs, etc.

---

**Everything is ready to go! The deployment guide has every command you need.**

**Good luck! 🚀**