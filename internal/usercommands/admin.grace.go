package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const GraceOfRenewalBuffId = 100

func Grace(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 || (len(args) == 1 && args[0] == "all") {
		ApplyGraceToAll()
		onlineCount := len(users.GetOnlineUserIds())
		user.SendText(fmt.Sprintf(`<ansi fg="green-bold">*** Grace of Renewal applied to all %d online players ***</ansi>`, onlineCount))
		return true, nil
	}

	targetUserId := user.UserId
	targetMobInstanceId := 0
	targetName := "yourself"

	if len(args) >= 1 && args[0] != "all" {
		if room == nil {
			room = rooms.LoadRoom(user.Character.RoomId)
			if room == nil {
				return false, fmt.Errorf(`room %d not found`, user.Character.RoomId)
			}
		}
		targetUserId, targetMobInstanceId = room.FindByName(args[0])
	}

	buffSpec := buffs.GetBuffSpec(GraceOfRenewalBuffId)
	if buffSpec == nil {
		user.SendText(`Error: Grace of Renewal buff not found. Please ensure buff ID 100 exists.`)
		return true, nil
	}

	if targetUserId > 0 && targetUserId != user.UserId {
		if targetUser := users.GetByUserId(targetUserId); targetUser != nil {
			targetUser.AddBuff(GraceOfRenewalBuffId, "admin-grace")
			targetName = targetUser.Character.Name
			targetUser.SendText(`<ansi fg="cyan-bold">*** You have been granted the Grace of Renewal ***</ansi>`)
			targetUser.SendText(`<ansi fg="cyan">Aggressive creatures will not attack you during this grace period.</ansi>`)
		} else {
			user.SendText(`Target user not found.`)
			return true, nil
		}
	} else if targetMobInstanceId > 0 {
		if targetMob := mobs.GetInstance(targetMobInstanceId); targetMob != nil {
			targetMob.AddBuff(GraceOfRenewalBuffId, "admin-grace")
			targetName = targetMob.Character.Name
		} else {
			user.SendText(`Target mob not found.`)
			return true, nil
		}
	} else {
		user.AddBuff(GraceOfRenewalBuffId, "admin-grace")
		targetName = "yourself"
	}
	if targetName == "yourself" {
		user.SendText(`<ansi fg="cyan-bold">*** You have granted yourself the Grace of Renewal ***</ansi>`)
		user.SendText(`<ansi fg="cyan">Aggressive creatures will not attack you during this time.</ansi>`)
	} else {
		user.SendText(fmt.Sprintf(`<ansi fg="green">Grace of Renewal applied to %s.</ansi>`, targetName))
	}

	if targetName != "yourself" {
		room.SendText(
			fmt.Sprintf(`<ansi fg="cyan">A divine light briefly surrounds %s.</ansi>`, targetName),
			user.UserId,
		)
	}

	return true, nil
}

func ApplyGraceToAll() {
	buffSpec := buffs.GetBuffSpec(GraceOfRenewalBuffId)
	if buffSpec == nil {
		mudlog.Error("Grace", "error", "Grace of Renewal buff spec not found")
		return
	}

	for _, userId := range users.GetOnlineUserIds() {
		if user := users.GetByUserId(userId); user != nil {
			user.AddBuff(GraceOfRenewalBuffId, "admin-grace")
			user.SendText(`<ansi fg="cyan-bold">*** You have been granted the Grace of Renewal ***</ansi>`)
			user.SendText(`<ansi fg="cyan">Aggressive creatures will not attack you during this grace period.</ansi>`)
		}
	}
}
