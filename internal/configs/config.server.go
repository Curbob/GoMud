package configs

type Server struct {
	MudName         ConfigString      `yaml:"MudName"`         // Name of the MUD
	CurrentVersion  ConfigString      `yaml:"CurrentVersion"`  // Current version this mud has been updated to
	Seed            ConfigSecret      `yaml:"Seed"`            // Seed that may be used for generating content
	MaxCPUCores     ConfigInt         `yaml:"MaxCPUCores"`     // How many cores to allow for multi-core operations
	OnLoginCommands ConfigSliceString `yaml:"OnLoginCommands"` // Commands to run when a user logs in
	Motd            ConfigString      `yaml:"Motd"`            // Message of the day to display when a user logs in
	NextRoomId      ConfigInt         `yaml:"NextRoomId"`      // The next room id to use when creating a new room
	Locked          ConfigSliceString `yaml:"Locked"`          // List of locked config properties that cannot be changed without editing the file directly.
}

func (s *Server) Validate() {

	// Ignore MudName
	// Ignore OnLoginCommands
	// Ignore Motd
	// Ignore NextRoomId
	// Ignore Locked

	if s.Seed == `` {
		s.Seed = `Mud` // default
	}

	if s.MaxCPUCores < 0 {
		s.MaxCPUCores = 0 // default
	}

	if s.CurrentVersion == `` {
		s.CurrentVersion = `0.9.0` // If no version found, failover to a known version
	}

}

func GetServerConfig() Server {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Server
}
