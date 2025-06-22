package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"gopkg.in/yaml.v2"
)

// Description:
// rooms.Room.ZoneConfig was removed when Zone data was migrated to zone-config.yaml in zone folders
// This function loads all of the yaml files in the DATAFILES/world/*/rooms/* and looks for any ZoneConfig data.
// If found, the data is moved to a zone-config.yaml file, and the ZoneConfig data in the Room datafile is removed.
func migrate_RoomZoneConfig() error {

	// This struct is how ZoneConfig looked as of 1.0.0
	// Since we will be upgrading an older version to this format, use a copy of the struct from that period
	// To ensure we aren't using a struct that has changed over time
	type zoneConfig_1_0_0 struct {
		Name         string `yaml:"name,omitempty"`
		RoomId       int    `yaml:"roomid,omitempty"`
		MobAutoScale struct {
			Minimum int `yaml:"minimum,omitempty"` // level scaling minimum
			Maximum int `yaml:"maximum,omitempty"` // level scaling maximum
		} `yaml:"autoscale,omitempty"` // level scaling range if any
		Mutators []struct {
			MutatorId      string `yaml:"mutatorid,omitempty"`      // Short text that will uniquely identify this modifier ("dusty")
			SpawnedRound   uint64 `yaml:"spawnedround,omitempty"`   // Tracks when this mutator was created (useful for decay)
			DespawnedRound uint64 `yaml:"despawnedround,omitempty"` // Track when it decayed to nothing.
		} `yaml:"mutators,omitempty"`
		IdleMessages []string         `yaml:"idlemessages,omitempty"` // list of messages that can be displayed to players in the zone, assuming a room has none defined
		MusicFile    string           `yaml:"musicfile,omitempty"`    // background music to play when in this zone
		DefaultBiome string           `yaml:"defaultbiome,omitempty"` // city, swamp etc. see biomes.go
		RoomIds      map[int]struct{} `yaml:"-"`                      // Does not get written. Built dyanmically when rooms are loaded.
	}

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

		mudlog.Info("Migration 1.0.0", "file", path, "message", "migrating zoneconfig from room data file to zone-config.yaml")

		//
		// From here on out, this code migrates zoneconfig data out of room file and into zone-config.yaml
		//
		roomFileInfo, _ := os.Stat(path)

		mudlog.Info("Migration 1.0.0", "file", path, "message", "isolating zoneconfig data")

		// Isolate the zoneconfig and write it to its own zone-config.yaml file
		zoneBytes, err := yaml.Marshal(filedata[`zoneconfig`])
		if err != nil {
			return err
		}

		zoneDataStruct := zoneConfig_1_0_0{}

		if err = yaml.Unmarshal(zoneBytes, &zoneDataStruct); err != nil {
			return err
		}

		if filedata[`zone`] != nil {
			if zoneName, ok := filedata[`zone`].(string); ok {
				zoneDataStruct.Name = zoneName
			} else {
				zoneDataStruct.Name = filedata[`title`].(string)
			}

			if defaultBiome, ok := filedata[`biome`].(string); ok {
				zoneDataStruct.DefaultBiome = defaultBiome
			}
		}

		zoneFileBytes, err := yaml.Marshal(zoneDataStruct)
		if err != nil {
			return err
		}

		zoneFilePath := filepath.Join(filepath.Dir(path), "zone-config.yaml")

		mudlog.Info("Migration 1.0.0", "file", path, "message", "writing "+zoneFilePath)

		if err := os.WriteFile(zoneFilePath, zoneFileBytes, roomFileInfo.Mode().Perm()); err != nil {
			return err
		}

		// Now clear "zoneconfig" and write the room data back
		filedata[`zoneconfig`] = nil

		mudlog.Info("Migration 1.0.0", "file", path, "message", "writing modified room data")

		// First marshal the modified room data into bytes
		modifiedRoomBytes, err := yaml.Marshal(filedata)
		if err != nil {
			return err
		}

		// Unmarshal the bytes into the proper struct
		modifiedRoomStruct := rooms.Room{}
		if err = yaml.Unmarshal(modifiedRoomBytes, &modifiedRoomStruct); err != nil {
			return err
		}

		// Marshal again, this time using the proper struct
		modifiedRoomBytes, err = yaml.Marshal(modifiedRoomStruct)
		if err != nil {
			return err
		}

		if err := os.WriteFile(path, modifiedRoomBytes, roomFileInfo.Mode().Perm()); err != nil {
			return err
		}

		mudlog.Info("Migration 1.0.0", "file", path, "message", "successfully updated")

	}

	return nil
}
