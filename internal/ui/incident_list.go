package ui

import (
	"fmt"
	"strings"

	"betterpd/internal/pd"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const linesPerItem = 2

type incidentList struct {
	incidents []pd.Incident
	selected  map[string]bool
	cursor    int
	topIndex  int
	width     int
	height    int
}

func newIncidentList() incidentList {
	return incidentList{
		selected: make(map[string]bool),
	}
}

func (il *incidentList) setSize(w, h int) {
	il.width = w
	il.height = h
	il.clampScroll()
}

func (il *incidentList) setIncidents(incidents []pd.Incident) {
	il.incidents = incidents
	if il.cursor >= len(incidents) {
		il.cursor = len(incidents) - 1
	}
	if il.cursor < 0 {
		il.cursor = 0
	}
	il.clampScroll()
}

func (il *incidentList) itemsPerPage() int {
	n := il.height / linesPerItem
	if n < 1 {
		n = 1
	}
	return n
}

func (il *incidentList) clampScroll() {
	perPage := il.itemsPerPage()
	if il.cursor < il.topIndex {
		il.topIndex = il.cursor
	}
	if il.cursor >= il.topIndex+perPage {
		il.topIndex = il.cursor - perPage + 1
	}
	maxTop := len(il.incidents) - perPage
	if maxTop < 0 {
		maxTop = 0
	}
	if il.topIndex > maxTop {
		il.topIndex = maxTop
	}
	if il.topIndex < 0 {
		il.topIndex = 0
	}
}

func (il *incidentList) moveCursor(delta int) {
	il.cursor += delta
	if il.cursor < 0 {
		il.cursor = 0
	}
	if il.cursor >= len(il.incidents) {
		il.cursor = len(il.incidents) - 1
	}
	if il.cursor < 0 {
		il.cursor = 0
	}
	il.clampScroll()
}

func (il *incidentList) toggleSelect() {
	if il.cursor >= 0 && il.cursor < len(il.incidents) {
		id := il.incidents[il.cursor].ID
		if il.selected[id] {
			delete(il.selected, id)
		} else {
			il.selected[id] = true
		}
	}
}

func (il *incidentList) selectedIDs() []string {
	if len(il.selected) > 0 {
		ids := make([]string, 0, len(il.selected))
		for id := range il.selected {
			ids = append(ids, id)
		}
		return ids
	}
	if il.cursor >= 0 && il.cursor < len(il.incidents) {
		return []string{il.incidents[il.cursor].ID}
	}
	return nil
}

func (il *incidentList) currentIncident() *pd.Incident {
	if il.cursor >= 0 && il.cursor < len(il.incidents) {
		return &il.incidents[il.cursor]
	}
	return nil
}

func (il *incidentList) clearSelection() {
	il.selected = make(map[string]bool)
}

func (il incidentList) Update(msg tea.Msg) (incidentList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyDown:
			il.moveCursor(1)
		case tea.KeyUp:
			il.moveCursor(-1)
		default:
			switch msg.String() {
			case "j":
				il.moveCursor(1)
			case "k":
				il.moveCursor(-1)
			}
		}
	}
	return il, nil
}

// statusColors returns the foreground color for a given incident status.
func statusColor(status string) lipgloss.Color {
	switch status {
	case "triggered":
		return lipgloss.Color("196") // red
	case "acknowledged":
		return lipgloss.Color("214") // amber
	case "resolved":
		return lipgloss.Color("40") // green
	default:
		return lipgloss.Color("245") // gray
	}
}

func statusIcon(status string) string {
	switch status {
	case "triggered":
		return "●"
	case "acknowledged":
		return "◐"
	case "resolved":
		return "✓"
	default:
		return "?"
	}
}

func statusLabel(status string) string {
	switch status {
	case "triggered":
		return "TRIGGERED"
	case "acknowledged":
		return "ACKED"
	case "resolved":
		return "RESOLVED"
	default:
		return strings.ToUpper(status)
	}
}

func (il incidentList) View() string {
	if len(il.incidents) == 0 {
		return styleHelp.Render("No incidents.")
	}

	perPage := il.itemsPerPage()
	end := il.topIndex + perPage
	if end > len(il.incidents) {
		end = len(il.incidents)
	}

	var b strings.Builder
	for i := il.topIndex; i < end; i++ {
		if i > il.topIndex {
			b.WriteString("\n")
		}
		b.WriteString(il.renderItem(i))
	}
	return b.String()
}

func (il incidentList) renderItem(i int) string {
	inc := il.incidents[i]
	isCursor := i == il.cursor
	isSelected := il.selected[inc.ID]
	snoozed := inc.IsSnoozed()

	fg := statusColor(inc.Status)
	if snoozed {
		fg = lipgloss.Color("39") // blue, distinct from the amber "acked" color
	}
	textStyle := lipgloss.NewStyle().Foreground(fg)
	if inc.Status == "triggered" {
		textStyle = textStyle.Bold(true)
	}
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	if isCursor {
		textStyle = textStyle.Background(lipgloss.Color("237"))
		dimStyle = dimStyle.Background(lipgloss.Color("237"))
	}

	bar := "▌"
	marker := " "
	if isSelected {
		marker = "●"
	}

	icon := statusIcon(inc.Status)
	if snoozed {
		icon = "z"
	}

	// Line 1: bar + marker + status icon + number + title
	prefix := fmt.Sprintf("%s%s %s #%d ", bar, marker, icon, inc.Number)
	prefixW := lipgloss.Width(prefix)
	titleW := il.width - prefixW
	if titleW < 5 {
		titleW = 5
	}
	title := truncate(inc.Title, titleW)
	line1 := prefix + title
	line1 = padTo(line1, il.width)

	// Line 2: indent + status label + urgency + service + assigned + created
	indent := "  "
	urgency := "low urgency"
	if inc.Urgency == "high" {
		urgency = "HIGH URGENCY"
	}

	label := statusLabel(inc.Status)
	if snoozed {
		label = fmt.Sprintf("SNOOZED until %s", inc.SnoozedUntil.Format("15:04"))
	}

	fields := []string{label, urgency}
	if inc.ServiceName != "" {
		fields = append(fields, inc.ServiceName)
	}
	if inc.AssignedTo != "" {
		fields = append(fields, inc.AssignedTo)
	}
	fields = append(fields, inc.CreatedAt.Format("Jan 02 15:04"))

	line2Content := indent + fitFields(fields, il.width-lipgloss.Width(indent))
	line2Content = padTo(line2Content, il.width)

	return textStyle.Render(line1) + "\n" + dimStyle.Render(line2Content)
}

// fitFields joins fields with a separator, dropping least-important
// (later) fields from the end until the result fits within maxWidth.
func fitFields(fields []string, maxWidth int) string {
	sep := "  ·  "
	for n := len(fields); n > 0; n-- {
		s := strings.Join(fields[:n], sep)
		if lipgloss.Width(s) <= maxWidth || n == 1 {
			if lipgloss.Width(s) > maxWidth {
				s = truncate(s, maxWidth)
			}
			return s
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

func padTo(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
