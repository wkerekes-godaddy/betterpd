package ui

import "github.com/charmbracelet/lipgloss"

type keyHint struct {
	key, label string
}

// Ordered by priority; later hints are dropped first on narrow terminals.
var commonKeyHints = []keyHint{
	{"j/k", "nav"},
	{"a", "ack"},
	{"x", "resolve"},
	{"s", "snooze"},
	{"o", "open"},
	{"enter", "detail"},
	{"?", "help"},
	{"q", "quit"},
	{"r", "refresh"},
	{"space", "select"},
}

func renderKeyBar(width int) string {
	sep := "  "
	innerWidth := width - 2 // account for Padding(0, 1) on each side

	var plain string
	var render string
	for i, h := range commonKeyHints {
		part := h.key + " " + h.label
		candidate := part
		if i > 0 {
			candidate = sep + part
		}
		if lipgloss.Width(plain+candidate) > innerWidth {
			break
		}
		plain += candidate
		if i > 0 {
			render += styleKeyBarLabel.Render(sep)
		}
		render += styleKeyBarKey.Render(h.key) + styleKeyBarLabel.Render(" "+h.label)
	}

	return styleKeyBar.Width(width).MaxHeight(1).Render(render)
}
