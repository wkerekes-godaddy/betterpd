package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"betterpd/internal/config"
	"betterpd/internal/pd"
	"betterpd/internal/ui"
	"betterpd/internal/updater"
)

var version = "dev"

const usage = `betterpd - A better PagerDuty TUI dashboard

Usage:
  betterpd              Start the dashboard
  betterpd --config     Interactive setup (creates config file)
  betterpd --update     Update to the latest release
  betterpd --help       Show this help

Keybindings:
  j/k, ↑/↓             Navigate incidents
  space                 Toggle select (for batch actions)
  a                     Acknowledge selected/current incident(s)
  x                     Resolve selected/current incident(s)
  s                     Snooze (pick duration)
  o                     Open incident in browser
  enter                 View incident details
  esc                   Back / close overlay
  r                     Force refresh
  1/2/3                 Switch scope: mine / team / all
  ?                     Show keybindings help
  q, ctrl+c            Quit

Configuration:
  Config file: ~/.config/betterpd/config.toml
  Env var:     PAGERDUTY_TOKEN (overrides config file token)

  Run 'betterpd --config' to create the config file interactively.

  Config file format:
    token = "your-pagerduty-api-token"
    refresh_interval = "30s"
    default_scope = "mine"    # mine, team, or all
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "help":
			fmt.Print(usage)
			return
		case "--config", "--setup", "config", "setup":
			if err := config.RunSetup(); err != nil {
				fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
				os.Exit(1)
			}
			return
		case "--update", "update":
			selfUpdate()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\nRun 'betterpd --help' for usage.\n", os.Args[1])
			os.Exit(1)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	client, err := pd.NewClient(cfg.Token, cfg.UserID, cfg.UserEmail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	scope := pd.ScopeMine
	switch cfg.DefaultScope {
	case "team":
		scope = pd.ScopeTeam
	case "all":
		scope = pd.ScopeAll
	}

	var scheduleIDs []string
	if len(cfg.ScheduleNames) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		schedMap, err := client.FindScheduleIDs(ctx, cfg.ScheduleNames)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not look up schedules: %v\n", err)
		} else {
			for _, id := range schedMap {
				scheduleIDs = append(scheduleIDs, id)
			}
		}
	}

	app := ui.NewApp(client, scope, cfg.RefreshInterval.Duration, version, scheduleIDs)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func selfUpdate() {
	if version == "dev" {
		fmt.Fprintln(os.Stderr, "Cannot update a dev build. Install from a release binary.")
		os.Exit(1)
	}

	fmt.Printf("Checking for updates (current: %s)...\n", version)
	rel, err := updater.CheckLatest(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if !rel.Newer {
		fmt.Printf("Already up to date (%s).\n", version)
		return
	}

	if rel.AssetURL == "" {
		fmt.Fprintf(os.Stderr, "No release binary available for your platform.\n")
		os.Exit(1)
	}

	fmt.Printf("Downloading %s...\n", rel.Version)
	if err := updater.SelfUpdate(rel.AssetURL); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated betterpd to %s.\n", rel.Version)
}
