package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func main() {
	var username string
	var oldName string
	var newName string
	var backupDir string
	var dryRun bool

	flag.StringVar(&username, "username", "", "Login username that owns the character to rename")
	flag.StringVar(&oldName, "old-name", "", "Existing character/display name to rename (main or alt)")
	flag.StringVar(&newName, "new-name", "", "New character/display name to set")
	flag.StringVar(&backupDir, "backup-dir", "", "Optional directory for backup copies (defaults next to the user YAML)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would happen without writing changes")
	flag.Parse()

	if username == "" || oldName == "" || newName == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/rename-character --username <login> --old-name <old> --new-name <new> [--backup-dir <dir>] [--dry-run]")
		os.Exit(2)
	}

	mudlog.SetupLogger(events.GetLogger(), "", "", false)

	if err := configs.ReloadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	idx := users.NewUserIndex()
	userID, found := idx.FindByUsername(username)
	if !found {
		fmt.Fprintf(os.Stderr, "user not found in index: %s\n", username)
		os.Exit(1)
	}

	if liveUser := users.GetByUserId(int(userID)); liveUser != nil {
		fmt.Fprintf(os.Stderr, "refusing to rename character for online user: %s (userId=%d)\n", liveUser.Username, liveUser.UserId)
		fmt.Fprintln(os.Stderr, "log the user out first, then run this tool again")
		os.Exit(1)
	}

	if strings.EqualFold(oldName, newName) {
		fmt.Fprintf(os.Stderr, "old and new character names are the same ignoring case: %s\n", oldName)
		os.Exit(1)
	}

	u, err := users.LoadUser(username, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load user: %v\n", err)
		os.Exit(1)
	}

	if ownerUserID, ownerUsername := users.CharacterNameSearch(newName); ownerUserID != 0 && !strings.EqualFold(ownerUsername, username) {
		fmt.Fprintf(os.Stderr, "new character name is already in use by %s (userId=%d)\n", ownerUsername, ownerUserID)
		os.Exit(1)
	}

	recordPath := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "users", fmt.Sprintf("%d.yaml", userID))
	altsPath := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "users", fmt.Sprintf("%d.alts.yaml", userID))
	backupBaseDir := backupDir
	if backupBaseDir == "" {
		backupBaseDir = filepath.Dir(recordPath)
	}
	timestamp := time.Now().Format("20060102-150405")
	recordBackupPath := filepath.Join(backupBaseDir, fmt.Sprintf("%d.yaml.bak.%s", userID, timestamp))
	altsBackupPath := filepath.Join(backupBaseDir, fmt.Sprintf("%d.alts.yaml.bak.%s", userID, timestamp))

	target := ""
	oldMainName := u.Character.Name
	oldAlts := characters.LoadAlts(u.UserId)
	updatedAlts := append([]characters.Character(nil), oldAlts...)
	altIndex := -1

	if strings.EqualFold(u.Character.Name, oldName) {
		target = "main"
		u.Character.Name = oldName
		if err := u.SetCharacterName(newName); err != nil {
			fmt.Fprintf(os.Stderr, "invalid new character name: %v\n", err)
			os.Exit(1)
		}
		u.Character.Name = oldMainName
	} else {
		for i := range updatedAlts {
			if strings.EqualFold(updatedAlts[i].Name, oldName) {
				target = "alt"
				altIndex = i
				break
			}
		}
		if target == "alt" {
			oldAltName := updatedAlts[altIndex].Name
			updatedAlts[altIndex].Name = ""
			if err := users.ValidateName(newName); err != nil {
				fmt.Fprintf(os.Stderr, "invalid new character name: %v\n", err)
				os.Exit(1)
			}
			updatedAlts[altIndex].Name = oldAltName
		}
	}

	if target == "" {
		fmt.Fprintf(os.Stderr, "character name %q not found for user %s\n", oldName, username)
		os.Exit(1)
	}

	fmt.Printf("userId: %d\nusername: %s\nold character name: %s\nnew character name: %s\ntarget: %s\nrecord: %s\nalts: %s\nrecord backup: %s\nalts backup: %s\n", userID, u.Username, oldName, newName, target, recordPath, altsPath, recordBackupPath, altsBackupPath)

	if dryRun {
		fmt.Printf("DRY RUN: would update %s character name from %q to %q\n", target, oldName, newName)
		if target == "alt" {
			fmt.Printf("DRY RUN: would update alt record in %s\n", altsPath)
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(recordBackupPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create backup directory: %v\n", err)
		os.Exit(1)
	}

	copyFile := func(srcPath, dstPath string) error {
		src, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, src, 0o600)
	}

	if err := copyFile(recordPath, recordBackupPath); err != nil {
		fmt.Fprintf(os.Stderr, "failed to back up user record: %v\n", err)
		os.Exit(1)
	}

	altsExisted := characters.AltsExists(u.UserId)
	if altsExisted {
		if err := copyFile(altsPath, altsBackupPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to back up alts record: %v\n", err)
			os.Exit(1)
		}
	}

	if target == "main" {
		if err := u.SetCharacterName(newName); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set main character name: %v\n", err)
			os.Exit(1)
		}
		if err := users.SaveUser(*u); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save renamed user: %v\n", err)
			os.Exit(1)
		}
	} else {
		updatedAlts[altIndex].Name = newName
		if ok := characters.SaveAlts(u.UserId, updatedAlts); !ok {
			fmt.Fprintf(os.Stderr, "failed to save renamed alt character\n")
			os.Exit(1)
		}
	}

	fmt.Printf("character rename complete: %s -> %s (userId=%d, target=%s)\n", oldName, newName, u.UserId, target)
	fmt.Printf("record: %s\nrecord backup: %s\n", recordPath, recordBackupPath)
	if target == "alt" || altsExisted {
		fmt.Printf("alts: %s\nalts backup: %s\n", altsPath, altsBackupPath)
	}
	fmt.Println("login username/index were not changed")
	fmt.Println("recommendation: restart GoMUD before allowing the user back online")
}
