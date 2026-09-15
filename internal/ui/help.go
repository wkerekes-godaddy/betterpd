package ui

import "github.com/charmbracelet/lipgloss"

type helpOverlay struct {
	width  int
	height int
}

func (h helpOverlay) View() string {
	content := styleTitle.Render("Keybindings") + "\n\n"

	bindings := []struct{ key, desc string }{
		{"j/k, ↑/↓", "Navigate"},
		{"space", "Toggle select"},
		{"a", "Acknowledge"},
		{"A", "Acknowledge all triggered"},
		{"x", "Resolve"},
		{"s", "Snooze"},
		{"o", "Open in browser"},
		{"enter", "View details"},
		{"esc", "Back"},
		{"r", "Refresh"},
		{"1/2/3", "Scope: mine/team/all"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}

	keyStyle := lipgloss.NewStyle().Bold(true).Width(12)
	for _, b := range bindings {
		content += keyStyle.Render(b.key) + " " + b.desc + "\n"
	}

	return styleOverlay.Render(content)
}
