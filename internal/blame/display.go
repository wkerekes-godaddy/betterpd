package blame

import (
	"fmt"
	"io"
	"strings"
	"time"
)

func DisplayChanges(changes []Change, w io.Writer) {
	if len(changes) == 0 {
		fmt.Fprintln(w, "No schedule changes detected.")
		return
	}

	var currentDate string
	for _, c := range changes {
		date := c.DetectedAt.Local().Format("Mon, Jan 2 2006")
		if date != currentDate {
			if currentDate != "" {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "── %s ──\n", date)
			currentDate = date
		}

		timeStr := c.DetectedAt.Local().Format("3:04 PM")
		schedName := shortScheduleName(c.ScheduleName)

		fmt.Fprintf(w, "\n  %s  %s\n", timeStr, schedName)
		fmt.Fprintf(w, "    %s %s\n", changeTypeLabel(c.ChangeType), c.Description)

		if !c.Start.IsZero() {
			fmt.Fprintf(w, "    Period: %s – %s\n",
				c.Start.Local().Format("Jan 2 3:04 PM"),
				c.End.Local().Format("Jan 2 3:04 PM"))
		}

		if c.ChangedBy != "" {
			fmt.Fprintf(w, "    Changed by: %s\n", c.ChangedBy)
		}
	}
	fmt.Fprintln(w)
}

func DisplayInitialSync(scheduleCount int, w io.Writer) {
	fmt.Fprintf(w, "Initial snapshot recorded for %d schedule(s).\n", scheduleCount)
	fmt.Fprintln(w, "Run 'betterpd blame' again after the next sync to see changes.")
}

func shortScheduleName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func changeTypeLabel(t string) string {
	switch t {
	case "added":
		return "[+]"
	case "removed":
		return "[-]"
	case "modified":
		return "[~]"
	case "override":
		return "[!]"
	default:
		return "[?]"
	}
}

func DisplaySyncSkipped(lastSync time.Time, w io.Writer) {
	ago := time.Since(lastSync).Truncate(time.Minute)
	fmt.Fprintf(w, "Last synced %s ago. Use --sync to force a refresh.\n\n", ago)
}
