package gmcp

import (
	"math"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	g := GMCPPartyModule{
		plug: plugins.New(`gmcp.Party`, `1.0`),
	}

	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(events.PartyUpdated{}, g.onPartyChange)
	events.RegisterListener(PartyUpdateVitals{}, g.onUpdateVitals)
	events.RegisterListener(GMCPPartyUpdate{}, g.buildAndSendGMCPPayload)

}

type GMCPPartyModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

// GMCPPartyUpdate is used to request Party module updates
type GMCPPartyUpdate struct {
	UserId     int
	Identifier string
}

func (g GMCPPartyUpdate) Type() string { return `GMCPPartyUpdate` }

// This is a uniqu event so that multiple party members moving thorugh an area all at once don't queue up a bunch for just one party
type PartyUpdateVitals struct {
	LeaderId int
}

func (g PartyUpdateVitals) Type() string     { return `PartyUpdateVitals` }
func (g PartyUpdateVitals) UniqueID() string { return `PartyVitals-` + strconv.Itoa(g.LeaderId) }

func (g *GMCPPartyModule) roomChangeHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.RoomChange)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "RoomChange", "Actual Type", e.Type())
		return events.Cancel
	}

	if evt.MobInstanceId > 0 {
		return events.Continue
	}

	party := parties.Get(evt.UserId)
	if party == nil {
		return events.Continue
	}

	events.AddToQueue(PartyUpdateVitals{
		LeaderId: party.LeaderUserId,
	})

	return events.Continue
}

func (g *GMCPPartyModule) onUpdateVitals(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(PartyUpdateVitals)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PartyUpdateVitals", "Actual Type", e.Type())
		return events.Cancel
	}

	party := parties.Get(evt.LeaderId)
	if party == nil {
		return events.Cancel
	}

	payload, moduleName := g.GetPartyNode(party, `Party.Vitals`)

	for _, userId := range party.GetMembers() {

		events.AddToQueue(GMCPOut{
			UserId:  userId,
			Module:  moduleName,
			Payload: payload,
		})

	}

	return events.Continue
}

func (g *GMCPPartyModule) onPartyChange(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PartyUpdated)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PartyUpdated", "Actual Type", e.Type())
		return events.Cancel
	}

	if len(evt.UserIds) == 0 {
		return events.Cancel
	}

	var party *parties.Party
	for _, uId := range evt.UserIds {
		if party = parties.Get(uId); party != nil {
			break
		}
	}

	// Send both Party.Info and Party.Vitals as separate messages
	infoPayload, _ := g.GetPartyNode(party, `Party.Info`)
	vitalsPayload, _ := g.GetPartyNode(party, `Party.Vitals`)

	inParty := map[int]struct{}{}
	if party != nil {
		for _, uId := range party.GetMembers() {
			inParty[uId] = struct{}{}
		}
		for _, uId := range party.GetInvited() {
			inParty[uId] = struct{}{}
		}
	}

	for _, userId := range evt.UserIds {

		if _, ok := inParty[userId]; ok {
			// Send party info (structure)
			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  `Party.Info`,
				Payload: infoPayload,
			})

			// Send party vitals (health/location)
			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  `Party.Vitals`,
				Payload: vitalsPayload,
			})

		} else {
			// Not in party - send empty payloads
			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  `Party.Info`,
				Payload: GMCPPartyModule_Payload_Info{},
			})

			events.AddToQueue(GMCPOut{
				UserId:  userId,
				Module:  `Party.Vitals`,
				Payload: map[string]GMCPPartyModule_Payload_Vitals{},
			})

		}

	}

	return events.Continue
}

func (g *GMCPPartyModule) GetPartyNode(party *parties.Party, gmcpModule string) (data any, moduleName string) {

	if party == nil {
		if gmcpModule == `Party.Info` {
			return GMCPPartyModule_Payload_Info{}, `Party.Info`
		}
		return map[string]GMCPPartyModule_Payload_Vitals{}, `Party.Vitals`
	}

	// Prepare both info and vitals data
	infoPayload := GMCPPartyModule_Payload_Info{
		Leader:  `None`,
		Members: []GMCPPartyModule_Payload_User{},
		Invited: []GMCPPartyModule_Payload_User{},
	}

	vitalsPayload := map[string]GMCPPartyModule_Payload_Vitals{}

	roomTitles := map[int]string{}

	for _, uId := range party.GetMembers() {

		if user := users.GetByUserId(uId); user != nil {

			hPct := int(math.Floor((float64(user.Character.Health) / float64(user.Character.HealthMax.Value)) * 100))
			if hPct < 0 {
				hPct = 0
			}

			roomTitle, ok := roomTitles[user.Character.RoomId]
			if !ok {
				if uRoom := rooms.LoadRoom(user.Character.RoomId); uRoom != nil {
					roomTitle = uRoom.Title
					roomTitles[user.Character.RoomId] = roomTitle
				}
			}

			vitalsPayload[user.Character.Name] = GMCPPartyModule_Payload_Vitals{
				Level:         user.Character.Level,
				HealthPercent: hPct,
				Location:      roomTitle,
			}

			// Only add to info payload if we're requesting Party.Info
			if gmcpModule == `Party.Info` {
				if user.UserId == party.LeaderUserId {
					infoPayload.Leader = user.Character.Name
				}

				status := "party"
				if user.UserId == party.LeaderUserId {
					status = "leader"
				}

				infoPayload.Members = append(infoPayload.Members,
					GMCPPartyModule_Payload_User{
						Name:     user.Character.Name,
						Status:   status,
						Position: party.GetRank(user.UserId),
					},
				)
			}

		}

	}

	for _, uId := range party.GetInvited() {

		if user := users.GetByUserId(uId); user != nil {

			// Invited users show as empty vitals
			vitalsPayload[user.Character.Name] = GMCPPartyModule_Payload_Vitals{
				Level:         0,
				HealthPercent: 0,
				Location:      ``,
			}

			// Only add to info payload if we're requesting Party.Info
			if gmcpModule == `Party.Info` {
				infoPayload.Invited = append(infoPayload.Invited,
					GMCPPartyModule_Payload_User{
						Name:     user.Character.Name,
						Status:   `invited`,
						Position: ``,
					},
				)
			}

		}

	}

	switch gmcpModule {
	case `Party.Info`:
		return infoPayload, `Party.Info`
	case `Party.Vitals`:
		return vitalsPayload, `Party.Vitals`
	default:
		mudlog.Error(`gmcp.Party`, `error`, `Bad module requested`, `module`, gmcpModule)
		return nil, ""
	}

}

// GMCPPartyModule_Payload_Info contains static party structure
type GMCPPartyModule_Payload_Info struct {
	Leader  string                         `json:"leader"`
	Members []GMCPPartyModule_Payload_User `json:"members"`
	Invited []GMCPPartyModule_Payload_User `json:"invited"`
}

// DEPRECATED: Old combined payload structure
type GMCPPartyModule_Payload struct {
	Leader  string
	Members []GMCPPartyModule_Payload_User
	Invited []GMCPPartyModule_Payload_User
	Vitals  map[string]GMCPPartyModule_Payload_Vitals
}

type GMCPPartyModule_Payload_User struct {
	Name     string `json:"name"`
	Status   string `json:"status"`   // party/leader/invited
	Position string `json:"position"` // frontrank/middle/backrank
}

type GMCPPartyModule_Payload_Vitals struct {
	Level         int    `json:"level"`    // level of user
	HealthPercent int    `json:"health"`   // 1 = 1%, 23 = 23% etc.
	Location      string `json:"location"` // Title of room they are in
}

func (g *GMCPPartyModule) buildAndSendGMCPPayload(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(GMCPPartyUpdate)
	if !typeOk {
		return events.Continue
	}

	if evt.UserId < 1 {
		return events.Continue
	}

	// Make sure they have GMCP enabled
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Continue
	}

	// If requesting "Party", send all sub-nodes
	if evt.Identifier == `Party` {
		g.sendAllPartyNodes(evt.UserId)
		return events.Continue
	}

	// Otherwise, send the specific node requested
	party := parties.Get(evt.UserId)
	// Note: party can be nil, GetPartyNode handles that case

	payload, moduleName := g.GetPartyNode(party, evt.Identifier)
	if payload != nil && moduleName != "" {
		events.AddToQueue(GMCPOut{
			UserId:  evt.UserId,
			Module:  moduleName,
			Payload: payload,
		})
	}

	return events.Continue
}

// sendAllPartyNodes sends all Party nodes as individual GMCP messages
func (g *GMCPPartyModule) sendAllPartyNodes(userId int) {
	party := parties.Get(userId)

	// Always send party structures, even if empty (when not in a party)
	// Send Party.Info
	if infoPayload, moduleName := g.GetPartyNode(party, `Party.Info`); infoPayload != nil {
		events.AddToQueue(GMCPOut{
			UserId:  userId,
			Module:  moduleName,
			Payload: infoPayload,
		})
	}

	// Send Party.Vitals
	if vitalsPayload, moduleName := g.GetPartyNode(party, `Party.Vitals`); vitalsPayload != nil {
		events.AddToQueue(GMCPOut{
			UserId:  userId,
			Module:  moduleName,
			Payload: vitalsPayload,
		})
	}
}
