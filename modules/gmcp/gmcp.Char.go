package gmcp

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	//
	// We can use all functions only, but this demonstrates
	// how to use a struct
	//
	g := GMCPCharModule{
		plug: plugins.New(`gmcp.Char`, `1.0`),
	}

	events.RegisterListener(events.EquipmentChange{}, g.equipmentChangeHandler)
	events.RegisterListener(events.ItemOwnership{}, g.ownershipChangeHandler)

	events.RegisterListener(events.PlayerSpawn{}, g.playerSpawnHandler)
	events.RegisterListener(events.CharacterVitalsChanged{}, g.vitalsChangedHandler)
	events.RegisterListener(events.CharacterAlignmentChanged{}, g.alignmentChangedHandler)
	events.RegisterListener(events.LevelUp{}, g.levelUpHandler)
	events.RegisterListener(events.CharacterTrained{}, g.charTrainedHandler)
	events.RegisterListener(GMCPCharUpdate{}, g.buildAndSendGMCPPayload)
	events.RegisterListener(events.GainExperience{}, g.xpGainHandler)
	events.RegisterListener(events.CharacterStatsChanged{}, g.statsChangeHandler)
	events.RegisterListener(events.CharacterChanged{}, g.charChangeHandler)
	events.RegisterListener(events.BuffsTriggered{}, g.buffTriggeredHandler)

	events.RegisterListener(events.Quest{}, g.questProgressHandler)

	// Clean up tracking maps on player disconnect
	events.RegisterListener(events.PlayerDespawn{}, g.playerDespawnHandler)

}

type GMCPCharModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

// Tell the system a wish to send specific GMCP Update data
type GMCPCharUpdate struct {
	UserId     int
	Identifier string
}

func (g GMCPCharUpdate) Type() string { return `GMCPCharUpdate` }

func (g *GMCPCharModule) questProgressHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.Quest)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Quests`,
	})

	return events.Continue
}

func (g *GMCPCharModule) buffTriggeredHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.BuffsTriggered)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Affects`,
	})

	return events.Continue
}

func (g *GMCPCharModule) charChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterChanged)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

// Track last vitals update time to prevent spam
var (
	vitalsUpdateMu   sync.Mutex
	lastVitalsUpdate = make(map[int]time.Time)
)

func (g *GMCPCharModule) vitalsChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterVitalsChanged)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Deduplicate rapid vitals updates (max 1 per 100ms)
	vitalsUpdateMu.Lock()
	lastUpdate, exists := lastVitalsUpdate[evt.UserId]
	now := time.Now()
	if exists && now.Sub(lastUpdate) < 100*time.Millisecond {
		vitalsUpdateMu.Unlock()
		return events.Continue // Skip this update
	}
	lastVitalsUpdate[evt.UserId] = now
	vitalsUpdateMu.Unlock()

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Vitals`,
	})

	return events.Continue
}

func (g *GMCPCharModule) alignmentChangedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterAlignmentChanged)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Info`,
	})

	return events.Continue
}

func (g *GMCPCharModule) xpGainHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.GainExperience)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Worth`,
	})

	return events.Continue
}

func (g *GMCPCharModule) ownershipChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.ItemOwnership)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	// Send both Items and Summary updates when inventory changes
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Inventory.Backpack.Items`,
	})
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char.Inventory.Backpack.Summary`,
	})

	return events.Continue
}

func (g *GMCPCharModule) statsChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterStatsChanged)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Stats`})
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Vitals`})
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Inventory.Backpack.Summary`})

	return events.Continue
}

func (g *GMCPCharModule) equipmentChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.EquipmentChange)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	if len(evt.ItemsRemoved) > 0 || len(evt.ItemsWorn) > 0 {
		// Queue individual events for each module
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Inventory.Worn`})
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Inventory.Backpack.Items`})
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Inventory.Backpack.Summary`})
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Stats`})
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Vitals`})
	}

	// If gold or bank changed
	if evt.BankChange != 0 || evt.GoldChange != 0 {
		events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Worth`})
	}

	return events.Continue
}

func (g *GMCPCharModule) charTrainedHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.CharacterTrained)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Stats`})
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Worth`})
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Vitals`})
	events.AddToQueue(GMCPCharUpdate{UserId: evt.UserId, Identifier: `Char.Inventory.Backpack.Summary`})

	return events.Continue
}
func (g *GMCPCharModule) levelUpHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.LevelUp)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Changing equipment might affect stats, inventory, maxhp/maxmp etc
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

func (g *GMCPCharModule) playerSpawnHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Continue // Return false to stop halt the event chain for this event
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Send full update
	events.AddToQueue(GMCPCharUpdate{
		UserId:     evt.UserId,
		Identifier: `Char`,
	})

	return events.Continue
}

func (g *GMCPCharModule) playerDespawnHandler(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId == 0 {
		return events.Continue
	}

	// Clean up vitals tracking
	vitalsUpdateMu.Lock()
	delete(lastVitalsUpdate, evt.UserId)
	vitalsUpdateMu.Unlock()

	return events.Continue
}

func (g *GMCPCharModule) buildAndSendGMCPPayload(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPCharUpdate)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPCharUpdate", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Make sure they have this gmcp module enabled.
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		mudlog.Warn("GMCP", "buildAndSendGMCPPayload", "GMCP not enabled for update",
			"userId", evt.UserId,
			"username", user.Username,
			"connId", user.ConnectionId(),
			"identifier", evt.Identifier)
		return events.Cancel
	}

	if len(evt.Identifier) >= 4 {
		// Normalize the identifier (handle case variations)
		identifierParts := strings.Split(strings.ToLower(evt.Identifier), `.`)
		for i := 0; i < len(identifierParts); i++ {
			identifierParts[i] = strings.Title(identifierParts[i])
		}

		requestedId := strings.Join(identifierParts, `.`)

		payload, moduleName := g.GetCharNode(user, requestedId)

		// Skip if nil payload (handled elsewhere, like sendAllCharNodes)
		if payload != nil && moduleName != "" {
			events.AddToQueue(GMCPOut{
				UserId:  evt.UserId,
				Module:  moduleName,
				Payload: payload,
			})
		}
	}

	return events.Continue
}

// sendAllCharNodes sends all Char nodes as individual GMCP messages
func (g *GMCPCharModule) sendAllCharNodes(user *users.UserRecord) {
	// Send each node individually
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Info`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Status`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Stats`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Vitals`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Worth`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Affects`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Inventory.Worn`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Inventory.Backpack.Items`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Inventory.Backpack.Summary`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Quests`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Pets`})
	events.AddToQueue(GMCPCharUpdate{UserId: user.UserId, Identifier: `Char.Enemies`})
}

func (g *GMCPCharModule) GetCharNode(user *users.UserRecord, gmcpModule string) (data any, moduleName string) {

	all := gmcpModule == `Char`

	// If requesting all, we'll handle it differently by queuing individual messages
	if all {
		g.sendAllCharNodes(user)
		return nil, ""
	}

	payload := GMCPCharModule_Payload{}

	if g.wantsGMCPPayload(`Char.Info`, gmcpModule) {
		payload.Info = &GMCPCharModule_Payload_Info{
			Account:   user.Username,
			Name:      util.StripANSI(user.Character.Name),
			Class:     skills.GetProfession(user.Character.GetAllSkillRanks()),
			Race:      user.Character.Race(),
			Alignment: user.Character.AlignmentName(),
			Level:     user.Character.Level,
		}

		return payload.Info, `Char.Info`
	}

	if g.wantsGMCPPayload(`Char.Status`, gmcpModule) {
		// Return character status
		status := map[string]interface{}{
			"state": "standing",
		}
		return status, `Char.Status`
	}

	if g.wantsGMCPPayload(`Char.Pets`, gmcpModule) {

		payload.Pets = []GMCPCharModule_Payload_Pet{}

		if user.Character.Pet.Exists() {

			p := GMCPCharModule_Payload_Pet{
				Name:   user.Character.Pet.Name,
				Type:   user.Character.Pet.Type,
				Hunger: user.Character.Pet.Food.String(),
			}

			payload.Pets = append(payload.Pets, p)
		}

		return payload.Pets, `Char.Pets`
	}

	if g.wantsGMCPPayload(`Char.Enemies`, gmcpModule) {

		payload.Enemies = []GMCPCharModule_Enemy{}

		aggroMobInstanceId := 0
		if user.Character.Aggro != nil {
			if user.Character.Aggro.MobInstanceId > 0 {
				aggroMobInstanceId = user.Character.Aggro.MobInstanceId
			}
		}

		if roomInfo := rooms.LoadRoom(user.Character.RoomId); roomInfo != nil {

			for _, mobInstanceId := range roomInfo.GetMobs(rooms.FindFighting) {
				mob := mobs.GetInstance(mobInstanceId)
				if mob == nil {
					continue
				}

				e := GMCPCharModule_Enemy{
					Id:        mob.ShorthandId(),
					Name:      util.StripANSI(mob.Character.Name),
					Level:     mob.Character.Level,
					Health:    mob.Character.Health,
					HealthMax: mob.Character.HealthMax.Value,
					Engaged:   mob.InstanceId == aggroMobInstanceId,
				}

				payload.Enemies = append(payload.Enemies, e)
			}

		}

		return payload.Enemies, `Char.Enemies`
	}

	// Allow specifically updating the Backpack Summary
	if `Char.Inventory.Backpack.Summary` == gmcpModule {

		payload.Inventory = &GMCPCharModule_Payload_Inventory{
			Backpack: &GMCPCharModule_Payload_Inventory_Backpack{
				Summary: GMCPCharModule_Payload_Inventory_Backpack_Summary{
					Count: len(user.Character.Items),
					Max:   user.Character.CarryCapacity(),
				},
			},
		}

		return payload.Inventory.Backpack.Summary, `Char.Inventory.Backpack.Summary`
	}

	// Allow specifically updating the Backpack Items
	if `Char.Inventory.Backpack.Items` == gmcpModule {
		items := []GMCPCharModule_Payload_Inventory_Item{}
		for _, itm := range user.Character.Items {
			items = append(items, newInventory_Item(itm))
		}
		return items, `Char.Inventory.Backpack.Items`
	}

	// Handle individual inventory nodes separately
	if g.wantsGMCPPayload(`Char.Inventory.Worn`, gmcpModule) {
		payload.Inventory = &GMCPCharModule_Payload_Inventory{
			Worn: &GMCPCharModule_Payload_Inventory_Worn{
				Weapon:  newInventory_Item(user.Character.Equipment.Weapon),
				Offhand: newInventory_Item(user.Character.Equipment.Offhand),
				Head:    newInventory_Item(user.Character.Equipment.Head),
				Neck:    newInventory_Item(user.Character.Equipment.Neck),
				Body:    newInventory_Item(user.Character.Equipment.Body),
				Belt:    newInventory_Item(user.Character.Equipment.Belt),
				Gloves:  newInventory_Item(user.Character.Equipment.Gloves),
				Ring:    newInventory_Item(user.Character.Equipment.Ring),
				Legs:    newInventory_Item(user.Character.Equipment.Legs),
				Feet:    newInventory_Item(user.Character.Equipment.Feet),
			},
		}
		return payload.Inventory.Worn, `Char.Inventory.Worn`
	}

	if g.wantsGMCPPayload(`Char.Stats`, gmcpModule) {

		payload.Stats = &GMCPCharModule_Payload_Stats{
			Strength:   user.Character.Stats.Strength.ValueAdj,
			Speed:      user.Character.Stats.Speed.ValueAdj,
			Smarts:     user.Character.Stats.Smarts.ValueAdj,
			Vitality:   user.Character.Stats.Vitality.ValueAdj,
			Mysticism:  user.Character.Stats.Mysticism.ValueAdj,
			Perception: user.Character.Stats.Perception.ValueAdj,
		}

		return payload.Stats, `Char.Stats`
	}

	if g.wantsGMCPPayload(`Char.Vitals`, gmcpModule) {

		payload.Vitals = &GMCPCharModule_Payload_Vitals{
			Health:         user.Character.Health,
			HealthMax:      user.Character.HealthMax.Value,
			SpellPoints:    user.Character.Mana,
			SpellPointsMax: user.Character.ManaMax.Value,
		}

		return payload.Vitals, `Char.Vitals`
	}

	if g.wantsGMCPPayload(`Char.Worth`, gmcpModule) {

		payload.Worth = &GMCPCharModule_Payload_Worth{
			GoldCarried:    user.Character.Gold,
			GoldBank:       user.Character.Bank,
			SkillPoints:    user.Character.StatPoints,
			TrainingPoints: user.Character.TrainingPoints,
			ToNextLevel:    user.Character.XPTL(user.Character.Level),
			Experience:     user.Character.Experience,
		}

		return payload.Worth, `Char.Worth`
	}

	if g.wantsGMCPPayload(`Char.Affects`, gmcpModule) {

		c := configs.GetTimingConfig()

		payload.Affects = make(map[string]GMCPCharModule_Payload_Affect)

		nameIncrement := 0
		for _, buff := range user.Character.GetBuffs() {

			buffSpec := buffs.GetBuffSpec(buff.BuffId)
			if buffSpec == nil {
				continue
			}

			timeLeft, timeMax := -1, -1

			if !buff.PermaBuff {
				roundsLeft, totalRounds := buffs.GetDurations(buff, buffSpec)
				timeMax = c.RoundsToSeconds(totalRounds)
				timeLeft = c.RoundsToSeconds(roundsLeft)
				if timeLeft < 0 {
					timeLeft = 0
				}
			}

			name, desc := buffSpec.VisibleNameDesc()

			buffSource := buff.Source
			if buffSource == `` {
				buffSource = `unknown`
			}
			aff := GMCPCharModule_Payload_Affect{
				Name:         name,
				Description:  desc,
				DurationMax:  timeMax,
				DurationLeft: timeLeft,
				Type:         buffSource,
			}

			aff.Mods = make(map[string]int)
			for name, value := range buffSpec.StatMods {
				aff.Mods[name] = value
			}

			if _, ok := payload.Affects[name]; ok {
				nameIncrement++
				name += `#` + strconv.Itoa(nameIncrement)
			}

			payload.Affects[name] = aff
		}

		return payload.Affects, `Char.Affects`
	}

	if g.wantsGMCPPayload(`Char.Quests`, gmcpModule) {

		payload.Quests = []GMCPCharModule_Payload_Quest{}

		for questId, questStep := range user.Character.GetQuestProgress() {

			questToken := quests.PartsToToken(questId, questStep)

			questInfo := quests.GetQuest(questToken)
			if questInfo == nil {
				continue
			}

			// Secret quests are not sent
			if questInfo.Secret {
				continue
			}

			completedSteps := 0
			totalSteps := len(questInfo.Steps)

			questPayload := GMCPCharModule_Payload_Quest{
				Name:        questInfo.Name,
				Completion:  0,
				Description: questInfo.Description,
			}

			for _, step := range questInfo.Steps {
				completedSteps++
				if step.Id == questStep {
					questPayload.Description = step.Description
					break
				}
			}

			questPayload.Completion = int(math.Floor(float64(completedSteps)/float64(totalSteps)) * 100)

			// Add to the returned output
			payload.Quests = append(payload.Quests, questPayload)
		}

		return payload.Quests, `Char.Quests`
	}

	// If we reached this point, we have a problem.
	mudlog.Error(`gmcp.Char`, `error`, `Bad module requested`, `module`, gmcpModule)
	return nil, ""
}

// wantsGMCPPayload(`Char.Info`, `Char`)
func (g *GMCPCharModule) wantsGMCPPayload(packageToConsider string, packageRequested string) bool {

	if packageToConsider == packageRequested {
		return true
	}

	if len(packageToConsider) < len(packageRequested) {
		return false
	}

	if packageToConsider[0:len(packageRequested)] == packageRequested {
		return true
	}

	return false
}

type GMCPCharModule_Payload struct {
	Info      *GMCPCharModule_Payload_Info             `json:"info"`
	Affects   map[string]GMCPCharModule_Payload_Affect `json:"affects"`
	Enemies   []GMCPCharModule_Enemy                   `json:"enemies"`
	Inventory *GMCPCharModule_Payload_Inventory        `json:"inventory"`
	Stats     *GMCPCharModule_Payload_Stats            `json:"stats"`
	Vitals    *GMCPCharModule_Payload_Vitals           `json:"vitals"`
	Worth     *GMCPCharModule_Payload_Worth            `json:"worth"`
	Quests    []GMCPCharModule_Payload_Quest           `json:"quests"`
	Pets      []GMCPCharModule_Payload_Pet             `json:"pets"`
}

// /////////////////
// Char.Info
// /////////////////
type GMCPCharModule_Payload_Info struct {
	Account   string `json:"account"`
	Name      string `json:"name"`
	Class     string `json:"class"`
	Race      string `json:"race"`
	Alignment string `json:"alignment"`
	Level     int    `json:"level"`
}

// /////////////////
// Char.Enemies
// /////////////////
type GMCPCharModule_Enemy struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Health    int    `json:"health"`
	HealthMax int    `json:"health_max"`
	Engaged   bool   `json:"engaged"`
}

// /////////////////
// Char.Inventory
// /////////////////
type GMCPCharModule_Payload_Inventory struct {
	Backpack *GMCPCharModule_Payload_Inventory_Backpack `json:"Backpack"`
	Worn     *GMCPCharModule_Payload_Inventory_Worn     `json:"Worn"`
}

type GMCPCharModule_Payload_Inventory_Backpack struct {
	Items   []GMCPCharModule_Payload_Inventory_Item           `json:"Items"`
	Summary GMCPCharModule_Payload_Inventory_Backpack_Summary `json:"Summary"`
}

type GMCPCharModule_Payload_Inventory_Backpack_Summary struct {
	Count int `json:"count"`
	Max   int `json:"max"`
}

type GMCPCharModule_Payload_Inventory_Worn struct {
	Weapon  GMCPCharModule_Payload_Inventory_Item `json:"weapon"`
	Offhand GMCPCharModule_Payload_Inventory_Item `json:"offhand"`
	Head    GMCPCharModule_Payload_Inventory_Item `json:"head"`
	Neck    GMCPCharModule_Payload_Inventory_Item `json:"neck"`
	Body    GMCPCharModule_Payload_Inventory_Item `json:"body"`
	Belt    GMCPCharModule_Payload_Inventory_Item `json:"belt"`
	Gloves  GMCPCharModule_Payload_Inventory_Item `json:"gloves"`
	Ring    GMCPCharModule_Payload_Inventory_Item `json:"ring"`
	Legs    GMCPCharModule_Payload_Inventory_Item `json:"legs"`
	Feet    GMCPCharModule_Payload_Inventory_Item `json:"feet"`
}

type GMCPCharModule_Payload_Inventory_Item struct {
	Id      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	SubType string   `json:"sub_type"`
	Uses    int      `json:"uses"`
	Details []string `json:"details"`
	Command string   `json:"command"`
}

func newInventory_Item(itm items.Item) GMCPCharModule_Payload_Inventory_Item {
	// Handle empty equipment slots
	if itm.ItemId == 0 {
		return GMCPCharModule_Payload_Inventory_Item{
			Id:      "",
			Name:    "",
			Type:    "",
			SubType: "",
			Uses:    0,
			Details: []string{},
			Command: "",
		}
	}

	// Handle disabled slots
	if itm.IsDisabled() {
		return GMCPCharModule_Payload_Inventory_Item{
			Id:      "",
			Name:    "disabled",
			Type:    "disabled",
			SubType: "",
			Uses:    0,
			Details: []string{"disabled"},
			Command: "",
		}
	}

	itmSpec := itm.GetSpec()
	d := GMCPCharModule_Payload_Inventory_Item{
		Id:      itm.ShorthandId(),
		Name:    util.StripANSI(itm.Name()),
		Type:    string(itmSpec.Type),
		SubType: string(itmSpec.Subtype),
		Uses:    itm.Uses,
		Details: []string{},
		Command: getItemCommand(itmSpec),
	}

	if !itm.Uncursed && itmSpec.Cursed {
		d.Details = append(d.Details, `cursed`)
	}

	if itmSpec.QuestToken != `` {
		d.Details = append(d.Details, `quest`)
	}

	return d
}

// getItemCommand returns the primary command for using an item based on its type and subtype
func getItemCommand(spec items.ItemSpec) string {
	// Check subtype first as it's more specific
	switch spec.Subtype {
	case items.Drinkable:
		return "drink"
	case items.Edible:
		return "eat"
	case items.Usable:
		return "use"
	case items.Throwable:
		return "throw"
	case items.Wearable:
		return "wear"
	}

	// Check type for specific commands
	switch spec.Type {
	case items.Weapon:
		return "wield"
	case items.Readable:
		return "read"
	case items.Grenade:
		return "drop" // drop to explode
	case items.Lockpicks:
		return "picklock"
	}

	// For worn equipment (already equipped items)
	if spec.Type == items.Offhand || spec.Type == items.Head ||
		spec.Type == items.Neck || spec.Type == items.Body ||
		spec.Type == items.Belt || spec.Type == items.Gloves ||
		spec.Type == items.Ring || spec.Type == items.Legs ||
		spec.Type == items.Feet {
		return "remove"
	}

	// Default - no specific command
	return ""
}

// /////////////////
// Char.Stats
// /////////////////
type GMCPCharModule_Payload_Stats struct {
	Strength   int `json:"strength"`
	Speed      int `json:"speed"`
	Smarts     int `json:"smarts"`
	Vitality   int `json:"vitality"`
	Mysticism  int `json:"mysticism"`
	Perception int `json:"perception"`
}

// /////////////////
// Char.Vitals
// /////////////////
type GMCPCharModule_Payload_Vitals struct {
	Health         int `json:"health"`
	HealthMax      int `json:"health_max"`
	SpellPoints    int `json:"spell_points"`
	SpellPointsMax int `json:"spell_points_max"`
}

// /////////////////
// Char.Worth
// /////////////////
type GMCPCharModule_Payload_Worth struct {
	GoldCarried    int `json:"gold_carried"`
	GoldBank       int `json:"gold_bank"`
	SkillPoints    int `json:"skill_points"`
	TrainingPoints int `json:"training_points"`
	ToNextLevel    int `json:"to_next_level"`
	Experience     int `json:"experience"`
}

// /////////////////
// Char.Affects
// /////////////////
type GMCPCharModule_Payload_Affect struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	DurationMax  int            `json:"duration_max"`
	DurationLeft int            `json:"duration_current"`
	Type         string         `json:"type"`
	Mods         map[string]int `json:"affects"`
}

// /////////////////
// Char.Quests
// /////////////////
type GMCPCharModule_Payload_Quest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Completion  int    `json:"completion"`
}

// /////////////////
// Char.Pets
// /////////////////
type GMCPCharModule_Payload_Pet struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Hunger string `json:"hunger"`
}
