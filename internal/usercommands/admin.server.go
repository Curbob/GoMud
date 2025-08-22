package usercommands

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	memoryReportCache = map[string]util.MemoryResult{}
	errValueLocked    = errors.New("This config value is locked. You must edit the config file directly.")
)

const (
	newValuePrompt = `New value for <ansi fg="6">%s</ansi>`
)

/*
* Role Permissions:
* server 				(All)
 */
func Server(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.server", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	args := util.SplitButRespectQuotes(rest)

	// Handle copyover-related commands
	if args[0] == "copyover" || args[0] == "restart" || args[0] == "shutdown" {
		return server_Copyover(args[0], strings.TrimSpace(rest[len(args[0]):]), user, room, flags)
	}

	if args[0] == "config" {
		return server_Config(strings.TrimSpace(rest[1:]), user, room, flags)
	}

	if args[0] == "set" {

		args = args[1:]

		if len(args) < 1 {

			user.SendText(``)

			cfgData := configs.GetConfig().AllConfigData()
			cfgKeys := make([]string, 0, len(cfgData))
			for k := range cfgData {
				cfgKeys = append(cfgKeys, k)
			}

			// sort the keys
			slices.Sort(cfgKeys)

			lastPrefix := ``
			longestKey := 0

			for _, k := range cfgKeys {
				if len(k) > longestKey {
					longestKey = len(k)
				}
			}

			lineLength := 158 - longestKey

			for _, k := range cfgKeys {
				displayName := k
				nameColorized := ``
				if strings.Index(k, `.`) != -1 {
					parts := strings.Split(k, `.`)
					if len(parts) > 1 {
						if lastPrefix != parts[0] {
							lastPrefix = parts[0]
							user.SendText(``)
						}
						nameColorized = `<ansi fg="yellow">` + parts[0] + `.</ansi>`
						displayName = strings.Join(parts[1:], `.`)
					}
				}
				extraSpace := strings.Repeat(` `, longestKey-len(k))

				user.SendText(`<ansi fg="yellow-bold">` + nameColorized + displayName + `</ansi>: <ansi fg="red-bold">` + extraSpace + util.SplitStringNL(fmt.Sprintf(`%v`, cfgData[k]), lineLength, strings.Repeat(` `, longestKey+2)) + `</ansi>`)

			}

			return true, nil
		}

		if args[0] == "day" {
			gametime.SetToDay(-1)
			gd := gametime.GetDate()
			user.SendText(`Time set to ` + gd.String())
			return true, nil
		} else if args[0] == "night" {
			gametime.SetToNight(-1)
			gd := gametime.GetDate()
			user.SendText(`Time set to ` + gd.String())
			return true, nil
		}

		configName := strings.ToLower(args[0])
		configValue := strings.Join(args[1:], ` `)

		if err := configs.SetVal(configName, configValue); err != nil {
			user.SendText(fmt.Sprintf(`config change error: %s=%s (%s)`, configName, configValue, err))
			return true, nil
		}

		user.SendText(fmt.Sprintf(`config changed: %s=%s`, configName, configValue))

		return true, nil
	}

	if rest == "reload-ansi" {
		templates.LoadAliases()
		user.SendText(`ansi aliases reloaded`)
		return true, nil
	}

	if rest == "ansi-strip" {
		templates.SetAnsiFlag(templates.AnsiTagsStrip)
	}

	if rest == "ansi-mono" {
		templates.SetAnsiFlag(templates.AnsiTagsMono)
	}

	if rest == "ansi-normal" {
		templates.SetAnsiFlag(templates.AnsiTagsDefault)
	}

	if rest == "stats" || rest == "info" {

		//
		// General Go stats
		//
		user.SendText(``)
		user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">IP/Port:</ansi>    <ansi fg="red">%s</ansi>`, util.GetServerAddress()))
		user.SendText(``)

		//
		// Special timing related stats
		//
		headers := []string{"Routine", "Avg", "Low", "High", "Ct", "/sec"}
		rows := [][]string{}
		formatting := []string{`<ansi fg="yellow-bold">%s</ansi>`, `<ansi fg="cyan-bold">%s</ansi>`, `<ansi fg="cyan-bold">%s</ansi>`, `<ansi fg="cyan-bold">%s</ansi>`, `<ansi fg="black-bold">%s</ansi>`, `<ansi fg="black-bold">%s</ansi>`}

		allTimers := map[string]util.Accumulator{}
		allNames := []string{}

		times := util.GetTimeTrackers()
		for _, timeAcc := range times {

			allNames = append(allNames, timeAcc.Name)
			allTimers[timeAcc.Name] = timeAcc
		}

		sort.Strings(allNames)
		for _, name := range allNames {
			acc := allTimers[name]
			lowest, highest, average, ct := acc.Stats()
			rows = append(rows, []string{acc.Name,
				fmt.Sprintf(`%4.3fms`, average*1000),
				fmt.Sprintf(`%4.3fms`, lowest*1000),
				fmt.Sprintf(`%4.3fms`, highest*1000),
				fmt.Sprintf(`%d`, int(ct)),
				fmt.Sprintf(`%4.3f`, ct/time.Since(acc.Start).Seconds()),
			})
		}

		tblData := templates.GetTable(`Timer Stats`, headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
		user.SendText(tplTxt)

		//
		// Alternative rendering
		//
		memRepHeaders := []string{"Section  ", "Items    ", "KB       ", "MB       ", "GB       ", "Change   "}
		memRepFormatting := []string{`<ansi fg="yellow-bold">%s</ansi>`,
			`<ansi fg="black-bold">%s</ansi>`,
			`<ansi fg="cyan-bold">%s</ansi>`,
			`<ansi fg="red">%s</ansi>`,
			`<ansi fg="red-bold">%s</ansi>`,
			`<ansi fg="black-bold">%s</ansi>`}

		memRepRows := [][]string{}
		memRepTotalTotal := uint64(0)

		sectionNames, memReports := util.GetMemoryReport()

		for idx, memReport := range memReports {

			sectionName := sectionNames[idx]

			tmpRowStorage := map[string]util.MemoryResult{}
			var memRepRowNames []string = []string{}
			var memRepTotal uint64 = 0

			for name, memResult := range memReport {
				usage := memResult.Memory
				memRepRowNames = append(memRepRowNames, name)
				tmpRowStorage[name] = memResult
				memRepTotal += usage
			}

			memRepRows = append(memRepRows, []string{`[ ` + sectionName + ` ]`, ``, ``, ``, ``, ``})
			sort.Strings(memRepRowNames)
			for _, name := range memRepRowNames {

				var rowData []string

				var prevString string = ``
				var prevCtString string = ``
				if cachedMemResult, ok := memoryReportCache[name]; ok {
					val := cachedMemResult.Memory
					if val > tmpRowStorage[name].Memory { // It has gone down
						prevString = fmt.Sprintf(`↓%s`, util.FormatBytes(val-tmpRowStorage[name].Memory))
					} else if val < tmpRowStorage[name].Memory {
						prevString = fmt.Sprintf(`↑%s`, util.FormatBytes(tmpRowStorage[name].Memory-val))
					}

					ct := cachedMemResult.Count
					if ct > tmpRowStorage[name].Count { // It has gone down
						prevCtString = fmt.Sprintf(`(↓%d)`, ct-tmpRowStorage[name].Count)
					} else if ct < tmpRowStorage[name].Count {
						prevCtString = fmt.Sprintf(`(↑%d)`, tmpRowStorage[name].Count-ct)
					}
				}
				memoryReportCache[name] = tmpRowStorage[name] // Cache the new val

				// foramt the new val
				bFormatted := util.FormatBytes(tmpRowStorage[name].Memory)

				count := ``
				if tmpRowStorage[name].Count > 0 {
					count = fmt.Sprintf(`%d %s`, tmpRowStorage[name].Count, prevCtString)
				}
				if strings.Contains(bFormatted, `KB`) {
					rowData = []string{name, count, bFormatted, ``, ``, prevString}
				} else if strings.Contains(bFormatted, `MB`) {
					rowData = []string{name, count, ``, bFormatted, ``, prevString}
				} else if strings.Contains(bFormatted, `GB`) {
					rowData = []string{name, count, ``, ``, bFormatted, prevString}
				} else {
					rowData = []string{name, count, ``, ``, ``, prevString}
				}

				memRepRows = append(memRepRows, rowData)
			}
			memRepRows = append(memRepRows, []string{``, ``, ``, ``, ``, ``})

			if sectionName != `Go` {
				memRepTotalTotal += memRepTotal
			}
			memRepTotal = 0
		}

		var rowData []string

		var name string = `Total (Non Go)`
		var prevString string = ``
		if cachedMemResult, ok := memoryReportCache[name]; ok {
			val := cachedMemResult.Memory
			if val > memRepTotalTotal { // It has gone down
				prevString = fmt.Sprintf(`↓%s`, util.FormatBytes(val-memRepTotalTotal))
			} else if val < memRepTotalTotal {
				prevString = fmt.Sprintf(`↑%s`, util.FormatBytes(memRepTotalTotal-val))
			}
		}

		memoryReportCache[name] = util.MemoryResult{memRepTotalTotal, 0} // Cache the new val

		bFormatted := util.FormatBytes(memRepTotalTotal)
		if strings.Contains(bFormatted, `KB`) {
			rowData = []string{`Total (Non Go)`, ``, bFormatted, ``, ``, prevString}
		} else if strings.Contains(bFormatted, `MB`) {
			rowData = []string{`Total (Non Go)`, ``, ``, bFormatted, ``, prevString}
		} else if strings.Contains(bFormatted, `GB`) {
			rowData = []string{`Total (Non Go)`, ``, ``, ``, bFormatted, prevString}
		} else {
			rowData = []string{`Total (Non Go)`, ``, ``, ``, ``, prevString}
		}

		memRepRows = append(memRepRows, rowData)
		memRepTblData := templates.GetTable(`Specific Memory`, memRepHeaders, memRepRows, memRepFormatting)
		memRepTxt, _ := templates.Process("tables/generic", memRepTblData, user.UserId)
		user.SendText(memRepTxt)
	}

	return true, nil
}

func server_Copyover(action string, rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Get the copyover manager
	manager := copyover.GetManager()

	// Parse rest for countdown or cancel
	args := strings.Fields(strings.ToLower(rest))

	switch action {
	case "copyover", "restart":
		// If no arguments, show help for this specific command
		if len(args) == 0 {
			templateName := fmt.Sprintf("admincommands/help/command.server.%s", action)
			if helpOutput, err := templates.Process(templateName, nil, user.UserId); err == nil {
				user.SendText(helpOutput)
			} else {
				// Fallback to basic help if template not found
				user.SendText(fmt.Sprintf(`<ansi fg="yellow">Usage:</ansi> server %s [now|<seconds>|cancel|status]`, action))
			}
			return true, nil
		}

		// Check for cancel subcommand
		if args[0] == "cancel" {
			if err := manager.Cancel(); err != nil {
				user.SendText(fmt.Sprintf(`<ansi fg="red">%s</ansi>`, err.Error()))
				return true, nil
			}
			user.SendText(`<ansi fg="green">Server ` + action + ` cancelled.</ansi>`)
			mudlog.Info("AdminCommand", "userId", user.UserId, "user", user.Character.Name, "command", fmt.Sprintf("server %s cancel", action))
			return true, nil
		}

		// Check for status subcommand
		if args[0] == "status" {
			status := manager.GetStatus()
			if status.State == copyover.StateIdle {
				user.SendText(`<ansi fg="yellow">No ` + action + ` scheduled.</ansi>`)
			} else {
				user.SendText(fmt.Sprintf(`<ansi fg="yellow">Server %s Status:</ansi>`, action))
				user.SendText(fmt.Sprintf(`  State: <ansi fg="cyan">%s</ansi>`, status.State))
				if !status.ScheduledFor.IsZero() {
					remaining := time.Until(status.ScheduledFor).Seconds()
					user.SendText(fmt.Sprintf(`  Scheduled: <ansi fg="cyan">%.0f seconds</ansi>`, remaining))
				}
				if status.Progress > 0 {
					user.SendText(fmt.Sprintf(`  Progress: <ansi fg="cyan">%d%%</ansi>`, status.Progress))
				}
			}
			return true, nil
		}

		// Parse countdown - "now" means immediate, numbers mean countdown
		countdown := 0
		if args[0] == "now" {
			countdown = 0 // Explicitly immediate
		} else {
			// Try to parse as number
			if seconds, err := strconv.Atoi(args[0]); err == nil && seconds > 0 {
				countdown = seconds
			} else {
				user.SendText(fmt.Sprintf(`<ansi fg="red">Invalid argument: %s</ansi>`, args[0]))
				user.SendText(`Use "now" for immediate or a number of seconds for countdown.`)
				return true, nil
			}
		}

		// Set up the appropriate action
		build := (action == "copyover")
		actionText := "restart"
		if build {
			actionText = "copyover"
		}

		options := copyover.Options{
			Countdown:   countdown,
			Reason:      fmt.Sprintf("Server %s initiated by %s", actionText, user.Character.Name),
			InitiatedBy: user.UserId,
			Build:       build,
		}

		// Send appropriate messages based on countdown
		if countdown > 0 {
			user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">*** SERVER %s SCHEDULED ***</ansi>`, strings.ToUpper(actionText)))
			user.SendText(fmt.Sprintf(`<ansi fg="cyan">Server will %s in %d seconds.</ansi>`, actionText, countdown))
			user.SendText(fmt.Sprintf(`Use <ansi fg="command">server %s cancel</ansi> to abort.`, action))
		} else {
			user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">*** SERVER %s INITIATED ***</ansi>`, strings.ToUpper(actionText)))
			if build {
				user.SendText(`<ansi fg="cyan">Rebuilding executable and restarting server...</ansi>`)
			} else {
				user.SendText(`<ansi fg="cyan">Restarting server without rebuild...</ansi>`)
			}
		}

		if err := manager.Initiate(options); err != nil {
			user.SendText(fmt.Sprintf(`<ansi fg="red">%s failed: %s</ansi>`, strings.Title(actionText), err.Error()))
			return true, nil
		}

		// Log the action
		logCmd := fmt.Sprintf("server %s", action)
		if countdown > 0 {
			logCmd += fmt.Sprintf(" %d", countdown)
		}
		mudlog.Info("AdminCommand", "userId", user.UserId, "user", user.Character.Name, "command", logCmd)
		return true, nil

	case "shutdown":
		// TODO: Implement proper shutdown
		// This needs to save everything and exit cleanly
		user.SendText(`<ansi fg="red">Server shutdown is not yet implemented.</ansi>`)
		user.SendText(`Use system signals (Ctrl+C) to shut down the server properly.`)
		return true, nil

	default:
		user.SendText(`<ansi fg="red">Unknown server action: ` + action + `</ansi>`)
		return true, nil
	}
}

func server_Config(_ string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Get if already exists, otherwise create new
	cmdPrompt, isNew := user.StartPrompt(`server config`, "")

	if isNew {
		user.SendText(``)
		menuOptions, _ := getConfigOptions("")
		tplTxt, _ := templates.Process("tables/numbered-list", menuOptions, user.UserId)
		user.SendText(tplTxt)
	}

	configPrefix := ""
	if selection, ok := cmdPrompt.Recall("config-selected"); ok {
		configPrefix = selection.(string)
	}

	if configPrefix != "" {
		allConfigData := configs.GetConfig().AllConfigData()
		if configVal, ok := allConfigData[configPrefix]; ok {

			if !isEditAllowed(configPrefix) {
				user.SendText(errValueLocked.Error())
				user.ClearPrompt()
				return true, nil
			}

			question := cmdPrompt.Ask(fmt.Sprintf(newValuePrompt, configPrefix), []string{fmt.Sprintf("%v", configVal)}, fmt.Sprintf("%v", configVal))
			if !question.Done {
				return true, nil
			}

			user.ClearPrompt()

			err := configs.SetVal(configPrefix, question.Response)
			if err == nil {
				allConfigData := configs.GetConfig().AllConfigData()
				user.SendText(``)
				user.SendText(fmt.Sprintf(`<ansi fg="6">%s</ansi> has been set to: <ansi fg="9">%s<ansi>`, configPrefix, allConfigData[configPrefix]))
				user.SendText(``)
				return true, nil
			}
			user.SendText(err.Error())
			return true, nil
		}
	}

	question := cmdPrompt.Ask(`Choose a config option, or "quit":`, []string{``}, ``)
	if !question.Done {
		return true, nil
	}

	if question.Response == "quit" {
		user.SendText("Quitting...")
		user.ClearPrompt()
		return true, nil
	}

	fullPath := strings.ToLower(configPrefix)
	if fullPath != `` {
		fullPath += "."
	}
	fullPath += question.Response

	if !isEditAllowed(fullPath) {
		user.SendText(errValueLocked.Error())
		question.RejectResponse()
		return true, nil
	}

	menuOptions, ok := getConfigOptions(fullPath)
	if !ok {
		question.RejectResponse()
		menuOptions, _ = getConfigOptions("")
		fullPath = strings.ToLower(configPrefix)
	} else {

		if len(menuOptions) == 1 {
			fullPath = menuOptions[0].Id.(string)

			cmdPrompt.Store("config-selected", fullPath)

			if !isEditAllowed(fullPath) {
				user.SendText(errValueLocked.Error())
				user.ClearPrompt()
				return true, nil
			}

			allConfigData := configs.GetConfig().AllConfigData()
			if configVal, ok := allConfigData[fullPath]; ok {

				cmdPrompt.Ask(fmt.Sprintf(newValuePrompt, fullPath), []string{fmt.Sprintf("%v", configVal)}, fmt.Sprintf("%v", configVal))
				return true, nil
			}
		}

		cmdPrompt.Store("config-selected", fullPath)
	}

	if fullPath != "" {
		user.SendText(``)
		user.SendText(`   [<ansi fg="6">` + fullPath + `</ansi>]`)
	}

	tplTxt, _ := templates.Process("tables/numbered-list", menuOptions, user.UserId)
	user.SendText(tplTxt)

	question.RejectResponse()

	return true, nil
}

func isEditAllowed(configPath string) bool {

	configPath = strings.ToLower(configPath)

	if strings.HasSuffix(configPath, "locked") {
		return false
	}

	sc := configs.GetServerConfig()
	for _, v := range sc.Locked {
		if strings.HasPrefix(configPath, strings.ToLower(v)) {
			return false
		}
	}

	return true
}

func getConfigOptions(input string) ([]templates.NameDescription, bool) {

	input = strings.ToLower(input)

	configOptions := []templates.NameDescription{}

	allConfigData := configs.GetConfig().AllConfigData()
	pathLookup := map[string]string{}
	for name, _ := range allConfigData {

		lowerName := strings.ToLower(name)
		pathLookup[lowerName] = name

		builtPath := ""
		for _, namePart := range strings.Split(name, ".") {
			builtPath += namePart
			if _, ok := pathLookup[builtPath]; !ok {
				pathLookup[strings.ToLower(builtPath)] = builtPath
			}
			builtPath += "."
		}
	}

	inputProperCase := input
	if caseCheck, ok := pathLookup[input]; ok {

		inputProperCase = caseCheck

		// Is this a full config path?
		if configVal, ok := allConfigData[inputProperCase]; ok {

			configOptions = append(configOptions, templates.NameDescription{
				Id:          inputProperCase,
				Name:        inputProperCase,
				Description: fmt.Sprintf("%v", configVal),
			})

			return configOptions, true

		}

	} else if input != "" {
		return configOptions, false
	}

	// Find which partial path we are on and populate options
	usedNames := map[string]struct{}{}
	for fullConfigPath, configVal := range allConfigData {

		if input != "" {
			if len(fullConfigPath) <= len(input) || fullConfigPath[0:len(inputProperCase)] != inputProperCase {
				continue
			}
		}

		nextConfigPathSection := fullConfigPath
		if len(inputProperCase) > 0 {
			nextConfigPathSection = nextConfigPathSection[len(inputProperCase)+1:]
		}

		desc := "..."
		if dotIdx := strings.Index(nextConfigPathSection, "."); dotIdx != -1 {
			nextConfigPathSection = nextConfigPathSection[:dotIdx]
		} else {
			desc = fmt.Sprintf("%v", configVal)
		}

		if _, ok := usedNames[nextConfigPathSection]; ok {
			continue
		}

		usedNames[nextConfigPathSection] = struct{}{}

		pathWithSection := nextConfigPathSection
		if len(inputProperCase) > 0 {
			pathWithSection = inputProperCase + "." + pathWithSection
		}

		configOptions = append(configOptions, templates.NameDescription{
			Id:          pathWithSection,
			Name:        nextConfigPathSection,
			Description: desc,
		})

	}

	if len(configOptions) > 0 {
		sort.Slice(configOptions, func(i, j int) bool {
			return configOptions[i].Name < configOptions[j].Name
		})
	}

	return configOptions, true
}
