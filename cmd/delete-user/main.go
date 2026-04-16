package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func main() {
	var username string
	var archiveDir string
	var dryRun bool
	var hardDelete bool

	flag.StringVar(&username, "username", "", "Username to delete/archive")
	flag.StringVar(&archiveDir, "archive-dir", "", "Optional archive directory (defaults to <datafiles>/users/deleted)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would happen without writing changes")
	flag.BoolVar(&hardDelete, "hard-delete", false, "Permanently delete files instead of archiving them")
	flag.Parse()

	if username == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/delete-user --username <name> [--archive-dir <dir>] [--dry-run] [--hard-delete]")
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
		fmt.Fprintf(os.Stderr, "refusing to delete online user: %s (userId=%d)\n", liveUser.Username, liveUser.UserId)
		fmt.Fprintln(os.Stderr, "log the user out first, then run this tool again")
		os.Exit(1)
	}

	dataDir := configs.GetFilePathsConfig().DataFiles.String()
	recordPath := filepath.Join(dataDir, "users", fmt.Sprintf("%d.yaml", userID))
	altsPath := filepath.Join(dataDir, "users", fmt.Sprintf("%d.alts.yaml", userID))

	timestamp := time.Now().Format("20060102-150405")
	if archiveDir == "" {
		archiveDir = filepath.Join(dataDir, "users", "deleted")
	}
	archivedRecordPath := filepath.Join(archiveDir, fmt.Sprintf("%d.yaml.deleted.%s", userID, timestamp))
	archivedAltsPath := filepath.Join(archiveDir, fmt.Sprintf("%d.alts.yaml.deleted.%s", userID, timestamp))

	recordExists := fileExists(recordPath)
	altsExists := fileExists(altsPath)
	if !recordExists {
		fmt.Fprintf(os.Stderr, "user record file not found: %s\n", recordPath)
		os.Exit(1)
	}

	fmt.Printf("user: %s\nuserId: %d\nrecord: %s\nalts: %s (exists=%t)\nmode: %s\n",
		username,
		userID,
		recordPath,
		altsPath,
		altsExists,
		modeName(hardDelete),
	)

	if dryRun {
		if hardDelete {
			fmt.Printf("DRY RUN: would delete %s\n", recordPath)
			if altsExists {
				fmt.Printf("DRY RUN: would delete %s\n", altsPath)
			}
		} else {
			fmt.Printf("DRY RUN: would archive %s -> %s\n", recordPath, archivedRecordPath)
			if altsExists {
				fmt.Printf("DRY RUN: would archive %s -> %s\n", altsPath, archivedAltsPath)
			}
		}
		fmt.Printf("DRY RUN: would remove %q from users.idx\n", username)
		return
	}

	if !hardDelete {
		if err := os.MkdirAll(archiveDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create archive directory: %v\n", err)
			os.Exit(1)
		}

		if err := os.Rename(recordPath, archivedRecordPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to archive user record: %v\n", err)
			os.Exit(1)
		}

		if altsExists {
			if err := os.Rename(altsPath, archivedAltsPath); err != nil {
				fmt.Fprintf(os.Stderr, "failed to archive alts record: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		if err := os.Remove(recordPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to delete user record: %v\n", err)
			os.Exit(1)
		}
		if altsExists {
			if err := os.Remove(altsPath); err != nil {
				fmt.Fprintf(os.Stderr, "failed to delete alts record: %v\n", err)
				os.Exit(1)
			}
		}
	}

	if err := idx.RemoveByUsername(username); err != nil {
		fmt.Fprintf(os.Stderr, "warning: file action succeeded, but failed to remove user from index: %v\n", err)
		fmt.Fprintln(os.Stderr, "recommended recovery: rebuild users.idx before restarting the server")
		os.Exit(1)
	}

	fmt.Printf("%s complete for %s (userId=%d)\n", modeName(hardDelete), username, userID)
	fmt.Println("recommendation: restart GoMUD before allowing new logins")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func modeName(hardDelete bool) string {
	if hardDelete {
		return "delete"
	}
	return "archive"
}
