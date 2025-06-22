package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// rooms.Room.ZoneConfig was removed when Zone data was migrated to zone-config.yaml in zone folders
// This function loads all of the yaml files in the DATAFILES/world/*/rooms/* and looks for any ZoneConfig data.
// If found, the data is moved to a zone-config.yaml file, and the ZoneConfig data in the Room datafile is removed.
func migrate_RoomZoneConfig() error {

	c := configs.GetConfig()

	worldfilesGlob := filepath.Join(string(c.FilePaths.DataFiles), "rooms", "*", "*.yaml")
	matches, err := filepath.Glob(worldfilesGlob)

	if err != nil {
		return err
	}

	// We only care about room files, so ###.yaml (possible negative)
	re := regexp.MustCompile(`^[\-0-9]+\.yaml$`)
	for _, path := range matches {

		filename := filepath.Base(path)
		if !re.MatchString(filename) {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		filedata := map[string]any{}

		err = yaml.Unmarshal(data, &filedata)
		if err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		if filedata[`zoneconfig`] == nil {
			continue
		}

		mudlog.Info("ZoneConfig found", "path", path)
		//fmt.Println(filedata[`zoneconfig`])

		filedata[`zoneconfig`] = nil

		fdata, _ := yaml.Marshal(filedata)

		info, _ := os.Stat(path)

		if err := os.WriteFile(path, fdata, info.Mode().Perm()); err != nil {
			return err
		}

	}

	return nil
}
