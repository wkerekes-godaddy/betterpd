This was all written by Claude.  Just download the binary from the [Releases](https://github.com/wkerekes-godaddy/betterpd/releases/latest) page.  That's probably easier.

---

# betterpd

A terminal UI for PagerDuty. View, acknowledge, resolve, and ~~snooze~~* incidents without leaving your terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Install
Download the [latest release](https://github.com/wkerekes-godaddy/betterpd/releases/latest).

Optionally place the binary in a folder in your $PATH, or call it with the full path to the file.

Grant execute permission with `chmod +x ./betterpd`

<!--
```
go install ./cmd/betterpd
```

Or build locally:

```
make build        # outputs to bin/betterpd
```
-->
## Setup

Run the interactive setup to create your config file:

```
betterpd --config
```

This creates `~/.config/betterpd/config.toml`. You'll need:

- A PagerDuty [REST API user token](https://support.pagerduty.com/main/docs/api-access-keys#generate-a-user-token-rest-api-key)
- Your PagerDuty user ID (from your profile URL)

Alternatively, set `PAGERDUTY_TOKEN` as an environment variable and edit `config.toml.example` as a reference.

## Usage

```
betterpd              # start the dashboard
betterpd --config     # interactive setup
betterpd --help       # show help
```

## Keybindings

| Key | Action |
|-----|--------|
| `j`/`k`, `↑`/`↓` | Navigate incidents |
| `space` | Toggle select (for batch actions) |
| `a` | Acknowledge selected/current incident(s) |
| `x` | Resolve selected/current incident(s) |
| `s` | Snooze (pick duration) |
| `o` | Open incident in browser |
| `enter` | View incident details |
| `esc` | Back / close overlay |
| `r` | Force refresh |
| `1`/`2`/`3` | Switch scope: mine / team / all |
| `?` | Show keybindings help |
| `q`, `ctrl+c` | Quit |

## Configuration

```toml
# PagerDuty API token
token = ""

# Your PagerDuty user ID (from your profile URL)
user_id = ""

# Your PagerDuty email (used as 'From' header for ack/resolve)
#user_email = ""

# How often to poll for updates
refresh_interval = "30s"

# Default scope: "mine", "team", or "all"
default_scope = "mine"
```

* Snoozing an alert is questionable at best.  Sometimes it works, sometimes it doesn't.  This might be a limit of the API.
