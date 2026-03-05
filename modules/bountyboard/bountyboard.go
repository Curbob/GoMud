package bountyboard

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	//go:embed files/*
	files embed.FS
)

// BountyModule manages the bounty board system
type BountyModule struct {
	plug       *plugins.Plugin
	activeBounties map[int]*PlayerBounty // keyed by UserId
	bountyLock     sync.RWMutex
}

// Bounty represents a bounty contract
type Bounty struct {
	MobId       int    `yaml:"mobid" json:"mobid"`             // The mob type to hunt
	MobName     string `yaml:"mobname" json:"mobname"`         // Display name
	Zone        string `yaml:"zone" json:"zone"`               // Zone where mob is found
	Required    int    `yaml:"required" json:"required"`       // Number to kill
	GoldReward  int    `yaml:"goldreward" json:"goldreward"`   // Gold reward
	ExpReward   int    `yaml:"expreward" json:"expreward"`     // Experience reward
	MinLevel    int    `yaml:"minlevel" json:"minlevel"`       // Minimum player level
	MaxLevel    int    `yaml:"maxlevel" json:"maxlevel"`       // Maximum player level (0 = no max)
	Description string `yaml:"description" json:"description"` // Flavor text
}

// PlayerBounty tracks a player's active bounty
type PlayerBounty struct {
	Bounty    Bounty    `json:"bounty"`
	Progress  int       `json:"progress"`  // Kills so far
	StartedAt time.Time `json:"startedat"` // When they accepted
}

// SaveData for persistence
type SaveData struct {
	ActiveBounties map[int]*PlayerBounty `json:"activebounties"`
}

// BountyConfig is the structure of the bounties.yaml file
type BountyConfig struct {
	Bounties []Bounty `yaml:"bounties"`
}

// Available bounties - loaded from datafile
var availableBounties = []Bounty{}

func init() {
	mod := &BountyModule{
		plug:           plugins.New(`bountyboard`, `1.0`),
		activeBounties: make(map[int]*PlayerBounty),
	}

	if err := mod.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	// Register commands
	mod.plug.AddUserCommand(`bounty`, mod.BountyCommand, true, false)
	mod.plug.AddUserCommand(`bounties`, mod.BountyCommand, true, false)

	// Register callbacks
	mod.plug.Callbacks.SetOnLoad(mod.load)
	mod.plug.Callbacks.SetOnSave(mod.save)

	// Listen for mob deaths
	events.RegisterListener(events.MobDeath{}, mod.onMobDeath)
}

func (mod *BountyModule) load() {
	// Load player progress
	var data SaveData
	mod.plug.ReadIntoStruct(`bountydata`, &data)
	if data.ActiveBounties != nil {
		mod.bountyLock.Lock()
		mod.activeBounties = data.ActiveBounties
		mod.bountyLock.Unlock()
	}

	// Load bounty definitions from config
	var config BountyConfig
	if err := mod.plug.ReadIntoStruct(`bounties`, &config); err == nil && len(config.Bounties) > 0 {
		availableBounties = config.Bounties
		// Sort by level
		sort.Slice(availableBounties, func(i, j int) bool {
			return availableBounties[i].MinLevel < availableBounties[j].MinLevel
		})
	}
}

func (mod *BountyModule) save() {
	mod.bountyLock.RLock()
	data := SaveData{ActiveBounties: mod.activeBounties}
	mod.bountyLock.RUnlock()
	mod.plug.WriteStruct(`bountydata`, data)
}

// onMobDeath handles mob death events to track bounty progress
func (mod *BountyModule) onMobDeath(e events.Event) events.ListenerReturn {
	death, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	// Check each player who dealt damage
	for userId, damage := range death.PlayerDamage {
		if damage <= 0 {
			continue
		}

		mod.bountyLock.Lock()
		playerBounty, exists := mod.activeBounties[userId]
		if !exists {
			mod.bountyLock.Unlock()
			continue
		}

		// Check if this mob matches the bounty
		if death.MobId == playerBounty.Bounty.MobId {
			playerBounty.Progress++

			user := users.GetByUserId(userId)
			if user == nil {
				mod.bountyLock.Unlock()
				continue
			}

			// Check if bounty is complete
			if playerBounty.Progress >= playerBounty.Bounty.Required {
				// Award rewards
				user.Character.Gold += playerBounty.Bounty.GoldReward
				user.Character.Experience += playerBounty.Bounty.ExpReward

				user.SendText(``)
				user.SendText(`<ansi fg="green-bold">═══════════════════════════════════════</ansi>`)
				user.SendText(`<ansi fg="green-bold">       🎯 BOUNTY COMPLETE! 🎯</ansi>`)
				user.SendText(`<ansi fg="green-bold">═══════════════════════════════════════</ansi>`)
				user.SendText(fmt.Sprintf(`You completed: <ansi fg="yellow">%s</ansi>`, playerBounty.Bounty.Description))
				user.SendText(fmt.Sprintf(`Reward: <ansi fg="gold">%d gold</ansi> + <ansi fg="cyan">%d experience</ansi>`, playerBounty.Bounty.GoldReward, playerBounty.Bounty.ExpReward))
				user.SendText(``)

				events.AddToQueue(events.EquipmentChange{
					UserId:     userId,
					GoldChange: playerBounty.Bounty.GoldReward,
				})

				// Remove completed bounty
				delete(mod.activeBounties, userId)
			} else {
				// Progress update
				remaining := playerBounty.Bounty.Required - playerBounty.Progress
				user.SendText(fmt.Sprintf(`<ansi fg="yellow">🎯 Bounty progress:</ansi> %d/%d %s (%d remaining)`,
					playerBounty.Progress, playerBounty.Bounty.Required, playerBounty.Bounty.MobName, remaining))
			}
		}
		mod.bountyLock.Unlock()
	}

	return events.Continue
}

// BountyCommand handles the bounty/bounties command
func (mod *BountyModule) BountyCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		return mod.showBountyBoard(user, room)
	}

	switch args[0] {
	case "list", "board":
		return mod.showBountyBoard(user, room)
	case "accept", "take":
		if len(args) < 2 {
			user.SendText(`Usage: <ansi fg="command">bounty accept [number]</ansi>`)
			return true, nil
		}
		return mod.acceptBounty(args[1], user)
	case "status", "progress":
		return mod.showProgress(user)
	case "abandon", "cancel":
		return mod.abandonBounty(user)
	default:
		// Try to accept by number
		if _, err := strconv.Atoi(args[0]); err == nil {
			return mod.acceptBounty(args[0], user)
		}
		user.SendText(`Unknown bounty command. Try <ansi fg="command">bounty</ansi> for the board.`)
	}

	return true, nil
}

func (mod *BountyModule) showBountyBoard(user *users.UserRecord, room *rooms.Room) (bool, error) {

	playerLevel := user.Character.Level

	user.SendText(``)
	user.SendText(`<ansi fg="yellow">╔═══════════════════════════════════════════════════════════════════╗</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>                    <ansi fg="white-bold">🎯 BOUNTY BOARD 🎯</ansi>                          <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">╠═══════════════════════════════════════════════════════════════════╣</ansi>`)

	// Get available bounties for player's level
	available := []Bounty{}
	for _, b := range availableBounties {
		if playerLevel >= b.MinLevel && (b.MaxLevel == 0 || playerLevel <= b.MaxLevel) {
			available = append(available, b)
		}
	}

	if len(available) == 0 {
		user.SendText(`<ansi fg="yellow">║</ansi>  No bounties available for your level.                           <ansi fg="yellow">║</ansi>`)
	} else {
		for i, b := range available {
			levelRange := fmt.Sprintf("Lvl %d", b.MinLevel)
			if b.MaxLevel > 0 {
				levelRange = fmt.Sprintf("Lvl %d-%d", b.MinLevel, b.MaxLevel)
			} else {
				levelRange = fmt.Sprintf("Lvl %d+", b.MinLevel)
			}
			
			line1 := fmt.Sprintf(`<ansi fg="yellow">║</ansi>  <ansi fg="cyan-bold">[%d]</ansi> <ansi fg="white-bold">%-15s</ansi> x%d   <ansi fg="gold">%4dg</ansi> <ansi fg="cyan">%4dxp</ansi>  %s`, 
				i+1, b.MobName, b.Required, b.GoldReward, b.ExpReward, levelRange)
			// Pad to fit box
			user.SendText(line1 + strings.Repeat(" ", 67-len(stripAnsi(line1))) + `<ansi fg="yellow">║</ansi>`)
			
			line2 := fmt.Sprintf(`<ansi fg="yellow">║</ansi>      <ansi fg="8">%s</ansi>`, b.Description)
			user.SendText(line2 + strings.Repeat(" ", 67-len(stripAnsi(line2))) + `<ansi fg="yellow">║</ansi>`)
		}
	}

	user.SendText(`<ansi fg="yellow">╠═══════════════════════════════════════════════════════════════════╣</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>  <ansi fg="command">bounty [#]</ansi> - Accept a bounty                                  <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>  <ansi fg="command">bounty status</ansi> - Check your active bounty                     <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>  <ansi fg="command">bounty abandon</ansi> - Give up your current bounty                 <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">╚═══════════════════════════════════════════════════════════════════╝</ansi>`)
	user.SendText(``)

	return true, nil
}

func (mod *BountyModule) acceptBounty(numStr string, user *users.UserRecord) (bool, error) {

	// Check if already has a bounty
	mod.bountyLock.RLock()
	_, exists := mod.activeBounties[user.UserId]
	mod.bountyLock.RUnlock()

	if exists {
		user.SendText(`You already have an active bounty! Use <ansi fg="command">bounty status</ansi> to check it.`)
		user.SendText(`Use <ansi fg="command">bounty abandon</ansi> to give it up first.`)
		return true, nil
	}

	num, err := strconv.Atoi(numStr)
	if err != nil || num < 1 {
		user.SendText(`Invalid bounty number.`)
		return true, nil
	}

	// Get available bounties for player
	playerLevel := user.Character.Level
	available := []Bounty{}
	for _, b := range availableBounties {
		if playerLevel >= b.MinLevel && (b.MaxLevel == 0 || playerLevel <= b.MaxLevel) {
			available = append(available, b)
		}
	}

	if num > len(available) {
		user.SendText(`Invalid bounty number.`)
		return true, nil
	}

	bounty := available[num-1]

	mod.bountyLock.Lock()
	mod.activeBounties[user.UserId] = &PlayerBounty{
		Bounty:    bounty,
		Progress:  0,
		StartedAt: time.Now(),
	}
	mod.bountyLock.Unlock()

	user.SendText(``)
	user.SendText(fmt.Sprintf(`<ansi fg="green">🎯 Bounty accepted!</ansi> Kill %d <ansi fg="yellow">%s</ansi>.`, bounty.Required, bounty.MobName))
	user.SendText(fmt.Sprintf(`Reward: <ansi fg="gold">%d gold</ansi> + <ansi fg="cyan">%d experience</ansi>`, bounty.GoldReward, bounty.ExpReward))
	user.SendText(``)

	return true, nil
}

func (mod *BountyModule) showProgress(user *users.UserRecord) (bool, error) {

	mod.bountyLock.RLock()
	playerBounty, exists := mod.activeBounties[user.UserId]
	mod.bountyLock.RUnlock()

	if !exists {
		user.SendText(`You don't have an active bounty. Use <ansi fg="command">bounty</ansi> to see available contracts.`)
		return true, nil
	}

	remaining := playerBounty.Bounty.Required - playerBounty.Progress
	pct := (playerBounty.Progress * 100) / playerBounty.Bounty.Required

	// Progress bar
	barLen := 20
	filled := (pct * barLen) / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLen-filled)

	user.SendText(``)
	user.SendText(`<ansi fg="yellow">═══════════════════════════════════════</ansi>`)
	user.SendText(`<ansi fg="yellow">       🎯 ACTIVE BOUNTY 🎯</ansi>`)
	user.SendText(`<ansi fg="yellow">═══════════════════════════════════════</ansi>`)
	user.SendText(fmt.Sprintf(`Target: <ansi fg="white-bold">%s</ansi> x%d`, playerBounty.Bounty.MobName, playerBounty.Bounty.Required))
	user.SendText(fmt.Sprintf(`Progress: <ansi fg="cyan">[%s]</ansi> %d/%d (%d%%)`, bar, playerBounty.Progress, playerBounty.Bounty.Required, pct))
	user.SendText(fmt.Sprintf(`Remaining: <ansi fg="red">%d</ansi>`, remaining))
	user.SendText(fmt.Sprintf(`Reward: <ansi fg="gold">%d gold</ansi> + <ansi fg="cyan">%d experience</ansi>`, playerBounty.Bounty.GoldReward, playerBounty.Bounty.ExpReward))
	user.SendText(``)

	return true, nil
}

func (mod *BountyModule) abandonBounty(user *users.UserRecord) (bool, error) {

	mod.bountyLock.Lock()
	_, exists := mod.activeBounties[user.UserId]
	if exists {
		delete(mod.activeBounties, user.UserId)
	}
	mod.bountyLock.Unlock()

	if !exists {
		user.SendText(`You don't have an active bounty to abandon.`)
		return true, nil
	}

	user.SendText(`<ansi fg="red">You abandon your bounty.</ansi> Check the board for new contracts.`)
	return true, nil
}

// Helper to strip ANSI codes for length calculation
func stripAnsi(s string) string {
	result := s
	for {
		start := strings.Index(result, "<ansi")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	for {
		start := strings.Index(result, "</ansi>")
		if start == -1 {
			break
		}
		result = result[:start] + result[start+7:]
	}
	return result
}

// Bounties are sorted by level during load()
