package migration

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/version"
)

func Run(lastConfigVersion version.Version, serverVersion version.Version) (err error) {

	if lastConfigVersion.IsEqualTo(serverVersion) {
		return nil
	}

	//
	// Note: Follow this pattern and keep these version upgrades in order of lowest to greatest to avoid problems
	// Note2: Take care to not shadow the err variable, since it is used in defer
	//
	var backupFolder string

	// This defer checks whether an error is present before returning
	// If so, restores backup.
	defer func() {
		if err != nil && backupFolder != `` {
			fmt.Println("OOPS", err)
			copyDir(backupFolder, string(configs.GetFilePathsConfig().DataFiles))
		}
		os.RemoveAll(backupFolder)
	}()

	backupFolder, err = datafilesBackup()
	if err != nil {
		err = fmt.Errorf(`could not backup datafiles: %w`, err)
		return
	}

	// 0.0.0 -> 1.0.0
	if lastConfigVersion.IsOlderThan(version.New(1, 0, 0)) {

		err = migrate_RoomZoneConfig()
		if err != nil {
			return
		}

	}

	//
	// Finally, since successful, update to the version this migration is for
	//
	configs.SetVal(`Server.CurrentVersion`, serverVersion.String())

	return
}
