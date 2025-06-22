package migration

import (
	"github.com/GoMudEngine/GoMud/internal/version"
)

func Run(binVersion version.Version) error {

	//
	// Note: Follow this pattern and keep these version upgrades in order of lowest to greatest to avoid problems
	//

	// 0.0.0 -> 1.0.0
	if binVersion.IsOlderThan(version.New(1, 0, 0)) {

		if err := migrate_RoomZoneConfig(); err != nil {
			return err
		}

	}

	return nil
}
