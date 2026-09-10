package ui

import (
	"fmt"
	"strings"

	"betterpd/internal/pd"
	"github.com/charmbracelet/lipgloss"
)

type incidentDetail struct {
	incident *pd.Incident
	width    int
	height   int
}

func (d incidentDetail) View() string {
	if d.incident == nil {
		return ""
	}

	inc := d.incident
	var b strings.Builder

	title := styleTitle.Render(fmt.Sprintf("#%d %s", inc.Number, inc.Title))
	b.WriteString(title + "\n\n")

	labelStyle := lipgloss.NewStyle().Bold(true).Width(12)

	b.WriteString(labelStyle.Render("Status:") + " " + statusString(inc.Status) + "\n")
	if inc.IsSnoozed() {
		b.WriteString(labelStyle.Render("Snoozed:") + " " + styleSnoozed.Render("until "+inc.SnoozedUntil.Format("2006-01-02 15:04:05 MST")) + "\n")
	}
	b.WriteString(labelStyle.Render("Urgency:") + " " + inc.Urgency + "\n")
	b.WriteString(labelStyle.Render("Service:") + " " + inc.ServiceName + "\n")
	b.WriteString(labelStyle.Render("Assigned:") + " " + inc.AssignedTo + "\n")
	b.WriteString(labelStyle.Render("Created:") + " " + inc.CreatedAt.Format("2006-01-02 15:04:05 MST") + "\n")
	b.WriteString(labelStyle.Render("URL:") + " " + inc.HTMLURL + "\n")

	b.WriteString("\n" + styleHelp.Render("a=ack  x=resolve  s=snooze  o=open  esc=back"))

	return b.String()
}

func statusString(s string) string {
	switch s {
	case "triggered":
		return styleTriggered.Render("TRIGGERED")
	case "acknowledged":
		return styleAcked.Render("ACKNOWLEDGED")
	case "resolved":
		return styleResolved.Render("RESOLVED")
	default:
		return s
	}
}
