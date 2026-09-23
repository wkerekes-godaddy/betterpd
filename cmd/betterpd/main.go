package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"betterpd/internal/config"
	"betterpd/internal/pd"
	"betterpd/internal/ui"
)

var version = "dev"

const usage = `betterpd - A better PagerDuty TUI dashboard

Usage:
  betterpd              Start the dashboard
  betterpd --config     Interactive setup (creates config file)
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

	app := ui.NewApp(client, scope, cfg.RefreshInterval.Duration, version)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
