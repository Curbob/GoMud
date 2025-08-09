package configs

import (
	"strings"
)

type UserInterface struct {
	Formats UserInterfaceFormats `yaml:"Formats"`
	Display UserInterfaceDisplay `yaml:"Display"`
}

type UserInterfaceFormats struct {
	Prompt                  ConfigString `yaml:"Prompt"`                  // The in-game status prompt style
	EnterRoomMessageWrapper ConfigString `yaml:"EnterRoomMessageWrapper"` // Special enter messages
	ExitRoomMessageWrapper  ConfigString `yaml:"ExitRoomMessageWrapper"`  // Special exit messages
	Time                    ConfigString `yaml:"Time"`                    // How to format time when displaying real time
	TimeShort               ConfigString `yaml:"TimeShort"`               // How to format time when displaying real time (shortform)
}

type UserInterfaceDisplay struct {
	ShowEmptyEquipmentSlots ConfigBool `yaml:"ShowEmptyEquipmentSlots"` // Whether to show empty equipment slots when looking at characters/mobs
}

func (u *UserInterface) Validate() {
	u.Formats.Validate()
	u.Display.Validate()
}

func (f *UserInterfaceFormats) Validate() {
	if f.Prompt == `` {
		f.Prompt = `{8}[{t} {T} {255}HP:{hp}{8}/{HP} {255}MP:{13}{mp}{8}/{13}{MP}{8}]{239}{h}{8}:`
	}

	// Must have a message wrapper...
	if f.EnterRoomMessageWrapper == `` {
		f.EnterRoomMessageWrapper = `%s` // default
	}
	if strings.LastIndex(string(f.EnterRoomMessageWrapper), `%s`) < 0 {
		f.EnterRoomMessageWrapper += `%s` // append if missing
	}

	// Must have a message wrapper...
	if f.ExitRoomMessageWrapper == `` {
		f.ExitRoomMessageWrapper = `%s` // default
	}
	if strings.LastIndex(string(f.ExitRoomMessageWrapper), `%s`) < 0 {
		f.ExitRoomMessageWrapper += `%s` // append if missing
	}

	if f.Time == `` {
		f.Time = `Monday, 02-Jan-2006 03:04:05PM`
	}

	if f.TimeShort == `` {
		f.TimeShort = `Jan 2 '06 3:04PM`
	}
}

func (d *UserInterfaceDisplay) Validate() {
	// ShowEmptyEquipmentSlots defaults to true (show all slots)
	// The ConfigBool type handles the default value
}

// Convenience method to check if empty equipment slots should be shown
func (c Config) ShouldShowEmptyEquipmentSlots() bool {
	return bool(c.UserInterface.Display.ShowEmptyEquipmentSlots)
}

// GetUserInterfaceConfig returns the UserInterface configuration
func GetUserInterfaceConfig() UserInterface {
	return GetConfig().UserInterface
}
