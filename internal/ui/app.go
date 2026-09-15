package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"betterpd/internal/pd"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	viewList view = iota
	viewDetail
	viewSnooze
	viewHelp
)

type incidentsFetchedMsg struct {
	incidents []pd.Incident
	err       error
}

type actionCompleteMsg struct {
	action string
	err    error

	// snoozedIDs/snoozedUntil are set only for a successful snooze action,
	// so the app can record which incidents the user actually snoozed.
	snoozedIDs   []string
	snoozedUntil time.Time
}

type tickMsg time.Time

type App struct {
	client   *pd.Client
	scope    pd.Scope
	interval time.Duration

	view     view
	prevView view
	list     incidentList
	detail   incidentDetail
	snooze   snoozePicker
	help     helpOverlay
	status   statusBar
	width    int
	height   int
	loading  bool
	err      error

	// snoozed tracks incidents the user has actually snoozed (via the 's'
	// action), keyed by incident ID, so the list can show "snoozed" only
	// for those — not for every acknowledged incident with a pending
	// auto-unacknowledge timeout.
	snoozed map[string]time.Time
}

func NewApp(client *pd.Client, scope pd.Scope, interval time.Duration) *App {
	return &App{
		client:   client,
		scope:    scope,
		interval: interval,
		list:     newIncidentList(),
		snooze:   newSnoozePicker(),
		status:   statusBar{orgName: client.OrgName()},
		loading:  true,
		snoozed:  make(map[string]time.Time),
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(a.fetchIncidents(), a.tick())
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.status.width = msg.Width
		a.list.setSize(msg.Width, msg.Height-4)
		a.detail.width = msg.Width
		a.detail.height = msg.Height - 4
		return a, nil

	case incidentsFetchedMsg:
		a.loading = false
		if msg.err != nil {
			a.err = msg.err
			a.status.setMessage(styleError.Render("Error: " + msg.err.Error()))
		} else {
			a.err = nil
			a.applySnoozeState(msg.incidents)
			a.list.setIncidents(msg.incidents)
			a.status.lastRefresh = time.Now()
			a.status.count = len(msg.incidents)
		}
		return a, nil

	case actionCompleteMsg:
		if msg.err != nil {
			a.status.setMessage(styleError.Render("Error: " + msg.err.Error()))
		} else {
			a.status.setMessage(styleSuccess.Render(msg.action))
			a.list.clearSelection()
			for _, id := range msg.snoozedIDs {
				a.snoozed[id] = msg.snoozedUntil
			}
		}
		return a, a.fetchIncidents()

	case tickMsg:
		return a, tea.Batch(a.fetchIncidents(), a.tick())

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	if a.view == viewList {
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch {
	case msg.String() == "q" || msg.String() == "ctrl+c":
		if a.view == viewList {
			return a, tea.Quit
		}
		a.view = viewList
		return a, nil
	case msg.String() == "esc":
		if a.view != viewList {
			a.view = viewList
			return a, nil
		}
	}

	switch a.view {
	case viewSnooze:
		return a.handleSnoozeKey(msg)
	case viewHelp:
		if msg.String() == "?" || msg.String() == "esc" {
			a.view = viewList
		}
		return a, nil
	case viewDetail:
		return a.handleDetailKey(msg)
	default:
		return a.handleListKey(msg)
	}
}

func (a *App) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case " ":
		a.list.toggleSelect()
		// move cursor down after selecting
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(tea.KeyMsg{Type: tea.KeyDown})
		return a, cmd
	case "a":
		ids := a.list.selectedIDs()
		if len(ids) > 0 {
			return a, a.acknowledge(ids)
		}
	case "A":
		ids := a.list.triggeredIDs()
		if len(ids) > 0 {
			return a, a.acknowledge(ids)
		}
	case "x":
		ids := a.list.selectedIDs()
		if len(ids) > 0 {
			return a, a.resolve(ids)
		}
	case "s":
		if a.list.currentIncident() != nil {
			a.prevView = viewList
			a.view = viewSnooze
			a.snooze = newSnoozePicker()
		}
		return a, nil
	case "o":
		if inc := a.list.currentIncident(); inc != nil {
			return a, openBrowser(inc.HTMLURL)
		}
	case "enter":
		if inc := a.list.currentIncident(); inc != nil {
			a.detail.incident = inc
			a.view = viewDetail
		}
		return a, nil
	case "r":
		a.loading = true
		return a, a.fetchIncidents()
	case "1":
		a.scope = pd.ScopeMine
		a.status.scope = a.scope
		a.loading = true
		return a, a.fetchIncidents()
	case "2":
		a.scope = pd.ScopeTeam
		a.status.scope = a.scope
		a.loading = true
		return a, a.fetchIncidents()
	case "3":
		a.scope = pd.ScopeAll
		a.status.scope = a.scope
		a.loading = true
		return a, a.fetchIncidents()
	case "?":
		a.view = viewHelp
		return a, nil
	}

	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	return a, cmd
}

func (a *App) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		if a.detail.incident != nil {
			return a, a.acknowledge([]string{a.detail.incident.ID})
		}
	case "x":
		if a.detail.incident != nil {
			return a, a.resolve([]string{a.detail.incident.ID})
		}
	case "s":
		if a.detail.incident != nil {
			a.prevView = viewDetail
			a.view = viewSnooze
			a.snooze = newSnoozePicker()
		}
	case "o":
		if a.detail.incident != nil {
			return a, openBrowser(a.detail.incident.HTMLURL)
		}
	}
	return a, nil
}

func (a *App) handleSnoozeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		var ids []string
		if a.prevView == viewDetail && a.detail.incident != nil {
			ids = []string{a.detail.incident.ID}
		} else {
			ids = a.list.selectedIDs()
		}
		duration := a.snooze.selectedDuration()
		a.view = a.prevView
		if len(ids) > 0 {
			return a, a.snoozeIncidents(ids, duration)
		}
		return a, nil
	case "esc":
		a.view = a.prevView
		return a, nil
	default:
		var cmd tea.Cmd
		a.snooze, cmd = a.snooze.Update(msg)
		return a, cmd
	}
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var content string

	switch a.view {
	case viewDetail:
		content = a.detail.View()
	case viewHelp:
		content = a.centerOverlay(a.help.View())
	case viewSnooze:
		bg := a.list.View()
		overlay := a.centerOverlay(a.snooze.View())
		content = a.overlayOn(bg, overlay)
	default:
		header := styleTitle.Render("betterpd")
		if a.loading {
			header += " " + styleHelp.Render("loading...")
		}
		content = header + "\n" + a.list.View()
	}

	// Reserve space for the key hint bar and status bar
	available := a.height - 3
	contentStyle := lipgloss.NewStyle().Height(available).MaxHeight(available)

	return contentStyle.Render(content) + "\n" + renderKeyBar(a.width) + "\n" + a.status.View()
}

func (a *App) centerOverlay(s string) string {
	return lipgloss.Place(a.width, a.height-3, lipgloss.Center, lipgloss.Center, s)
}

func (a *App) overlayOn(bg, overlay string) string {
	_ = bg
	return overlay
}

// Commands

// apiTimeout bounds every PagerDuty API call so a stalled request surfaces
// as a visible error instead of leaving the UI silently hanging forever.
const apiTimeout = 15 * time.Second

func (a *App) fetchIncidents() tea.Cmd {
	scope := a.scope
	client := a.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		incidents, err := client.ListIncidents(ctx, scope)
		return incidentsFetchedMsg{incidents: incidents, err: err}
	}
}

// applySnoozeState overlays the app's locally-tracked snooze state onto
// freshly-fetched incidents, and forgets any snooze that has expired.
func (a *App) applySnoozeState(incidents []pd.Incident) {
	now := time.Now()
	for i := range incidents {
		until, ok := a.snoozed[incidents[i].ID]
		if ok && until.After(now) {
			incidents[i].SnoozedUntil = until
		} else if ok {
			delete(a.snoozed, incidents[i].ID)
		}
	}
}

func (a *App) tick() tea.Cmd {
	interval := a.interval
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *App) acknowledge(ids []string) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		err := client.Acknowledge(ctx, ids)
		return actionCompleteMsg{action: fmt.Sprintf("Acknowledged %d incident(s)", len(ids)), err: err}
	}
}

func (a *App) resolve(ids []string) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		err := client.Resolve(ctx, ids)
		return actionCompleteMsg{action: fmt.Sprintf("Resolved %d incident(s)", len(ids)), err: err}
	}
}

func (a *App) snoozeIncidents(ids []string, duration time.Duration) tea.Cmd {
	client := a.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		for _, id := range ids {
			if err := client.Snooze(ctx, id, duration); err != nil {
				return actionCompleteMsg{action: "snooze", err: err}
			}
		}
		return actionCompleteMsg{
			action:       fmt.Sprintf("Snoozed %d incident(s) for %s", len(ids), duration),
			err:          nil,
			snoozedIDs:   ids,
			snoozedUntil: time.Now().Add(duration),
		}
	}
}

var isWSL = sync.OnceValue(func() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
})

func openBrowser(url string) tea.Cmd {
	var c *exec.Cmd
	switch {
	case isWSL():
		// cmd.exe /C start launches the browser and returns immediately; it
		// needs no interactive I/O. Discard its stdio to suppress the "UNC
		// paths are not supported" notice it prints when WSL's interop
		// layer can't map our \\wsl.localhost\... cwd to a Windows path.
		c = exec.Command("cmd.exe", "/C", "start", strings.ReplaceAll(url, "&", "^&"))
		c.Stdout = io.Discard
		c.Stderr = io.Discard
	case runtime.GOOS == "darwin":
		c = exec.Command("open", url)
	case runtime.GOOS == "windows":
		c = exec.Command("cmd", "/C", "start", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}
