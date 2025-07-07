package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Status(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	statNames := user.Character.Stats.GetStatInfoNames()

	if rest != `` {

		if rest != `train` {
			user.SendText("status WHAT???")
			return true, nil
		}

		user.DidTip(`status train`, true)

		cmdPrompt, isNew := user.StartPrompt(`status`, rest)

		if isNew {
			tplTxt, _ := templates.Process("character/status-train", user, user.UserId)
			user.SendText(tplTxt)
		}

		question := cmdPrompt.Ask(`Increase which?`, append(statNames, `quit`), `quit`)
		if !question.Done {
			return true, nil
		}

		if question.Response == `quit` {
			user.ClearPrompt()
			return true, nil
		}

		match, closeMatch := util.FindMatchIn(question.Response, statNames...)

		question.RejectResponse() // Always reset this question, since we want to keep reusing it.

		if user.Character.StatPoints < 1 {
			user.SendText(`Oops! You have no stat points to spend!`)
			user.ClearPrompt()
			return true, nil
		}
		selection := match
		if match == `` {
			selection = closeMatch
		}

		before := 0
		after := 0
		spent := 0

		// TODO: Now that we have removed the switch statement and replaced it with a map this command will always succeed so spent = 1 will always be true if the character has training points to spend. Need to come back to this and add some checks to see if the stat can actually be trained.
		before = user.Character.Stats.Get(selection).Value - user.Character.Stats.Get(selection).Mods
		user.Character.Stats.Get(selection).Training += 1
		spent = 1

		if spent > 0 {
			after = before + 1
			user.Character.StatPoints -= 1

			user.Character.Validate()

			user.SendText(fmt.Sprintf(term.CRLFStr+`<ansi fg="210">Your <ansi fg="yellow">%s</ansi> training improves from <ansi fg="201">%d</ansi> to <ansi fg="201">%d</ansi>!</ansi>`, selection, before, after))

			events.AddToQueue(events.CharacterTrained{UserId: user.UserId})
		}

		tplTxt, _ := templates.Process("character/status-train", user, user.UserId)

		if spent > 0 {
			tplTxt = strings.Replace(tplTxt, `fakeprop="`+selection+`"`, `bg="highlight"`, 1)
		}

		user.SendText(tplTxt)

		return true, nil
	}

	tplTxt, _ := templates.Process("character/status", user, user.UserId)
	user.SendText(tplTxt)

	Inventory(``, user, room, flags)

	return true, nil
}
