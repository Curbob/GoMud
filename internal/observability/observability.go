package observability

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const maxRecentEvents = 200

type RecentEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail,omitempty"`
}

type OnlinePlayerSnapshot struct {
	UserID        int    `json:"userId"`
	Username      string `json:"username"`
	CharacterName string `json:"characterName"`
	Level         int    `json:"level"`
	Role          string `json:"role"`
	OnlineTime    string `json:"onlineTime"`
	AFK           bool   `json:"afk"`
	Zone          string `json:"zone"`
	RoomID        int    `json:"roomId"`
	RoomName      string `json:"roomName"`
}

type ringBuffer struct {
	items []RecentEvent
	start int
	count int
}

func newRingBuffer(size int) ringBuffer {
	return ringBuffer{items: make([]RecentEvent, size)}
}

func (r *ringBuffer) add(evt RecentEvent) {
	if len(r.items) == 0 {
		return
	}

	idx := (r.start + r.count) % len(r.items)
	if r.count == len(r.items) {
		r.start = (r.start + 1) % len(r.items)
	} else {
		r.count++
	}
	r.items[idx] = evt
}

func (r *ringBuffer) snapshot() []RecentEvent {
	result := make([]RecentEvent, 0, r.count)
	for i := 0; i < r.count; i++ {
		idx := (r.start + i) % len(r.items)
		result = append(result, r.items[idx])
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

var recentEventsStore = struct {
	sync.RWMutex
	buffer ringBuffer
}{buffer: newRingBuffer(maxRecentEvents)}

func AddRecentEvent(eventType, title, detail string) {
	recentEventsStore.Lock()
	defer recentEventsStore.Unlock()

	recentEventsStore.buffer.add(RecentEvent{
		Timestamp: time.Now(),
		Type:      eventType,
		Title:     title,
		Detail:    detail,
	})
}

func RecentEvents() []RecentEvent {
	recentEventsStore.RLock()
	defer recentEventsStore.RUnlock()
	return recentEventsStore.buffer.snapshot()
}

func OnlinePlayersSnapshot() []OnlinePlayerSnapshot {
	userIDs := users.GetOnlineUserIds()
	result := make([]OnlinePlayerSnapshot, 0, len(userIDs))
	for _, uid := range userIDs {
		u := users.GetByUserId(uid)
		if u == nil || u.Character == nil {
			continue
		}

		info := u.GetOnlineInfo()
		roomName := ""
		if room := rooms.LoadRoom(u.Character.RoomId); room != nil {
			roomName = room.Title
		}

		result = append(result, OnlinePlayerSnapshot{
			UserID:        u.UserId,
			Username:      u.Username,
			CharacterName: info.CharacterName,
			Level:         info.Level,
			Role:          info.Role,
			OnlineTime:    info.OnlineTimeStr,
			AFK:           info.IsAFK,
			Zone:          u.Character.Zone,
			RoomID:        u.Character.RoomId,
			RoomName:      roomName,
		})
	}
	return result
}

func RegisterListeners() {
	events.RegisterListener(events.LevelUp{}, onLevelUp, events.Last)
	events.RegisterListener(events.PlayerDeath{}, onPlayerDeath, events.Last)
	events.RegisterListener(events.MobDeath{}, onMobDeath, events.Last)
	events.RegisterListener(events.Quest{}, onQuest, events.Last)
	events.RegisterListener(events.ItemOwnership{}, onItemOwnership, events.Last)
	events.RegisterListener(events.Input{}, onAdminInput, events.Last)
}

func onLevelUp(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.LevelUp)
	if !ok {
		mudlog.Error("Observability", "expected", "LevelUp", "actual", e.Type())
		return events.Continue
	}

	AddRecentEvent("level_up", fmt.Sprintf("%s reached level %d", evt.CharacterName, evt.NewLevel), fmt.Sprintf("Gained %d level(s)", evt.LevelsGained))
	return events.Continue
}

func onPlayerDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDeath)
	if !ok {
		mudlog.Error("Observability", "expected", "PlayerDeath", "actual", e.Type())
		return events.Continue
	}

	detail := ""
	if len(evt.KilledByUsers) > 0 {
		names := make([]string, 0, len(evt.KilledByUsers))
		for _, uid := range evt.KilledByUsers {
			if u := users.GetByUserId(uid); u != nil && u.Character != nil {
				names = append(names, u.Character.Name)
			}
		}
		if len(names) > 0 {
			detail = fmt.Sprintf("Killed by %s", joinNames(names))
		}
	}
	if evt.Permanent {
		AddRecentEvent("player_death", fmt.Sprintf("%s permanently died", evt.CharacterName), detail)
	} else {
		AddRecentEvent("player_death", fmt.Sprintf("%s died", evt.CharacterName), detail)
	}
	return events.Continue
}

func onMobDeath(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		mudlog.Error("Observability", "expected", "MobDeath", "actual", e.Type())
		return events.Continue
	}

	killerNames := make([]string, 0, len(evt.PlayerDamage))
	for uid := range evt.PlayerDamage {
		if u := users.GetByUserId(uid); u != nil && u.Character != nil {
			killerNames = append(killerNames, u.Character.Name)
		}
	}
	detail := ""
	if len(killerNames) > 0 {
		detail = fmt.Sprintf("Players involved: %s", joinNames(killerNames))
	}
	AddRecentEvent("mob_death", fmt.Sprintf("%s was killed", evt.CharacterName), detail)
	return events.Continue
}

func onQuest(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.Quest)
	if !ok {
		mudlog.Error("Observability", "expected", "Quest", "actual", e.Type())
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil || user.Character == nil || evt.QuestToken == "" {
		return events.Continue
	}

	remove := evt.QuestToken[0] == '-'
	if remove {
		return events.Continue
	}

	questInfo := quests.GetQuest(evt.QuestToken)
	if questInfo == nil || questInfo.Secret {
		return events.Continue
	}

	_, stepName := quests.TokenToParts(evt.QuestToken)
	title := fmt.Sprintf("%s updated quest %s", user.Character.Name, questInfo.Name)
	detail := fmt.Sprintf("Step: %s", stepName)
	if stepName == "start" {
		title = fmt.Sprintf("%s started quest %s", user.Character.Name, questInfo.Name)
		detail = ""
	} else if stepName == "end" {
		title = fmt.Sprintf("%s completed quest %s", user.Character.Name, questInfo.Name)
		detail = ""
	}

	AddRecentEvent("quest", title, detail)
	return events.Continue
}

func onItemOwnership(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.ItemOwnership)
	if !ok {
		mudlog.Error("Observability", "expected", "ItemOwnership", "actual", e.Type())
		return events.Continue
	}

	if !evt.Gained || evt.UserId <= 0 {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil || user.Character == nil {
		return events.Continue
	}

	if !isNotableItem(evt.Item) {
		return events.Continue
	}

	AddRecentEvent("item", fmt.Sprintf("%s picked up %s", user.Character.Name, evt.Item.DisplayName()), fmt.Sprintf("Item #%d", evt.Item.ItemId))
	return events.Continue
}

func onAdminInput(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.Input)
	if !ok {
		mudlog.Error("Observability", "expected", "Input", "actual", e.Type())
		return events.Continue
	}

	if evt.UserId <= 0 || evt.InputText == "" {
		return events.Continue
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil || user.Character == nil || user.Role == users.RoleUser {
		return events.Continue
	}

	cmd := evt.InputText
	if len(cmd) > 80 {
		cmd = cmd[:80] + "…"
	}
	AddRecentEvent("admin", fmt.Sprintf("%s ran an admin command", user.Character.Name), cmd)
	return events.Continue
}

func isNotableItem(item items.Item) bool {
	spec := item.GetSpec()
	if spec.QuestToken != "" {
		return true
	}
	if spec.Value >= 1000 {
		return true
	}
	if spec.Cursed || len(spec.BuffIds) > 0 || len(spec.WornBuffIds) > 0 {
		return true
	}
	return false
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		result += ", " + names[i]
	}
	return result
}
