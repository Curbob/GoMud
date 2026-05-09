package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func main() {
	var includeAlts bool

	flag.BoolVar(&includeAlts, "include-alts", false, "Also show alt character names for each account")
	flag.Parse()

	mudlog.SetupLogger(events.GetLogger(), "", "", false)

	if err := configs.ReloadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	idx := users.NewUserIndex()
	records, err := idx.ListRecords()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read user index: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("USERID\tLOGIN_USERNAME\tCHARACTER_NAME")
	for _, rec := range records {
		username := rec.UsernameString()
		u, err := users.LoadUser(username, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load user %s (userId=%d): %v\n", username, rec.UserID, err)
			continue
		}

		characterName := u.Character.Name
		if includeAlts {
			altNames := make([]string, 0)
			for _, alt := range characters.LoadAlts(u.UserId) {
				altNames = append(altNames, alt.Name)
			}
			if len(altNames) > 0 {
				characterName = fmt.Sprintf("%s [alts: %s]", characterName, strings.Join(altNames, ", "))
			}
		}
		fmt.Printf("%d\t%s\t%s\n", rec.UserID, u.Username, characterName)
	}
}
