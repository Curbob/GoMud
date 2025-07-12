package hooks

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/gmcp"
)

//
// Some clean up
//

func HandleLeave(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerDespawn", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		mudlog.Error("HandleLeave", "error", fmt.Sprintf(`user %d not found`, evt.UserId))
		return events.Cancel
	}

	connId := user.ConnectionId()

	// Remove any zombie tracking for the user since they've been despawned from the world.
	if users.IsZombieConnection(connId) {
		users.RemoveZombieUser(evt.UserId)
	}

	room := rooms.LoadRoom(user.Character.RoomId)

	if currentParty := parties.Get(evt.UserId); currentParty != nil {
		currentParty.Leave(evt.UserId)
	}

	for _, mobInstId := range room.GetMobs(rooms.FindCharmed) {
		if mob := mobs.GetInstance(mobInstId); mob != nil {
			if mob.Character.IsCharmed(evt.UserId) {
				mob.Character.Charmed.Expire()
			}
		}
	}

	if _, ok := room.RemovePlayer(evt.UserId); ok {
		tplTxt, _ := templates.Process("player-despawn", user.Character.Name)
		room.SendText(tplTxt)
	}

	tplTxt, _ := templates.Process("goodbye", nil, evt.UserId)
	connections.SendTo([]byte(templates.AnsiParse(tplTxt)), connId)

	if err := users.LogOutUserByConnectionId(connId); err != nil {
		mudlog.Error("Log Out Error", "connectionId", connId, "error", err)
	}
	
	// Clean up all GMCP state for this user
	gmcp.CleanupUser(evt.UserId)
	
	connections.Remove(connId)

	specialRooms := configs.GetSpecialRoomsConfig()
	testRoomId := rooms.GetOriginalRoom(user.Character.RoomId)
	for i := range specialRooms.TutorialRooms {
		roomId, _ := strconv.Atoi(specialRooms.TutorialRooms[i])
		if roomId == testRoomId {
			user.Character.RoomId = -1
			break
		}
	}

	users.SaveUser(*user)

	return events.Continue
}
