package mobcommands

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
)

func Go(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// If has a buff that prevents combat, skip the player
	if mob.Character.HasBuffFlag(buffs.NoMovement) {
		return true, nil
	}

	// Special behavior allowed for mobs to travel to specific rooms, even if disconnected.
	if forceRoomId, err := strconv.Atoi(rest); err == nil {

		foundRoomExit := false
		for exitName, exitInfo := range room.Exits {
			if exitInfo.RoomId == forceRoomId {
				rest = exitName
				foundRoomExit = true
			}
		}

		if !foundRoomExit {
			c := configs.GetUserInterfaceConfig().Formats

			if forceRoomId == room.RoomId {
				return true, nil
			}

			destRoom := rooms.LoadRoom(forceRoomId)
			if destRoom == nil {
				return true, nil
			}

			room.RemoveMob(mob.InstanceId)
			destRoom.AddMob(mob.InstanceId)

			// Tell the old room they are leaving
			room.SendText(
				fmt.Sprintf(string(c.ExitRoomMessageWrapper),
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> runs off suddenly.`, mob.Character.Name),
				))

			// Tell the new room they have arrived

			destRoom.SendText(
				fmt.Sprintf(string(c.EnterRoomMessageWrapper),
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> enters from nearby.`, mob.Character.Name),
				))

			return true, nil

		}
	}

	exitName := ``
	goRoomId := 0

	exitName, goRoomId = room.FindExitByName(rest)

	if rest == `home` {
		mob.Command(`pathto home`)
		return true, nil
	}

	exitInfo, _ := room.GetExitInfo(exitName)
	if exitInfo.Lock.IsLocked() {

		mob.Command(fmt.Sprintf(`emote tries to go the <ansi fg="exit">%s</ansi> exit, but it's locked.`, exitName))

		return true, nil
	}

	if exitName != `` {

		// Load current room details
		destRoom := rooms.LoadRoom(goRoomId)
		if destRoom == nil {
			return false, fmt.Errorf(`room %d not found`, goRoomId)
		}

		// Grab the exit in the target room that leads to this room (if any)
		enterFromExit := destRoom.FindExitTo(room.RoomId)

		if len(enterFromExit) < 1 {
			enterFromExit = "somewhere"
		} else {

			// Entering through the other side unlocks this side
			exitInfo, _ := destRoom.GetExitInfo(enterFromExit)

			if exitInfo.Lock.IsLocked() {

				// For now, mobs won't go through doors if it unlocks them.
				return true, nil

				//destRoom.Exits[enterFromExit] = exitInfo
			}

			enterFromExit = fmt.Sprintf(`the <ansi fg="exit">%s</ansi>`, enterFromExit)
		}

		room.RemoveMob(mob.InstanceId)
		destRoom.AddMob(mob.InstanceId)

		c := configs.GetUserInterfaceConfig().Formats

		// Tell the old room they are leaving
		room.SendText(
			fmt.Sprintf(string(c.ExitRoomMessageWrapper),
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> leaves towards the <ansi fg="exit">%s</ansi> exit.`, mob.Character.Name, exitName),
			))

		// Tell the new room they have arrived

		destRoom.SendText(
			fmt.Sprintf(string(c.EnterRoomMessageWrapper),
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> enters from %s.`, mob.Character.Name, enterFromExit),
			))

		destRoom.SendTextToExits(`You hear someone moving around.`, true, room.GetPlayers(rooms.FindAll)...)

		room.PlaySound(`room-exit`, `movement`)
		destRoom.PlaySound(`room-enter`, `movement`)

		// We want the `waypoint` onPath event triggered right after they enter the room.
		if currentStep := mob.Path.Current(); currentStep != nil && currentStep.Waypoint() {

			// Anytime a mob reaches a waypoint, introduce a 1 second delay before they can perform any additional commands.
			// This gives a more natural feel to mob behavior, and gives those following a moment to catch up before the mob does something.
			mob.Command("noop", 1)

			if endPathingAndSkip, _ := scripting.TryMobScriptEvent("onPath", mob.InstanceId, 0, ``, map[string]any{`status`: `waypoint`}); endPathingAndSkip {
				mob.Path.Clear()
			}
		}

		return true, nil
	}

	return false, nil
}
