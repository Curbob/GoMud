package migration

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/version"
)

func Run(lastConfigVersion version.Version, serverVersion version.Version) error {

	//
	// Note: Follow this pattern and keep these version upgrades in order of lowest to greatest to avoid problems
	//

	// 0.0.0 -> 1.0.0
	if lastConfigVersion.IsOlderThan(version.New(1, 0, 0)) {

		if err := migrate_RoomZoneConfig(); err != nil {
			return err
		}

	}

	//
	// Finally, update to the version this migration is for
	//
	configs.SetVal(`Server.CurrentVersion`, serverVersion.String())

	return nil
}
