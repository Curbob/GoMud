package fishing

import (
	"embed"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	//go:embed files/*
	files embed.FS
)

// Map fish YAML IDs to real item IDs
var fishItemIds = map[string]int{
	// Common fish
	"pool_minnow": 30100,
	"mudfish":     30101,
	"frost_trout": 30102,
	"silver_carp": 30103,
	// Buff fish
	"glowfin":   30110,
	"swiftscale": 30111,
	"ironbelly": 30112,
	"healfin":   30113,
	// Junk
	"old_boot":         30120,
	"seaweed":          30121,
	"rusty_hook":       30122,
	"waterlogged_book": 30123,
}

// FishingModule manages the fishing system
type FishingModule struct {
	plug          *plugins.Plugin
	activeFishers map[int]*FishingSession // UserId -> session
	fisherLock    sync.RWMutex
	fishData      FishConfig
	spotData      SpotConfig
}

// FishingSession tracks an active fishing attempt
type FishingSession struct {
	StartedAt time.Time
	CatchTime time.Time
	RoomId    int
	Zone      string
}

// FishConfig holds fish definitions (for names/rarity config)
type FishConfig struct {
	Fish map[string]Fish `yaml:"fish"`
}

// Fish defines a catchable fish
type Fish struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Value       int    `yaml:"value"`
	Type        string `yaml:"type"` // common, buff, junk, treasure
	MinGold     int    `yaml:"mingold"`
	MaxGold     int    `yaml:"maxgold"`
}

// SpotConfig holds fishing spot definitions
type SpotConfig struct {
	Default       SpotCatches            `yaml:"default"`
	Spots         map[string]FishingSpot `yaml:"spots"`
	RarityWeights RarityWeights          `yaml:"rarity_weights"`
	Timing        TimingConfig           `yaml:"timing"`
}

type FishingSpot struct {
	Zones      []string    `yaml:"zones"`
	Catches    SpotCatches `yaml:"catches"`
	Junk       []string    `yaml:"junk"`
	JunkChance int         `yaml:"junkchance"`
}

type SpotCatches struct {
	Common     []string `yaml:"common"`
	Uncommon   []string `yaml:"uncommon"`
	Rare       []string `yaml:"rare"`
	Legendary  []string `yaml:"legendary"`
	Junk       []string `yaml:"junk"`
	JunkChance int      `yaml:"junkchance"`
}

type RarityWeights struct {
	Common    int `yaml:"common"`
	Uncommon  int `yaml:"uncommon"`
	Rare      int `yaml:"rare"`
	Legendary int `yaml:"legendary"`
}

type TimingConfig struct {
	MinWait       int `yaml:"min_wait"`
	MaxWait       int `yaml:"max_wait"`
	NothingChance int `yaml:"nothing_chance"`
}

var mod *FishingModule

func init() {
	mod = &FishingModule{
		plug:          plugins.New(`fishing`, `1.0`),
		activeFishers: make(map[int]*FishingSession),
	}

	if err := mod.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	// Register commands - only fishing commands, eat/sell use built-in systems
	mod.plug.AddUserCommand(`fish`, mod.FishCommand, true, false)
	mod.plug.AddUserCommand(`cast`, mod.FishCommand, true, false)
	mod.plug.AddUserCommand(`reel`, mod.ReelCommand, true, false)

	// Register callbacks
	mod.plug.Callbacks.SetOnLoad(mod.load)

	// Listen for round events to process fishing
	events.RegisterListener(events.NewRound{}, mod.onNewRound)
}

func (mod *FishingModule) load() {
	// Load fish definitions
	mod.plug.ReadIntoStruct(`fish`, &mod.fishData)
	mod.plug.ReadIntoStruct(`fishspots`, &mod.spotData)

	// Set defaults if not loaded
	if mod.spotData.Timing.MinWait == 0 {
		mod.spotData.Timing.MinWait = 5
		mod.spotData.Timing.MaxWait = 15
		mod.spotData.Timing.NothingChance = 10
	}
	if mod.spotData.RarityWeights.Common == 0 {
		mod.spotData.RarityWeights = RarityWeights{
			Common: 60, Uncommon: 25, Rare: 12, Legendary: 3,
		}
	}
}

// onNewRound checks for fish catches
func (mod *FishingModule) onNewRound(e events.Event) events.ListenerReturn {
	now := time.Now()

	mod.fisherLock.Lock()
	defer mod.fisherLock.Unlock()

	for userId, session := range mod.activeFishers {
		if now.After(session.CatchTime) {
			user := users.GetByUserId(userId)
			if user == nil {
				delete(mod.activeFishers, userId)
				continue
			}

			// Check if still in same room
			room := rooms.LoadRoom(session.RoomId)
			if room == nil || user.Character.RoomId != session.RoomId {
				user.SendText(`<ansi fg="red">Your line snaps as you move away!</ansi>`)
				delete(mod.activeFishers, userId)
				continue
			}

			// Attempt catch
			mod.processCatch(user, room, session.Zone)
			delete(mod.activeFishers, userId)
		}
	}

	return events.Continue
}

// FishCommand starts fishing
func (mod *FishingModule) FishCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !room.IsFishable {
		user.SendText(`You can't fish here. Find a lake, river, or pool.`)
		return true, nil
	}

	mod.fisherLock.Lock()
	if _, exists := mod.activeFishers[user.UserId]; exists {
		mod.fisherLock.Unlock()
		user.SendText(`You're already fishing! Wait for a bite or <ansi fg="command">reel</ansi> in your line.`)
		return true, nil
	}

	// Calculate catch time
	waitTime := mod.spotData.Timing.MinWait +
		util.RollDice(1, mod.spotData.Timing.MaxWait-mod.spotData.Timing.MinWait+1) - 1

	catchTime := time.Now().Add(time.Duration(waitTime) * time.Second)

	mod.activeFishers[user.UserId] = &FishingSession{
		StartedAt: time.Now(),
		CatchTime: catchTime,
		RoomId:    room.RoomId,
		Zone:      room.Zone,
	}
	mod.fisherLock.Unlock()

	user.SendText(``)
	user.SendText(`<ansi fg="cyan">You cast your line into the water...</ansi>`)
	user.SendText(`<ansi fg="8">Wait for a bite, or type <ansi fg="command">reel</ansi> to stop.</ansi>`)

	room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> casts a fishing line into the water.`, user.Character.Name), user.UserId)

	return true, nil
}

// ReelCommand stops fishing early
func (mod *FishingModule) ReelCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	mod.fisherLock.Lock()
	_, exists := mod.activeFishers[user.UserId]
	if exists {
		delete(mod.activeFishers, user.UserId)
	}
	mod.fisherLock.Unlock()

	if !exists {
		user.SendText(`You're not fishing.`)
		return true, nil
	}

	user.SendText(`You reel in your line. Nothing caught.`)
	return true, nil
}

// processCatch determines what the player caught
func (mod *FishingModule) processCatch(user *users.UserRecord, room *rooms.Room, zone string) {

	// Get the spot config for this zone
	spot := mod.getSpotForZone(zone)

	// Check for nothing
	if util.RollDice(1, 100) <= mod.spotData.Timing.NothingChance {
		user.SendText(`<ansi fg="8">The fish got away! Your line comes up empty.</ansi>`)
		return
	}

	// Check for junk
	if util.RollDice(1, 100) <= spot.JunkChance {
		mod.catchJunk(user, room, spot)
		return
	}

	// Roll for rarity
	mod.catchFish(user, room, spot)
}

func (mod *FishingModule) getSpotForZone(zone string) SpotCatches {
	for _, spot := range mod.spotData.Spots {
		for _, z := range spot.Zones {
			if strings.EqualFold(z, zone) {
				return SpotCatches{
					Common:     spot.Catches.Common,
					Uncommon:   spot.Catches.Uncommon,
					Rare:       spot.Catches.Rare,
					Legendary:  spot.Catches.Legendary,
					Junk:       spot.Junk,
					JunkChance: spot.JunkChance,
				}
			}
		}
	}
	return mod.spotData.Default
}

func (mod *FishingModule) catchJunk(user *users.UserRecord, room *rooms.Room, spot SpotCatches) {
	if len(spot.Junk) == 0 {
		user.SendText(`<ansi fg="8">You feel a tug but reel in nothing but water.</ansi>`)
		return
	}

	junkId := spot.Junk[util.RollDice(1, len(spot.Junk))-1]
	fish, exists := mod.fishData.Fish[junkId]
	if !exists {
		user.SendText(`<ansi fg="8">You caught some debris and threw it back.</ansi>`)
		return
	}

	// Get the item ID for this junk
	itemId, hasItem := fishItemIds[junkId]
	if !hasItem {
		user.SendText(`<ansi fg="8">You caught some debris and threw it back.</ansi>`)
		return
	}

	user.SendText(``)
	user.SendText(`<ansi fg="yellow">*splash*</ansi>`)
	user.SendText(fmt.Sprintf(`You caught... <ansi fg="8">%s</ansi>`, fish.Name))
	if fish.Description != "" {
		user.SendText(fmt.Sprintf(`<ansi fg="8">%s</ansi>`, fish.Description))
	}

	// Give the item to the player
	item := items.New(itemId)
	if user.Character.StoreItem(item) {
		user.SendText(`<ansi fg="8">Added to your inventory.</ansi>`)
		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   item,
			Gained: true,
		})
	} else {
		user.SendText(`<ansi fg="red">Your inventory is full! The junk slips back into the water.</ansi>`)
	}
}

func (mod *FishingModule) catchFish(user *users.UserRecord, room *rooms.Room, spot SpotCatches) {
	// Roll for rarity
	roll := util.RollDice(1, 100)
	var fishList []string
	var rarity string

	weights := mod.spotData.RarityWeights
	if roll <= weights.Legendary {
		fishList = spot.Legendary
		rarity = "legendary"
	} else if roll <= weights.Legendary+weights.Rare {
		fishList = spot.Rare
		rarity = "rare"
	} else if roll <= weights.Legendary+weights.Rare+weights.Uncommon {
		fishList = spot.Uncommon
		rarity = "uncommon"
	} else {
		fishList = spot.Common
		rarity = "common"
	}

	// Fallback to common if rarity list is empty
	if len(fishList) == 0 {
		fishList = spot.Common
		rarity = "common"
	}
	if len(fishList) == 0 {
		user.SendText(`<ansi fg="8">The waters seem empty here.</ansi>`)
		return
	}

	fishId := fishList[util.RollDice(1, len(fishList))-1]
	fish, exists := mod.fishData.Fish[fishId]
	if !exists {
		user.SendText(`<ansi fg="8">Something slipped off the hook!</ansi>`)
		return
	}

	// Get the item ID for this fish
	itemId, hasItem := fishItemIds[fishId]
	if !hasItem {
		user.SendText(`<ansi fg="8">Something slipped off the hook!</ansi>`)
		return
	}

	// Display catch based on rarity
	user.SendText(``)

	switch rarity {
	case "legendary":
		user.SendText(`<ansi fg="yellow-bold">✨ LEGENDARY CATCH! ✨</ansi>`)
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> catches something <ansi fg="yellow-bold">legendary</ansi>!`, user.Character.Name), user.UserId)
	case "rare":
		user.SendText(`<ansi fg="green-bold">🎣 RARE CATCH!</ansi>`)
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> catches something <ansi fg="green">rare</ansi>!`, user.Character.Name), user.UserId)
	default:
		user.SendText(`<ansi fg="cyan">*splash*</ansi>`)
	}

	user.SendText(fmt.Sprintf(`You caught a <ansi fg="white-bold">%s</ansi>!`, fish.Name))

	// Handle treasure specially - gives gold directly
	if fish.Type == "treasure" && fish.MinGold > 0 {
		goldAmount := fish.MinGold + util.RollDice(1, fish.MaxGold-fish.MinGold+1) - 1
		user.SendText(fmt.Sprintf(`Inside you find <ansi fg="gold">%d gold</ansi>!`, goldAmount))
		user.Character.Gold += goldAmount
		events.AddToQueue(events.EquipmentChange{UserId: user.UserId, GoldChange: goldAmount})
		return
	}

	// Give the fish item to the player
	item := items.New(itemId)
	if user.Character.StoreItem(item) {
		itemSpec := item.GetSpec()
		user.SendText(fmt.Sprintf(`Worth <ansi fg="gold">%d gold</ansi> at a merchant.`, itemSpec.Value))
		user.SendText(`<ansi fg="8">Added to your inventory. Use <ansi fg="command">eat %s</ansi> to consume or sell to a merchant.</ansi>`)
		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   item,
			Gained: true,
		})
	} else {
		user.SendText(`<ansi fg="red">Your inventory is full! The fish slips back into the water.</ansi>`)
	}
}
