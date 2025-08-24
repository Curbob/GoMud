//go:generate go run cmd/generate/module-imports.go
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/GoMudEngine/GoMud/internal/audio"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/flags"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/hooks"
	"github.com/GoMudEngine/GoMud/internal/inputhandlers"
	"github.com/GoMudEngine/GoMud/internal/integrations/discord"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/migration"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/version"
	"github.com/gorilla/websocket"

	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/suggestions"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/web"
	_ "github.com/GoMudEngine/GoMud/modules"
	textLang "golang.org/x/text/language"
)

// Version of the binary
// Should be kept in lockstep with github releases
// When updating this version:
// 1. Expect to update the github release version
// 2. Consider whether any migration code is needed for breaking changes, particularly in datafiles (see internal/migration)
const VERSION = "0.9.1"

var (
	sigChan            = make(chan os.Signal, 1)
	workerShutdownChan = make(chan bool, 1)

	serverAlive atomic.Bool

	worldManager = NewWorld(sigChan)

	// Start a pool of worker goroutines
	wg sync.WaitGroup
)

func main() {

	serverStartTime := time.Now()

	// Capture panic and write msg/stack to logs
	defer func() {
		if r := recover(); r != nil {
			mudlog.Error("PANIC", "error", r)
			s := string(debug.Stack())
			for _, str := range strings.Split(s, "\n") {
				mudlog.Error("PANIC", "stack", str)
			}
		}
	}()

	// Setup logging
	mudlog.SetupLogger(
		events.GetLogger(),
		os.Getenv(`LOG_LEVEL`),
		os.Getenv(`LOG_PATH`),
		os.Getenv(`LOG_NOCOLOR`) == ``,
	)

	flags.HandleFlags(VERSION)

	configs.ReloadConfig()
	c := configs.GetConfig()

	// Check if we're recovering from copyover
	isCopyoverRecovery := os.Getenv("GOMUD_COPYOVER") == "1"
	if isCopyoverRecovery {
		mudlog.Info("Copyover", "status", "Recovery mode detected")
	}

	lastKnownVersion, err := version.Parse(string(configs.GetServerConfig().CurrentVersion))
	if err != nil {
		mudlog.Error("Versioning", "error", err)
		os.Exit(1)
	}

	currentVersion, _ := version.Parse(VERSION)

	if err = migration.Run(lastKnownVersion, currentVersion); err != nil {
		mudlog.Error("migration.Run()", "error", err)
		os.Exit(1)
	}

	// Default i18n localize folders
	if len(c.Translation.LanguagePaths) == 0 {
		c.Translation.LanguagePaths = []string{
			path.Join("_datafiles", "localize"),
			path.Join(c.FilePaths.DataFiles.String(), "localize"),
		}
	}

	mudlog.Info(`========================`)
	//
	mudlog.Info(`  _____             `)
	mudlog.Info(` / ____|            `)
	mudlog.Info(`| |  __  ___        `)
	mudlog.Info(`| | |_ |/ _ \       `)
	mudlog.Info(`| |__| | (_) |      `)
	mudlog.Info(` \_____|\___/       `)
	mudlog.Info(` __  __           _ `)
	mudlog.Info(`|  \/  |         | |`)
	mudlog.Info(`| \  / |_   _  __| |`)
	mudlog.Info(`| |\/| | | | |/ _' |`)
	mudlog.Info(`| |  | | |_| | (_| |`)
	mudlog.Info(`|_|  |_|\__,_|\__,_|`)

	//
	mudlog.Info(`========================`)
	//
	cfgData := c.AllConfigData()
	cfgKeys := make([]string, 0, len(cfgData))
	for k := range cfgData {
		cfgKeys = append(cfgKeys, k)
	}

	// sort the keys
	slices.Sort(cfgKeys)
	for _, k := range cfgKeys {
		mudlog.Info("Config", "name", k, "value", cfgData[k])
	}
	//
	mudlog.Info(`========================`)

	// Older versions of GoMud may not have this folder present.
	// Also deleting the folder is a quick way to reset instance state, so this corrects that if it happens.
	os.Mkdir(util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `rooms.instances`), os.ModeDir|0755)

	// Register the plugin filesystem with the template system
	templates.RegisterFS(plugins.GetPluginRegistry())
	usercommands.AddFunctionExporter(plugins.GetPluginRegistry())

	inputhandlers.AddIACHandler(plugins.GetPluginRegistry())

	//
	// System Configurations
	runtime.GOMAXPROCS(int(c.Server.MaxCPUCores))

	// Validate chosen world:
	if err := util.ValidateWorldFiles(`_datafiles/world/default`, c.FilePaths.DataFiles.String()); err != nil {
		mudlog.Error("World Validation", "error", err)
		os.Exit(1)
	}

	language.InitTranslation(language.BundleCfg{
		DefaultLanguage: textLang.Make(c.Translation.DefaultLanguage.String()),
		Language:        textLang.Make(c.Translation.Language.String()),
		LanguagePaths:   c.Translation.LanguagePaths,
	})

	hooks.RegisterListeners()

	// Discord integration
	if webhookUrl := string(c.Integrations.Discord.WebhookUrl); webhookUrl != "" {
		discord.Init(webhookUrl)
		mudlog.Info("Discord", "info", "integration is enabled")
	} else {
		mudlog.Warn("Discord", "info", "integration is disabled")
	}

	mudlog.Info(
		"Starting server",
		"name", string(c.Server.MudName),
	)

	mudlog.Info(`========================`)

	// Initialize combat registry event handlers
	combat.InitializeRegistry()

	// Initialize combat system before loading data files
	// This ensures combat commands are available during module loading
	gamePlayConfig := configs.GetGamePlayConfig()
	mudlog.Info("Combat System", "info", "Initializing combat system", "style", gamePlayConfig.Combat.Style.String())
	if err := combat.SetActiveCombatSystem(gamePlayConfig.Combat.Style.String()); err != nil {
		mudlog.Error("Combat System", "error", err, "module", gamePlayConfig.Combat.Style.String())
		mudlog.Info("Combat System", "info", "Using default combat system")
		// The fallback will be to use the existing hardcoded combat
	} else {
		mudlog.Info("Combat System", "info", "Successfully initialized", "style", gamePlayConfig.Combat.Style.String())
	}

	// Load all the data files up front.
	loadAllDataFiles(false)

	mudlog.Info(`========================`)

	mudlog.Info("Mapper", "status", "precaching")
	timeStart := time.Now()
	mapper.PreCacheMaps()
	mudlog.Info("Mapper", "status", "done", "time taken", time.Since(timeStart))

	mudlog.Info(`========================`)

	// Create the user index
	idx := users.NewUserIndex()
	if !idx.Exists() {
		// Since it doesn't exist yet, that's a good indication we should do a quick format migration check
		users.DoUserMigrations()
	}
	idx.Create()
	idx.Rebuild()
	mudlog.Info("UserIndex", "info", "User index recreated.")

	// Load the round count from the file
	if util.LoadRoundCount(c.FilePaths.DataFiles.String()+`/`+util.RoundCountFilename) == util.RoundCountMinimum {
		gametime.SetToDay(-3)
	}

	gametime.GetZodiac(1) // The first time this is called it randomizes all zodiacs

	scripting.Setup(int(c.Scripting.LoadTimeoutMs), int(c.Scripting.RoomTimeoutMs))

	mudlog.Info(`========================`)

	// Trigger the load plugins event
	plugins.Load(
		configs.GetFilePathsConfig().DataFiles.String(),
	)

	web.SetWebPlugin(plugins.GetPluginRegistry())

	//
	// Capture OS signals to gracefully shutdown the server
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// for testing purposes, enable event debugging
	//events.SetDebug(true)

	//
	// Spin up server listeners
	//

	// Set the server to be alive
	serverAlive.Store(true)

	mudlog.Info(`========================`)

	// Handle copyover recovery if needed
	var preservedListeners map[string]net.Listener
	if isCopyoverRecovery {
		mudlog.Info("Copyover", "status", "Starting recovery process")

		// Recover state before starting listeners
		if err := copyover.Recover(); err != nil {
			mudlog.Error("Copyover", "error", "Recovery failed", "err", err)
			// Continue with normal startup even if recovery fails
		} else {
			mudlog.Info("Copyover", "status", "Recovery completed successfully")
			// Get preserved listeners after recovery
			preservedListeners = copyover.GetPreservedListeners()

			// Set up input handlers for recovered connections
			setupRecoveredConnectionHandlers()
		}
	}

	// Web listener is more complex to preserve due to HTTP server setup
	// For now, always create new web listener
	web.Listen(&wg, HandleWebSocketConnection)

	allServerListeners := make([]net.Listener, 0, len(c.Network.TelnetPort))

	// Check for preserved telnet listeners
	for _, port := range c.Network.TelnetPort {
		if p, err := strconv.Atoi(port); err == nil {
			listenerName := fmt.Sprintf("telnet-%d", p)
			if listener, exists := preservedListeners[listenerName]; exists {
				mudlog.Info("Copyover", "status", "Using preserved telnet listener", "port", p)
				allServerListeners = append(allServerListeners, listener)
				connections.RegisterListener(listenerName, listener)
				// Start accepting connections on preserved listener
				go AcceptOnListener(listener, &wg, int(c.Network.MaxTelnetConnections))
			} else {
				if s := TelnetListenOnPort(``, p, &wg, int(c.Network.MaxTelnetConnections)); s != nil {
					allServerListeners = append(allServerListeners, s)
					connections.RegisterListener(listenerName, s)
				}
			}
		}
	}

	if c.Network.LocalPort > 0 {
		listenerName := fmt.Sprintf("local-%d", c.Network.LocalPort)
		if listener, exists := preservedListeners[listenerName]; exists {
			mudlog.Info("Copyover", "status", "Using preserved local listener", "port", c.Network.LocalPort)
			connections.RegisterListener(listenerName, listener)
			go AcceptOnListener(listener, &wg, 0)
		} else {
			if s := TelnetListenOnPort(`127.0.0.1`, int(c.Network.LocalPort), &wg, 0); s != nil {
				connections.RegisterListener(listenerName, s)
			}
		}
	}

	// Secure telnet local ports - where TLS proxy forwards to
	for _, port := range c.Network.SecureTelnetLocalPort {
		if p, err := strconv.Atoi(port); err == nil && p > 0 {
			listenerName := fmt.Sprintf("secure-local-%d", p)
			if listener, exists := preservedListeners[listenerName]; exists {
				mudlog.Info("Copyover", "status", "Using preserved secure local listener", "port", p)
				connections.RegisterListener(listenerName, listener)
				go AcceptOnListener(listener, &wg, 0)
			} else {
				mudlog.Info("Telnet", "stage", "Listening on secure local port (localhost only)", "port", p)
				if s := TelnetListenOnPort(`127.0.0.1`, p, &wg, 0); s != nil {
					connections.RegisterListener(listenerName, s)
				}
			}
		}
	}

	go worldManager.InputWorker(workerShutdownChan, &wg)
	go worldManager.MainWorker(workerShutdownChan, &wg)

	// If this was a copyover recovery, apply grace to all reconnected users
	if isCopyoverRecovery {
		// Give workers and GMCP time to fully initialize
		go func() {
			time.Sleep(1 * time.Second)
			mudlog.Info("Copyover", "grace", "Applying grace period to all reconnected users")
			usercommands.ApplyGraceToAll()
		}()
	}

	// Register pre-copyover hook to save all rooms
	copyover.RegisterPreCopyoverHook(func() error {
		mudlog.Info("Copyover", "pre-hook", "Saving all rooms")
		if err := rooms.SaveAllRooms(); err != nil {
			mudlog.Error("Copyover", "pre-hook", "Failed to save rooms", "err", err)
			return err
		}
		mudlog.Info("Copyover", "pre-hook", "Rooms saved successfully")
		return nil
	})

	mudlog.Info("Server Ready", "Time Taken", time.Since(serverStartTime))

	// block until a signal comes in
	<-sigChan

	tplTxt, err := templates.Process("goodbye", nil)
	if err != nil {
		mudlog.Error("Template Error", "error", err)
	}

	events.AddToQueue(events.Broadcast{
		Text: templates.AnsiParse(tplTxt),
	})

	serverAlive.Store(false) // immediately stop processing incoming connections

	util.SaveRoundCount(c.FilePaths.DataFiles.String() + `/` + util.RoundCountFilename)

	// some last minute stats reporting
	totalConnections, totalDisconnections := connections.Stats()
	mudlog.Info(
		"Stopping server",
		"LifetimeConnections", totalConnections,
		"LifetimeDisconnects", totalDisconnections,
		"ActiveConnections", totalConnections-totalDisconnections,
	)

	// cleanup all connections
	connections.Cleanup()

	for _, s := range allServerListeners {
		s.Close()
	}

	web.Shutdown()

	// Final plugin save before shutting down
	plugins.Save()

	// Just a goroutine that spins its wheels until the program shuts down")
	go func() {
		for {
			mudlog.Warn("Waiting on workers")
			// sleep for 3 seconds
			time.Sleep(time.Duration(3) * time.Second)
		}
	}()

	// Close the channel, signalling to the worker threads to shutdown.
	close(workerShutdownChan)

	// Wait for all workers to finish their tasks.
	// Otherwise we end up getting flushed file saves incomplete.
	wg.Wait()

	// Give it a second to disaptch any final messages in the event queue
	// Example: discord server shutdown
	time.Sleep(1 * time.Second)
}

func handleTelnetConnection(connDetails *connections.ConnectionDetails, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	mudlog.Info("New Connection", "connectionID", connDetails.ConnectionId(), "remoteAddr", connDetails.RemoteAddr().String())

	// Setup shared state map for this connection's handlers
	// Needs to be created BEFORE the first handler call
	var sharedState map[string]any = make(map[string]any)

	// Add starting handlers

	// Special escape handlers
	connDetails.AddInputHandler("TelnetIACHandler", inputhandlers.TelnetIACHandler)
	connDetails.AddInputHandler("AnsiHandler", inputhandlers.AnsiHandler)
	// Consider a macro handler at this point?
	// Text Processing
	connDetails.AddInputHandler("CleanserInputHandler", inputhandlers.CleanserInputHandler)

	loginHandler := inputhandlers.GetLoginPromptHandler()           // Get the configured handler func
	connDetails.AddInputHandler("LoginPromptHandler", loginHandler) // Add it with a unique name

	// Turn off "line at a time", send chars as typed
	connections.SendTo(
		term.TelnetWILL(term.TELNET_OPT_SUP_GO_AHD),
		connDetails.ConnectionId(),
	)
	// Tell the client we expect chars as they are typed
	connections.SendTo(
		term.TelnetWONT(term.TELNET_OPT_LINE_MODE),
		connDetails.ConnectionId(),
	)

	// Tell the client we intend to echo back what they type
	// So they shouldn't locally echo it

	connections.SendTo(
		term.TelnetWILL(term.TELNET_OPT_ECHO),
		connDetails.ConnectionId(),
	)
	// Request that the client report window size changes as they happen
	connections.SendTo(
		term.TelnetDO(term.TELNET_OPT_NAWS),
		connDetails.ConnectionId(),
	)

	// Send request to change charset
	connections.SendTo(
		term.TelnetRequestChangeCharset.BytesWithPayload(nil),
		connDetails.ConnectionId(),
	)

	// Send request to enable MSP
	connections.SendTo(
		term.MspEnable.BytesWithPayload(nil),
		connDetails.ConnectionId(),
	)

	connections.SendTo(
		term.TelnetSuppressGoAhead.BytesWithPayload(nil),
		connDetails.ConnectionId(),
	)

	clientSetupCommands := "" + //term.AnsiAltModeStart.String() + // alternative mode (No scrollback)
		//term.AnsiCursorHide.String() + // Hide Cursor (Because we will manually echo back)
		//term.AnsiCharSetUTF8.String() + // UTF8 mode
		//term.AnsiReportMouseClick.String() + // Request client to capture and report mouse clicks
		term.AnsiRequestResolution.String() // Request resolution
		//""

	connections.SendTo(
		[]byte(clientSetupCommands),
		connDetails.ConnectionId(),
	)

	plugins.OnNetConnect(connDetails)

	// an input buffer for reading data sent over the network
	inputBuffer := make([]byte, connections.ReadBufferSize)

	// Describes whatever the client sent us
	clientInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		DataIn:       []byte{},
		Buffer:       make([]byte, 0, connections.ReadBufferSize), // DataIn is appended to this buffer after processing
		EnterPressed: false,
		Clipboard:    []byte{},
		History:      connections.InputHistory{},
	}

	if audioConfig := audio.GetFile(`intro`); audioConfig.FilePath != `` {
		v := 100
		if audioConfig.Volume > 0 && audioConfig.Volume <= 100 {
			v = audioConfig.Volume
		}
		connections.SendTo(
			term.MspCommand.BytesWithPayload([]byte("!!MUSIC("+audioConfig.FilePath+" V="+strconv.Itoa(v)+" L=-1 C=1)")),
			clientInput.ConnectionId,
		)
	}

	// --- Send Initial Welcome/Splash ---
	// (This part was mostly correct before)
	splashTxt, _ := templates.Process("login/connect-splash", nil)
	connections.SendTo([]byte(templates.AnsiParse(splashTxt)), connDetails.ConnectionId())

	// --- Trigger the Prompt Handler to initialize state and send the FIRST prompt ---
	// Create a dummy input that signifies "start the process" but has no actual user data/control codes.
	initialTriggerInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		// Ensure flags like EnterPressed are false
	}
	// Call the handler function directly ONCE.
	// This executes the `!ok` block inside the handler, which:
	// 1. Creates the PromptHandlerState in sharedState.
	// 2. Calls advanceAndSendPromptCustom -> sendPromptFunc for the *first* step (username).
	// 3. Returns false (which we ignore here, as we aren't in the main loop yet).
	loginHandler(initialTriggerInput, sharedState)

	var userObject *users.UserRecord
	var sug suggestions.Suggestions
	lastInput := time.Now()
	c := configs.GetConfig()

	for {

		clientInput.EnterPressed = false // Default state is always false
		clientInput.TabPressed = false   // Default state is always false
		clientInput.BSPressed = false    // Default state is always false

		n, err := connDetails.Read(inputBuffer)
		if err != nil {

			// If failed to read from the connection, switch to zombie state
			if userObject != nil {

				userObject.EventLog.Add(`conn`, `Disconnected`)

				if c.Network.ZombieSeconds > 0 {

					connDetails.SetState(connections.Zombie)
					worldManager.SendSetZombie(userObject.UserId, true)

				} else {

					worldManager.SendLeaveWorld(userObject.UserId)
					worldManager.SendLogoutConnectionId(connDetails.ConnectionId())

				}

			}

			mudlog.Warn("Telnet", "connectionID", connDetails.ConnectionId(), "error", err)

			connections.Remove(connDetails.ConnectionId())

			break
		}

		if connDetails.InputDisabled() {
			continue
		}

		clientInput.DataIn = inputBuffer[:n]
		// Input handler processes any special commands, transforms input, sets flags from input, etc
		okContinue, lastHandlerName, err := connDetails.HandleInput(clientInput, sharedState)
		// Was there an error? If so, we should probably just stop processing input
		if err != nil {
			mudlog.Warn("InputHandler Error", "handler", lastHandlerName, "error", err)
			// Decide if disconnect is needed based on error type
			continue
		}

		// If a handler aborted processing, just keep track of where we are so
		// far and jump back to waiting.
		if !okContinue {

			// if no user signed in, loop back
			if userObject == nil {
				continue
			}

			_, suggested := userObject.GetUnsentText()

			redrawPrompt := false

			if clientInput.TabPressed {

				if sug.Count() < 1 {
					sug.Set(worldManager.GetAutoComplete(userObject.UserId, string(clientInput.Buffer)))
				}

				if sug.Count() > 0 {
					suggested = sug.Next()
					userObject.SetUnsentText(string(clientInput.Buffer), suggested)
					redrawPrompt = true
				}

			} else if clientInput.BSPressed {
				// If a suggestion is pending, remove it
				// otherwise just do a normal backspace operation
				userObject.SetUnsentText(string(clientInput.Buffer), ``)
				if suggested != `` {
					suggested = ``
					sug.Clear()
					redrawPrompt = true
				}

			} else {

				if suggested != `` {

					// If they hit space, accept the suggestion
					if len(clientInput.Buffer) > 0 && clientInput.Buffer[len(clientInput.Buffer)-1] == term.ASCII_SPACE {
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)-1], []byte(suggested)...)
						clientInput.Buffer = append(clientInput.Buffer[0:len(clientInput.Buffer)], []byte(` `)...)
						redrawPrompt = true
						userObject.SetUnsentText(string(clientInput.Buffer), ``)
						sug.Clear()
					} else {
						suggested = ``
						sug.Clear()
						// Otherwise, just keep the suggestion
						userObject.SetUnsentText(string(clientInput.Buffer), suggested)
						redrawPrompt = true
					}
				}

				userObject.SetUnsentText(string(clientInput.Buffer), suggested)
			}

			if redrawPrompt {
				pTxt := userObject.GetCommandPrompt()
				if connections.IsWebsocket(clientInput.ConnectionId) {
					connections.SendTo([]byte(pTxt), clientInput.ConnectionId)
				} else {
					connections.SendTo([]byte(templates.AnsiParse(pTxt)), clientInput.ConnectionId)
				}
			}

			continue
		}

		// The prompt handler returns 'true' from its HandleInput func only when
		// the entire sequence is complete *and* successful (e.g., login or creation ok).
		// If it returns true, it means we should proceed to the logged-in state.
		if okContinue && lastHandlerName == "LoginPromptHandler" {

			// Prompt sequence finished successfully

			// Stop intro music if playing
			connections.SendTo(
				term.MspCommand.BytesWithPayload([]byte("!!MUSIC(Off)")),
				clientInput.ConnectionId,
			)

			// Retrieve the UserObject stored by the completion function
			if uo, exists := sharedState["UserObject"]; exists {
				userObject = uo.(*users.UserRecord)
				// Remove it from shared state if no longer needed there
				delete(sharedState, "UserObject")
			} else {
				// This shouldn't happen if the completion function worked correctly
				mudlog.Error("Login process completed but UserObject not found in sharedState", "connectionId", clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId) // Disconnect problematic connection
				break                                        // Exit the read loop for this connection
			}

			// Remove the prompt handler (it signaled completion by returning true)
			connDetails.RemoveInputHandler("LoginPromptHandler")
			// Replace it with a regular echo handler.
			connDetails.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
			// Add admin command handler
			connDetails.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler) // Put history tracking after login handling, since login handling aborts input until complete

			if userObject.Role == users.RoleAdmin {
				connDetails.AddInputHandler("SystemCommandInputHandler", inputhandlers.SystemCommandInputHandler)
			}

			// Add a signal handler (shortcut ctrl combos) after the AnsiHandler
			// This captures signals and replaces user input so should happen after AnsiHandler to ensure it happens before other processes.
			connDetails.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")

			connDetails.SetState(connections.LoggedIn)

			worldManager.SendEnterWorld(userObject.UserId, userObject.Character.RoomId)

			clientInput.Reset()
			continue
		}

		// If okContinue is false OR the last handler was *not* the prompt handler,
		// it means either an error occurred (handled above), a handler aborted (like IAC/ANSI),
		// or the prompt handler is still waiting for input for the current/next step.
		// The existing logic for handling Tab/Backspace suggestions AFTER the input handlers run
		// might need slight adjustment depending on exactly how you want suggestions during prompts.
		// For simplicity, you might disable suggestions during the prompt sequence.
		if !okContinue {
			if userObject == nil {
				continue
			}
		}

		// If they have pressed enter (submitted their input), and nothing else has handled/aborted
		if clientInput.EnterPressed {

			// Update config after enter presses
			// No need to update it every loop
			c = configs.GetConfig()

			if time.Since(lastInput) < time.Duration(c.Timing.TurnMs)*time.Millisecond {
				/*
					connections.SendTo(
						[]byte("Slow down! You're typing too fast! "+time.Since(lastInput).String()+"\n"),
						connDetails.ConnectionId(),
					)
				*/

				// Reset the buffer for future commands.
				clientInput.Reset()

				// Capturing and resetting the unsent text is purely to allow us to
				// Keep updating the prompt without losing the typed in text.
				userObject.SetUnsentText(``, ``)

			} else {

				_, suggested := userObject.GetUnsentText()

				if len(suggested) > 0 {
					// solidify it in the render for UX reasons

					clientInput.Buffer = append(clientInput.Buffer, []byte(suggested)...)
					sug.Clear()
					userObject.SetUnsentText(string(clientInput.Buffer), ``)

					if connections.IsWebsocket(clientInput.ConnectionId) {
						connections.SendTo([]byte(userObject.GetCommandPrompt()), clientInput.ConnectionId)
					} else {
						connections.SendTo([]byte(templates.AnsiParse(userObject.GetCommandPrompt())), clientInput.ConnectionId)
					}

				}

				wi := WorldInput{
					FromId:    userObject.UserId,
					InputText: string(clientInput.Buffer),
				}

				// Buffer should be processed as an in-game command
				worldManager.SendInput(wi)
				// Reset the buffer for future commands.
				clientInput.Reset()

				// Capturing and resetting the unsent text is purely to allow us to
				// Keep updating the prompt without losing the typed in text.
				userObject.SetUnsentText(``, ``)

				lastInput = time.Now()
			}

			time.Sleep(time.Duration(10) * time.Millisecond)
			//	time.Sleep(time.Duration(util.TurnMs) * time.Millisecond)
		}

	}

}

// handleRestoredConnection handles connections restored from copyover (already logged in)
func handleRestoredConnection(connDetails *connections.ConnectionDetails, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	mudlog.Info("Restored Connection", "connectionID", connDetails.ConnectionId(), "remoteAddr", connDetails.RemoteAddr().String())

	// The connection already has input handlers set up from setupRecoveredConnectionHandlers()
	// and the user is already logged in and attached to this connection.
	// We just need to read input and process it.

	var sharedState map[string]any = make(map[string]any)

	// Get the user associated with this connection
	userObject := users.GetByConnectionId(connDetails.ConnectionId())
	if userObject == nil {
		mudlog.Error("Restored connection has no user", "connectionId", connDetails.ConnectionId())
		connections.Remove(connDetails.ConnectionId())
		return
	}

	// Send them their prompt
	if connections.IsWebsocket(connDetails.ConnectionId()) {
		connections.SendTo([]byte(userObject.GetCommandPrompt()), connDetails.ConnectionId())
	} else {
		connections.SendTo([]byte(templates.AnsiParse(userObject.GetCommandPrompt())), connDetails.ConnectionId())
	}

	// Input buffer for this connection
	var clientInput connections.ClientInput
	clientInput.ConnectionId = connDetails.ConnectionId()
	readBuffer := make([]byte, connections.ReadBufferSize)
	var sug suggestions.Suggestions

	// Main input loop - similar to the logged-in portion of handleTelnetConnection
	for {
		n, err := connDetails.Read(readBuffer)
		if err != nil {
			mudlog.Warn("Telnet", "connectionID", connDetails.ConnectionId(), "error", err)
			connections.Remove(connDetails.ConnectionId())
			return
		}

		clientInput.DataIn = readBuffer[:n]

		// Let input handlers process the data
		runNextHandler, lastHandler, err := connDetails.HandleInput(&clientInput, sharedState)
		if err != nil {
			mudlog.Error("Input Handler Error", "handler", lastHandler, "connectionID", clientInput.ConnectionId, "error", err)
			connections.Remove(connDetails.ConnectionId())
			return
		}

		if !runNextHandler {
			continue
		}

		// Check if user is still connected
		userObject = users.GetByConnectionId(clientInput.ConnectionId)
		if userObject == nil {
			connections.Remove(clientInput.ConnectionId)
			return
		}

		// Handle the input
		if clientInput.EnterPressed {
			// Process the command
			if len(clientInput.Buffer) == 0 {
				// Just resend the prompt
				if connections.IsWebsocket(clientInput.ConnectionId) {
					connections.SendTo([]byte(userObject.GetCommandPrompt()), clientInput.ConnectionId)
				} else {
					connections.SendTo([]byte(templates.AnsiParse(userObject.GetCommandPrompt())), clientInput.ConnectionId)
				}
			} else {
				// Send command to world
				wi := WorldInput{
					FromId:    userObject.UserId,
					InputText: string(clientInput.Buffer),
				}
				worldManager.SendInput(wi)
				clientInput.Reset()
			}
		} else {
			// Handle suggestions like in the original code
			_, suggested := userObject.GetUnsentText()
			if len(suggested) > 0 {
				clientInput.Buffer = append(clientInput.Buffer, []byte(suggested)...)
				sug.Clear()
				userObject.SetUnsentText(string(clientInput.Buffer), ``)

				if connections.IsWebsocket(clientInput.ConnectionId) {
					connections.SendTo([]byte(userObject.GetCommandPrompt()), clientInput.ConnectionId)
				} else {
					connections.SendTo([]byte(templates.AnsiParse(userObject.GetCommandPrompt())), clientInput.ConnectionId)
				}
			}
		}
	}
}

func HandleWebSocketConnection(conn *websocket.Conn) {

	var userObject *users.UserRecord
	connDetails := connections.Add(nil, conn)

	// Setup shared state map for this connection's handlers
	// Needs to be created BEFORE the first handler call
	var sharedState map[string]any = make(map[string]any)

	loginHandler := inputhandlers.GetLoginPromptHandler()           // Get the configured handler func
	connDetails.AddInputHandler("LoginPromptHandler", loginHandler) // Add it with a unique name

	// Describes whatever the client sent us
	clientInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		DataIn:       []byte{},
		Buffer:       make([]byte, 0, connections.ReadBufferSize), // DataIn is appended to this buffer after processing
		EnterPressed: false,
		Clipboard:    []byte{},
		History:      connections.InputHistory{},
	}

	connections.SendTo(
		[]byte("!!SOUND(Off U="+configs.GetConfig().FilePaths.WebCDNLocation.String()+")"),
		clientInput.ConnectionId,
	)

	plugins.OnNetConnect(connDetails)

	if audioConfig := audio.GetFile(`intro`); audioConfig.FilePath != `` {
		v := 100
		if audioConfig.Volume > 0 && audioConfig.Volume <= 100 {
			v = audioConfig.Volume
		}
		connections.SendTo(
			[]byte("!!MUSIC("+audioConfig.FilePath+" V="+strconv.Itoa(v)+" L=-1 C=1)"),
			clientInput.ConnectionId,
		)
	}

	// --- Send Initial Welcome/Splash ---
	// (This part was mostly correct before)
	splashTxt, _ := templates.Process("login/connect-splash", nil)
	connections.SendTo([]byte(templates.AnsiParse(splashTxt)), connDetails.ConnectionId())

	// --- Trigger the Prompt Handler to initialize state and send the FIRST prompt ---
	// Create a dummy input that signifies "start the process" but has no actual user data/control codes.
	initialTriggerInput := &connections.ClientInput{
		ConnectionId: connDetails.ConnectionId(),
		// Ensure flags like EnterPressed are false
	}
	// Call the handler function directly ONCE.
	// This executes the `!ok` block inside the handler, which:
	// 1. Creates the PromptHandlerState in sharedState.
	// 2. Calls advanceAndSendPromptCustom -> sendPromptFunc for the *first* step (username).
	// 3. Returns false (which we ignore here, as we aren't in the main loop yet).
	loginHandler(initialTriggerInput, sharedState)

	c := configs.GetConfig()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {

			// If failed to read from the connection, switch to zombie state
			if userObject != nil {

				userObject.EventLog.Add(`conn`, `Disconnected`)

				if c.Network.ZombieSeconds > 0 {

					connDetails.SetState(connections.Zombie)
					worldManager.SendSetZombie(userObject.UserId, true)

				} else {

					worldManager.SendLeaveWorld(userObject.UserId)
					worldManager.SendLogoutConnectionId(connDetails.ConnectionId())

				}

			}

			mudlog.Warn("WS Read", "error", err)
			break
		}

		clientInput.DataIn = message
		clientInput.Buffer = message
		clientInput.EnterPressed = true

		// Input handler processes any special commands, transforms input, sets flags from input, etc
		okContinue, lastHandlerName, err := connDetails.HandleInput(clientInput, sharedState)
		// Was there an error? If so, we should probably just stop processing input
		if err != nil {
			mudlog.Warn("InputHandler Error", "handler", lastHandlerName, "error", err)
			// Decide if disconnect is needed based on error type
			continue
		}

		// If okContinue is false OR the last handler was *not* the prompt handler,
		// it means either an error occurred (handled above), a handler aborted (like IAC/ANSI),
		// or the prompt handler is still waiting for input for the current/next step.
		// The existing logic for handling Tab/Backspace suggestions AFTER the input handlers run
		// might need slight adjustment depending on exactly how you want suggestions during prompts.
		// For simplicity, you might disable suggestions during the prompt sequence.
		if !okContinue {
			continue
		}

		// The prompt handler returns 'true' from its HandleInput func only when
		// the entire sequence is complete *and* successful (e.g., login or creation ok).
		// If it returns true, it means we should proceed to the logged-in state.
		if okContinue && lastHandlerName == "LoginPromptHandler" {

			// Prompt sequence finished successfully

			// Make sure web client text masking is off

			events.AddToQueue(events.WebClientCommand{
				ConnectionId: clientInput.ConnectionId,
				Text:         `TEXTMASK:false`,
			})

			// Stop intro music if playing
			connections.SendTo(
				[]byte("!!MUSIC(Off)"),
				clientInput.ConnectionId,
			)

			// Retrieve the UserObject stored by the completion function
			if uo, exists := sharedState["UserObject"]; exists {
				userObject = uo.(*users.UserRecord)
				// Remove it from shared state if no longer needed there
				delete(sharedState, "UserObject")
			} else {
				// This shouldn't happen if the completion function worked correctly
				mudlog.Error("Login process completed but UserObject not found in sharedState", "connectionId", clientInput.ConnectionId)
				connections.Remove(clientInput.ConnectionId) // Disconnect problematic connection
				break                                        // Exit the read loop for this connection
			}

			// Remove the prompt handler (it signaled completion by returning true)
			connDetails.RemoveInputHandler("LoginPromptHandler")
			// Replace it with a regular echo handler.
			connDetails.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
			// Add admin command handler
			connDetails.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler) // Put history tracking after login handling, since login handling aborts input until complete

			if userObject.Role == users.RoleAdmin {
				connDetails.AddInputHandler("SystemCommandInputHandler", inputhandlers.SystemCommandInputHandler)
			}

			// Add a signal handler (shortcut ctrl combos) after the AnsiHandler
			// This captures signals and replaces user input so should happen after AnsiHandler to ensure it happens before other processes.
			connDetails.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")

			connDetails.SetState(connections.LoggedIn)

			worldManager.SendEnterWorld(userObject.UserId, userObject.Character.RoomId)

			clientInput.Reset()
			continue
		}

		wi := WorldInput{
			FromId:    userObject.UserId,
			InputText: string(message),
		}

		// Buffer should be processed as an in-game command
		worldManager.SendInput(wi)

		c = configs.GetConfig()
	}
}

// setupRecoveredConnectionHandlers sets up input handlers for connections restored from copyover
func setupRecoveredConnectionHandlers() {
	// Get all active connections
	for _, connId := range connections.GetAllConnectionIds() {
		cd := connections.Get(connId)
		if cd == nil {
			continue
		}

		// Only set up handlers for logged-in connections
		if cd.State() == connections.LoggedIn {
			// Add the standard telnet input handlers
			cd.AddInputHandler("TelnetIACHandler", inputhandlers.TelnetIACHandler)
			cd.AddInputHandler("AnsiHandler", inputhandlers.AnsiHandler)
			cd.AddInputHandler("CleanserInputHandler", inputhandlers.CleanserInputHandler)
			cd.AddInputHandler("EchoInputHandler", inputhandlers.EchoInputHandler)
			cd.AddInputHandler("HistoryInputHandler", inputhandlers.HistoryInputHandler)
			cd.AddInputHandler("SignalHandler", inputhandlers.SignalHandler, "AnsiHandler")

			mudlog.Info("Copyover", "setup", "Added input handlers", "connId", connId)

			// Start the input reading goroutine for this restored connection
			// Use a special handler that skips login for already logged-in connections
			wg.Add(1)
			go handleRestoredConnection(cd, &wg)
			mudlog.Info("Copyover", "setup", "Started input goroutine for restored connection", "connId", connId)
		}
	}

	// Restore users to their rooms after all connections are set up
	restoreUsersToRooms()
}

// restoreUsersToRooms adds users back to their room's player lists after copyover
func restoreUsersToRooms() {
	// Get all active users
	for _, user := range users.GetAllActiveUsers() {
		if user == nil || user.Character.RoomId <= 0 {
			continue
		}

		// Use MoveToRoom with the same room ID to properly restore room tracking
		// This will add the user to the room's player list and update roomManager.roomsWithUsers
		rooms.MoveToRoom(user.UserId, user.Character.RoomId, false)
		mudlog.Info("Copyover", "restore", "Restored user to room", "userId", user.UserId, "username", user.Username, "roomId", user.Character.RoomId)
	}
}

func TelnetListenOnPort(hostname string, portNum int, wg *sync.WaitGroup, maxConnections int) net.Listener {

	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", hostname, portNum))
	if err != nil {
		mudlog.Error("Error creating server", "error", err)
		return nil
	}

	// Start accepting connections on the listener
	go AcceptOnListener(server, wg, maxConnections)

	mudlog.Info("Telnet", "stage", "Listening", "address", server.Addr())

	return server
}

// AcceptOnListener handles accepting connections on a listener (used for both new and preserved listeners)
func AcceptOnListener(server net.Listener, wg *sync.WaitGroup, maxConnections int) {
	// Loop to accept connections
	for {
		conn, err := server.Accept()

		if !serverAlive.Load() {
			mudlog.Warn("Connections disabled.")
			return
		}

		if err != nil {
			mudlog.Warn("Connection error", "error", err)
			continue
		}

		if maxConnections > 0 {
			if connections.ActiveConnectionCount() >= maxConnections {
				conn.Write([]byte(fmt.Sprintf("\n\n\n!!! Server is full (%d connections). Try again later. !!!\n\n\n", connections.ActiveConnectionCount())))
				conn.Close()
				continue
			}
		}

		wg.Add(1)
		// hand off the connection to a handler goroutine so that we can continue handling new connections
		go handleTelnetConnection(
			connections.Add(conn, nil),
			wg,
		)
	}
}

func loadAllDataFiles(isReload bool) {

	if isReload {

		defer func() {
			if r := recover(); r != nil {
				mudlog.Error("RELOAD FAILED", "err", r)
			}
		}()

	}

	// Force clear all cached VM's
	scripting.PruneVMs(true)

	// Load biomes before rooms since rooms reference biomes
	rooms.LoadBiomeDataFiles()
	spells.LoadSpellFiles()
	rooms.LoadDataFiles()
	buffs.LoadDataFiles() // Load buffs before items for cost calculation reasons
	items.LoadDataFiles()
	races.LoadDataFiles()
	mobs.LoadDataFiles()
	pets.LoadDataFiles()
	quests.LoadDataFiles()
	templates.LoadAliases(plugins.GetPluginRegistry())
	keywords.LoadAliases(plugins.GetPluginRegistry())
	mutators.LoadDataFiles()
	colorpatterns.LoadColorPatterns()
	audio.LoadAudioConfig()
	characters.CompileAdjectiveSwaps() // This should come after loading color patterns.
}
