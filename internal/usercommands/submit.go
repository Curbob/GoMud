package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Submit is a thin wrapper so room scripts can reliably handle `submit ...`
// without depending on fallback command parsing.
func Submit(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	handled, err := TryRoomScripts("submit "+rest, "submit", rest, user.UserId)
	if handled || err != nil {
		return handled, err
	}

	user.SendText("Submit what?")
	return true, nil
}
