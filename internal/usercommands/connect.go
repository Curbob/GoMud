package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Connect is a thin wrapper so room scripts can reliably handle `connect ...`
// without depending on fallback command parsing.
func Connect(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	handled, err := TryRoomScripts("connect "+rest, "connect", rest, user.UserId)
	if handled || err != nil {
		return handled, err
	}

	user.SendText("Connect what?")
	return true, nil
}
