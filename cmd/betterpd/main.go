package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"betterpd/internal/blame"
	"betterpd/internal/config"
	"betterpd/internal/pd"
	"betterpd/internal/ui"
	"betterpd/internal/updater"
)

var version = "dev"

const usage = `betterpd - A better PagerDuty TUI dashboard

Usage:
  betterpd              Start the dashboard
  betterpd blame        Show on-call schedule changes
  betterpd blame --sync Force a fresh sync before showing changes
  betterpd blame --days N  Show changes from the last N days (default 30)
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
		case "blame", "--blame":
			runBlame()
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
	var scheduleMap map[string]string
	if len(cfg.ScheduleNames) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		scheduleMap, err = client.FindScheduleIDs(ctx, cfg.ScheduleNames)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not look up schedules: %v\n", err)
		} else {
			for _, id := range scheduleMap {
				scheduleIDs = append(scheduleIDs, id)
			}
		}
	}

	app := ui.NewApp(ui.AppOptions{
		Client:             client,
		Scope:              scope,
		Interval:           cfg.RefreshInterval.Duration,
		Version:            version,
		ScheduleIDs:        scheduleIDs,
		ScheduleMap:        scheduleMap,
		BlameSyncInterval:  cfg.BlameSyncInterval.Duration,
		BlameLookAheadDays: cfg.BlameLookAheadDays,
	})
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runBlame() {
	fs := flag.NewFlagSet("blame", flag.ExitOnError)
	forceSync := fs.Bool("sync", false, "Force a fresh sync regardless of cooldown")
	days := fs.Int("days", 30, "Show changes from the last N days")
	showAll := fs.Bool("all", false, "Include past (already-elapsed) shifts")
	var showEveryone bool
	fs.BoolVar(&showEveryone, "everyone", false, "Show changes for all users, not just you")
	fs.BoolVar(&showEveryone, "team", false, "Show changes for all users, not just you")
	fs.Parse(os.Args[2:])

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

	if len(cfg.ScheduleNames) == 0 {
		fmt.Fprintln(os.Stderr, "No schedule_names configured. Run 'betterpd --config' or add schedule_names to config.toml.")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schedMap, err := client.FindScheduleIDs(ctx, cfg.ScheduleNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error looking up schedules: %v\n", err)
		os.Exit(1)
	}
	if len(schedMap) == 0 {
		fmt.Fprintln(os.Stderr, "No matching schedules found for configured schedule_names.")
		os.Exit(1)
	}

	store, err := blame.OpenDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	syncCtx, syncCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer syncCancel()

	result, err := blame.Sync(syncCtx, client, store, schedMap, cfg.BlameLookAheadDays, cfg.BlameSyncInterval.Duration, *forceSync)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Sync error: %v\n", err)
		os.Exit(1)
	}

	if result.IsInitial {
		blame.DisplayInitialSync(result.ScheduleCount, os.Stdout)
		return
	}

	since := time.Now().Add(-time.Duration(*days) * 24 * time.Hour)
	changes, err := store.ListChanges(since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing changes: %v\n", err)
		os.Exit(1)
	}

	opts := blame.DisplayOptions{
		FutureOnly: !*showAll,
	}
	if !showEveryone {
		opts.UserID = cfg.UserID
	}
	changes = blame.FilterChanges(changes, opts)

	blame.DisplayChanges(changes, os.Stdout)
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
