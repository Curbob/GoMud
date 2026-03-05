package gambling

import (
	"embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	//go:embed files/*
	files embed.FS
)

func init() {
	plug := plugins.New(`gambling`, `1.0`)

	if err := plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	// Register gambling commands
	plug.AddUserCommand(`gamble`, GambleCommand, true, false)
	plug.AddUserCommand(`dice`, DiceCommand, true, false)
	plug.AddUserCommand(`highorlow`, HighOrLowCommand, true, false)
}

// GambleCommand - Main gambling menu/help
func GambleCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !room.IsBar {
		user.SendText(`You need to be at a bar or tavern to gamble.`)
		return true, nil
	}

	user.SendText(``)
	user.SendText(`<ansi fg="yellow">╔══════════════════════════════════════╗</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>    <ansi fg="white-bold">🎲 GAMBLING GAMES 🎲</ansi>           <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">╠══════════════════════════════════════╣</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>                                      <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>  <ansi fg="command">dice [amount]</ansi>                    <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>    Roll dice against the house.      <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>    Highest roll wins!                <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>                                      <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>  <ansi fg="command">highorlow [amount]</ansi>               <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>    Guess if next roll is higher      <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>    or lower. Win streak = bonus!     <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">║</ansi>                                      <ansi fg="yellow">║</ansi>`)
	user.SendText(`<ansi fg="yellow">╚══════════════════════════════════════╝</ansi>`)
	user.SendText(``)
	user.SendText(fmt.Sprintf(`You have <ansi fg="gold">%d gold</ansi> on hand.`, user.Character.Gold))
	user.SendText(``)

	return true, nil
}

// DiceCommand - Simple dice roll against the house
func DiceCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !room.IsBar {
		user.SendText(`You need to be at a bar or tavern to gamble.`)
		return true, nil
	}

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		user.SendText(`Usage: <ansi fg="command">dice [amount]</ansi> - Bet gold on a dice roll against the house.`)
		user.SendText(fmt.Sprintf(`You have <ansi fg="gold">%d gold</ansi> on hand.`, user.Character.Gold))
		return true, nil
	}

	betAmount, err := strconv.Atoi(args[0])
	if err != nil || betAmount < 1 {
		user.SendText(`You must bet at least 1 gold.`)
		return true, nil
	}

	if betAmount > user.Character.Gold {
		user.SendText(fmt.Sprintf(`You don't have that much gold! You only have <ansi fg="gold">%d gold</ansi>.`, user.Character.Gold))
		return true, nil
	}

	// Cap bet at 1000 to prevent abuse
	if betAmount > 1000 {
		user.SendText(`The house limit is <ansi fg="gold">1000 gold</ansi> per roll.`)
		return true, nil
	}

	// Roll 2d6 for player and house
	playerRoll1 := util.RollDice(1, 6)
	playerRoll2 := util.RollDice(1, 6)
	playerTotal := playerRoll1 + playerRoll2

	houseRoll1 := util.RollDice(1, 6)
	houseRoll2 := util.RollDice(1, 6)
	houseTotal := houseRoll1 + houseRoll2

	user.SendText(``)
	user.SendText(fmt.Sprintf(`<ansi fg="yellow">You bet <ansi fg="gold">%d gold</ansi> and roll the dice...</ansi>`, betAmount))
	user.SendText(``)

	// Dramatic pause effect via text
	user.SendText(fmt.Sprintf(`<ansi fg="white">Your roll:</ansi> 🎲 <ansi fg="cyan-bold">%d</ansi> + 🎲 <ansi fg="cyan-bold">%d</ansi> = <ansi fg="white-bold">%d</ansi>`, playerRoll1, playerRoll2, playerTotal))
	user.SendText(fmt.Sprintf(`<ansi fg="white">House roll:</ansi> 🎲 <ansi fg="red">%d</ansi> + 🎲 <ansi fg="red">%d</ansi> = <ansi fg="white-bold">%d</ansi>`, houseRoll1, houseRoll2, houseTotal))
	user.SendText(``)

	if playerTotal > houseTotal {
		winnings := betAmount
		user.Character.Gold += winnings
		user.SendText(fmt.Sprintf(`<ansi fg="green-bold">🎉 YOU WIN!</ansi> You pocket <ansi fg="gold">%d gold</ansi>!`, winnings))
		
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> wins <ansi fg="gold">%d gold</ansi> at dice! 🎲`, user.Character.Name, winnings), user.UserId)
		
		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: winnings,
		})
	} else if playerTotal < houseTotal {
		user.Character.Gold -= betAmount
		user.SendText(fmt.Sprintf(`<ansi fg="red">💸 You lose!</ansi> The house takes your <ansi fg="gold">%d gold</ansi>.`, betAmount))
		
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> loses <ansi fg="gold">%d gold</ansi> at dice.`, user.Character.Name, betAmount), user.UserId)
		
		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: -betAmount,
		})
	} else {
		user.SendText(`<ansi fg="yellow">🤝 It's a tie!</ansi> Your bet is returned.`)
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> ties with the house at dice.`, user.Character.Name), user.UserId)
	}

	user.SendText(``)
	user.SendText(fmt.Sprintf(`You now have <ansi fg="gold">%d gold</ansi>.`, user.Character.Gold))

	return true, nil
}

// HighOrLowCommand - Guess if next roll is higher or lower
func HighOrLowCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !room.IsBar {
		user.SendText(`You need to be at a bar or tavern to gamble.`)
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText(`Usage: <ansi fg="command">highorlow [amount]</ansi> - Start a high/low guessing game.`)
		user.SendText(`After the first roll, guess <ansi fg="command">high</ansi> or <ansi fg="command">low</ansi> for the next roll.`)
		user.SendText(`Keep guessing correctly to multiply your winnings!`)
		user.SendText(fmt.Sprintf(`You have <ansi fg="gold">%d gold</ansi> on hand.`, user.Character.Gold))
		return true, nil
	}

	// Check if they're making a guess on an existing game
	if args[0] == `high` || args[0] == `low` || args[0] == `cash` || args[0] == `cashout` {
		return handleHighOrLowGuess(args[0], user, room)
	}

	// Starting a new game
	betAmount, err := strconv.Atoi(args[0])
	if err != nil || betAmount < 1 {
		user.SendText(`You must bet at least 1 gold.`)
		return true, nil
	}

	if betAmount > user.Character.Gold {
		user.SendText(fmt.Sprintf(`You don't have that much gold! You only have <ansi fg="gold">%d gold</ansi>.`, user.Character.Gold))
		return true, nil
	}

	if betAmount > 500 {
		user.SendText(`The house limit for high/low is <ansi fg="gold">500 gold</ansi>.`)
		return true, nil
	}

	// Take the bet
	user.Character.Gold -= betAmount

	// First roll
	firstRoll := util.RollDice(2, 6)

	// Store game state in user's temp data
	user.SetTempData(`highorlow_bet`, betAmount)
	user.SetTempData(`highorlow_roll`, firstRoll)
	user.SetTempData(`highorlow_streak`, 0)

	user.SendText(``)
	user.SendText(fmt.Sprintf(`<ansi fg="yellow">You bet <ansi fg="gold">%d gold</ansi> on High or Low!</ansi>`, betAmount))
	user.SendText(``)
	user.SendText(fmt.Sprintf(`The dice show: 🎲 <ansi fg="cyan-bold">%d</ansi>`, firstRoll))
	user.SendText(``)
	user.SendText(`Will the next roll be <ansi fg="command">high</ansi>er or <ansi fg="command">low</ansi>er?`)
	user.SendText(`(Or <ansi fg="command">highorlow cash</ansi> to take your winnings)`)

	events.AddToQueue(events.EquipmentChange{
		UserId:     user.UserId,
		GoldChange: -betAmount,
	})

	return true, nil
}

func handleHighOrLowGuess(guess string, user *users.UserRecord, room *rooms.Room) (bool, error) {

	betData := user.GetTempData(`highorlow_bet`)
	rollData := user.GetTempData(`highorlow_roll`)
	streakData := user.GetTempData(`highorlow_streak`)

	if betData == nil || rollData == nil {
		user.SendText(`You don't have an active high/low game. Start one with <ansi fg="command">highorlow [amount]</ansi>`)
		return true, nil
	}

	bet := betData.(int)
	lastRoll := rollData.(int)
	streak := 0
	if streakData != nil {
		streak = streakData.(int)
	}

	// Cash out
	if guess == `cash` || guess == `cashout` {
		multiplier := 1.0 + (float64(streak) * 0.5) // 1.0, 1.5, 2.0, 2.5, etc.
		winnings := int(float64(bet) * multiplier)
		
		user.Character.Gold += winnings
		user.SendText(fmt.Sprintf(`<ansi fg="green">You cash out with <ansi fg="gold">%d gold</ansi>!</ansi> (x%.1f multiplier from %d streak)`, winnings, multiplier, streak))
		
		user.SetTempData(`highorlow_bet`, nil)
		user.SetTempData(`highorlow_roll`, nil)
		user.SetTempData(`highorlow_streak`, nil)

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: winnings,
		})

		return true, nil
	}

	// New roll
	newRoll := util.RollDice(2, 6)

	user.SendText(``)
	user.SendText(fmt.Sprintf(`Last roll was <ansi fg="cyan-bold">%d</ansi>, you guessed <ansi fg="white-bold">%s</ansi>...`, lastRoll, strings.ToUpper(guess)))
	user.SendText(fmt.Sprintf(`New roll: 🎲 <ansi fg="cyan-bold">%d</ansi>`, newRoll))

	won := false
	if guess == `high` && newRoll > lastRoll {
		won = true
	} else if guess == `low` && newRoll < lastRoll {
		won = true
	} else if newRoll == lastRoll {
		user.SendText(`<ansi fg="yellow">🤝 It's a tie! You keep your streak but don't advance.</ansi>`)
		user.SetTempData(`highorlow_roll`, newRoll)
		user.SendText(``)
		user.SendText(`Will the next roll be <ansi fg="command">high</ansi>er or <ansi fg="command">low</ansi>er?`)
		return true, nil
	}

	if won {
		streak++
		multiplier := 1.0 + (float64(streak) * 0.5)
		potentialWin := int(float64(bet) * multiplier)
		
		user.SendText(fmt.Sprintf(`<ansi fg="green-bold">✓ Correct!</ansi> Streak: %d (potential win: <ansi fg="gold">%d gold</ansi> at x%.1f)`, streak, potentialWin, multiplier))
		user.SendText(``)
		user.SendText(`Will the next roll be <ansi fg="command">high</ansi>er or <ansi fg="command">low</ansi>er?`)
		user.SendText(`(Or <ansi fg="command">highorlow cash</ansi> to take your winnings)`)
		
		user.SetTempData(`highorlow_roll`, newRoll)
		user.SetTempData(`highorlow_streak`, streak)
	} else {
		user.SendText(`<ansi fg="red">✗ Wrong!</ansi> You lose your bet.`)
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> loses at high/low after a %d streak.`, user.Character.Name, streak), user.UserId)
		
		user.SetTempData(`highorlow_bet`, nil)
		user.SetTempData(`highorlow_roll`, nil)
		user.SetTempData(`highorlow_streak`, nil)
	}

	return true, nil
}
