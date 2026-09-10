package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Token           string   `toml:"token"`
	UserID          string   `toml:"user_id"`
	UserEmail       string   `toml:"user_email"`
	RefreshInterval Duration `toml:"refresh_interval"`
	DefaultScope    string   `toml:"default_scope"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	s := normalizeDuration(string(text))
	var err error
	d.Duration, err = time.ParseDuration(s)
	return err
}

func normalizeDuration(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	last := s[len(s)-1]
	if last >= '0' && last <= '9' {
		return s + "s"
	}
	return s
}

func Load() (*Config, error) {
	cfg := &Config{
		RefreshInterval: Duration{30 * time.Second},
		DefaultScope:    "mine",
	}

	configPath := filepath.Join(ConfigDir(), "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		if _, err := toml.DecodeFile(configPath, cfg); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
	}

	if token := os.Getenv("PAGERDUTY_TOKEN"); token != "" {
		cfg.Token = token
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("no PagerDuty token found.\n\nRun 'betterpd --config' to configure, or set PAGERDUTY_TOKEN env var")
	}

	return cfg, nil
}

func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "betterpd")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "betterpd")
}

func RunSetup() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("betterpd setup")
	fmt.Println("==============")
	fmt.Println()
	fmt.Println("You'll need a PagerDuty API token.")
	fmt.Println("Create one at: https://support.pagerduty.com/main/docs/api-access-keys#generate-a-user-token-rest-api-key")
	fmt.Println()

	fmt.Print("PagerDuty API token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	fmt.Println()
	fmt.Println("Your PagerDuty user ID (found in your profile URL, e.g. https://yourco.pagerduty.com/users/XXXXXXX/profile)")
	fmt.Print("User ID: ")
	userID, _ := reader.ReadString('\n')
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	fmt.Println()
	fmt.Println("Default incident scope:")
	fmt.Println("  1) mine  - only incidents assigned to you")
	fmt.Println("  2) team  - incidents for your teams")
	fmt.Println("  3) all   - all incidents")
	fmt.Print("Choice [1]: ")
	scopeInput, _ := reader.ReadString('\n')
	scopeInput = strings.TrimSpace(scopeInput)

	scope := "mine"
	switch scopeInput {
	case "2", "team":
		scope = "team"
	case "3", "all":
		scope = "all"
	}

	fmt.Print("Refresh interval in seconds [30]: ")
	intervalInput, _ := reader.ReadString('\n')
	intervalInput = strings.TrimSpace(intervalInput)
	if intervalInput == "" {
		intervalInput = "30s"
	} else {
		intervalInput = normalizeDuration(intervalInput)
	}
	if _, err := time.ParseDuration(intervalInput); err != nil {
		return fmt.Errorf("invalid duration %q: %w", intervalInput, err)
	}

	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	configPath := filepath.Join(dir, "config.toml")
	content := fmt.Sprintf(`# betterpd configuration
# See: betterpd --help

# PagerDuty API token
token = %q

# Your PagerDuty user ID (from your profile URL)
user_id = %q

# How often to poll for updates (number = seconds, or use Go duration: 30s, 1m, etc.)
refresh_interval = %q

# Default scope: "mine", "team", or "all"
default_scope = %q
`, token, userID, intervalInput, scope)

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Println()
	fmt.Printf("Config saved to %s\n", configPath)
	fmt.Println("Run 'betterpd' to start the dashboard.")
	return nil
}
