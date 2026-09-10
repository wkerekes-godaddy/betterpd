package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type snoozeOption struct {
	label    string
	duration time.Duration
}

var snoozeOptions = []snoozeOption{
	{"30 minutes", 30 * time.Minute},
	{"1 hour", 1 * time.Hour},
	{"4 hours", 4 * time.Hour},
	{"24 hours", 24 * time.Hour},
}

type snoozePicker struct {
	cursor int
	width  int
	height int
}

func newSnoozePicker() snoozePicker {
	return snoozePicker{}
}

func (sp snoozePicker) Update(msg tea.Msg) (snoozePicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			sp.cursor++
			if sp.cursor >= len(snoozeOptions) {
				sp.cursor = len(snoozeOptions) - 1
			}
		case "k", "up":
			sp.cursor--
			if sp.cursor < 0 {
				sp.cursor = 0
			}
		}
	}
	return sp, nil
}

func (sp snoozePicker) selectedDuration() time.Duration {
	return snoozeOptions[sp.cursor].duration
}

func (sp snoozePicker) View() string {
	var items []string
	for i, opt := range snoozeOptions {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == sp.cursor {
			cursor = "▸ "
			style = style.Bold(true).Foreground(lipgloss.Color("86"))
		}
		items = append(items, style.Render(cursor+opt.label))
	}

	content := styleTitle.Render("Snooze for:") + "\n\n"
	for _, item := range items {
		content += item + "\n"
	}
	content += "\n" + styleHelp.Render("enter=confirm  esc=cancel")

	return styleOverlay.Render(content)
}
