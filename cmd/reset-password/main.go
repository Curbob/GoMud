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
	var newPassword string
	var backupDir string
	var dryRun bool

	flag.StringVar(&username, "username", "", "Username to reset")
	flag.StringVar(&newPassword, "password", "", "New plaintext password to set")
	flag.StringVar(&backupDir, "backup-dir", "", "Optional directory for backup copies (defaults next to the user YAML)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would happen without writing changes")
	flag.Parse()

	if username == "" || newPassword == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/reset-password --username <name> --password <newpass> [--backup-dir <dir>] [--dry-run]")
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
		fmt.Fprintln(os.Stderr, "available indexed usernames are likely different; check _datafiles/.../users/users.idx or try the exact username field from the YAML record")
		os.Exit(1)
	}

	recordPath := filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "users", fmt.Sprintf("%d.yaml", userID))
	backupBaseDir := backupDir
	if backupBaseDir == "" {
		backupBaseDir = filepath.Dir(recordPath)
	}

	backupPath := filepath.Join(backupBaseDir, fmt.Sprintf("%d.yaml.bak.%s", userID, time.Now().Format("20060102-150405")))

	if dryRun {
		fmt.Printf("DRY RUN\nrecord: %s\nbackup: %s\n", recordPath, backupPath)
		return
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create backup directory: %v\n", err)
		os.Exit(1)
	}

	src, err := os.ReadFile(recordPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read user record: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(backupPath, src, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write backup file: %v\n", err)
		os.Exit(1)
	}

	u, err := users.LoadUser(username, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load user: %v\n", err)
		os.Exit(1)
	}

	if err := u.SetPassword(newPassword); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set password: %v\n", err)
		os.Exit(1)
	}

	if err := users.SaveUser(*u); err != nil {
		fmt.Fprintf(os.Stderr, "failed to save user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("password reset for %s\nrecord: %s\nbackup: %s\n", u.Username, recordPath, backupPath)
}
