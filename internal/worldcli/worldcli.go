package worldcli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func CreateWorld() {
	fmt.Println("GoMud World Creator")
	fmt.Println("==================")
	fmt.Println()
	fmt.Println("This will create a new world based on the 'empty' template and")
	fmt.Println("update your configuration to use the new world.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the name for your new world: ")
	worldName, err := reader.ReadString('\n')
	if err != nil {
		mudlog.Error("World Creation", "error", "Failed to read input: "+err.Error())
		return
	}

	worldName = strings.TrimSpace(worldName)
	if worldName == "" {
		mudlog.Error("World Creation", "error", "World name cannot be empty")
		return
	}

	folderName := sanitizeWorldName(worldName)
	if folderName == "" {
		mudlog.Error("World Creation", "error", "Invalid world name (reserved names: 'default', 'empty')")
		return
	}

	worldPath := filepath.Join("_datafiles", "world", folderName)
	if _, err := os.Stat(worldPath); err == nil {
		mudlog.Error("World Creation", "error", "World folder already exists: "+worldPath)
		return
	}

	fmt.Println()
	fmt.Printf("World Name: %s\n", worldName)
	fmt.Printf("Folder Name: %s\n", folderName)
	fmt.Printf("World Path: %s\n", worldPath)
	fmt.Println()
	fmt.Print("Create this world? (y/N): ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		mudlog.Error("World Creation", "error", "Failed to read confirmation: "+err.Error())
		return
	}

	confirm = strings.ToLower(strings.TrimSpace(confirm))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("World creation cancelled.")
		return
	}

	fmt.Printf("\nCreating world '%s'...\n", worldName)

	emptyWorldPath := filepath.Join("_datafiles", "world", "empty")
	if _, err := os.Stat(emptyWorldPath); err != nil {
		mudlog.Error("World Creation", "error", "Empty world template not found: "+emptyWorldPath)
		return
	}

	if err := copyDir(emptyWorldPath, worldPath); err != nil {
		mudlog.Error("World Creation", "error", "Failed to copy world template: "+err.Error())
		return
	}

	if err := updateConfigForNewWorld(folderName); err != nil {
		mudlog.Error("World Creation", "error", "Failed to update config: "+err.Error())
		os.RemoveAll(worldPath)
		return
	}

	fmt.Println()
	fmt.Printf("✓ World '%s' created successfully!\n", worldName)
	fmt.Printf("✓ World files copied to: %s\n", worldPath)
	fmt.Printf("✓ Configuration updated to use new world\n")
	fmt.Println()
	fmt.Println("You can now start the server to begin building your world.")
	fmt.Println("Use the in-game building commands to create rooms, items, and NPCs.")
}

func ListWorlds() {
	fmt.Println("Available GoMud Worlds")
	fmt.Println("=====================")
	fmt.Println()

	worldsPath := filepath.Join("_datafiles", "world")
	entries, err := os.ReadDir(worldsPath)
	if err != nil {
		mudlog.Error("List Worlds", "error", "Failed to read worlds directory: "+err.Error())
		return
	}

	currentWorld := getCurrentWorldFromConfig()

	var worlds []string
	for _, entry := range entries {
		if entry.IsDir() {
			worlds = append(worlds, entry.Name())
		}
	}

	if len(worlds) == 0 {
		fmt.Println("No worlds found.")
		return
	}

	for _, world := range worlds {
		if world == currentWorld {
			fmt.Printf("* %s (currently active)\n", world)
		} else {
			fmt.Printf("  %s\n", world)
		}
	}

	fmt.Println()
	fmt.Printf("Total worlds: %d\n", len(worlds))
	fmt.Println()
	fmt.Println("To switch to a different world, update the DataFiles path in _datafiles/config.yaml")
	fmt.Println("or use the --create-world flag to create a new world.")
}

func sanitizeWorldName(name string) string {
	name = strings.ToLower(name)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	name = regexp.MustCompile(`_+`).ReplaceAllString(name, "_")
	
	if name == "" || name == "default" || name == "empty" {
		return ""
	}
	
	return name
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func updateConfigForNewWorld(worldFolderName string) error {
	configPath := filepath.Join("_datafiles", "config.yaml")
	
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	configText := string(configData)
	newWorldPath := fmt.Sprintf("_datafiles/world/%s", worldFolderName)

	dataFilesRegex := regexp.MustCompile(`(\s*DataFiles:\s*)_datafiles/world/[^\s]+`)
	configText = dataFilesRegex.ReplaceAllString(configText, "${1}"+newWorldPath)

	langPathRegex := regexp.MustCompile(`(\s*LanguagePaths:\s*\n\s*-\s*'_datafiles/localize'\s*\n\s*-\s*)'_datafiles/world/[^/]+/localize'`)
	configText = langPathRegex.ReplaceAllString(configText, "${1}'"+newWorldPath+"/localize'")

	if err := os.WriteFile(configPath, []byte(configText), 0644); err != nil {
		return fmt.Errorf("failed to write updated config: %w", err)
	}

	return nil
}

func getCurrentWorldFromConfig() string {
	configPath := filepath.Join("_datafiles", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "unknown"
	}

	dataFilesRegex := regexp.MustCompile(`DataFiles:\s*_datafiles/world/([^\s]+)`)
	matches := dataFilesRegex.FindStringSubmatch(string(configData))
	if len(matches) > 1 {
		return matches[1]
	}

	return "unknown"
}