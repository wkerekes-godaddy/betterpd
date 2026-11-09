package blame

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type DisplayOptions struct {
	UserID     string
	FutureOnly bool
}

func FilterChanges(changes []Change, opts DisplayOptions) []Change {
	now := time.Now()
	var filtered []Change
	for _, c := range changes {
		if opts.FutureOnly && !c.End.IsZero() && c.End.Before(now) {
			continue
		}
		if opts.UserID != "" && !changeInvolvesUser(c, opts.UserID) {
			continue
		}
		filtered = append(filtered, c)
	}
	return filtered
}

func changeInvolvesUser(c Change, userID string) bool {
	if c.OriginalUserID != "" || c.NewUserID != "" {
		return c.OriginalUserID == userID || c.NewUserID == userID
	}
	return false
}

type changeGroup struct {
	time     string
	schedule string
	changes  []Change
}

func groupChanges(changes []Change) []changeGroup {
	var groups []changeGroup
	for _, c := range changes {
		key := c.DetectedAt.Truncate(time.Minute).Format(time.RFC3339)
		sched := c.ScheduleName

		if len(groups) > 0 {
			last := &groups[len(groups)-1]
			if last.time == key && last.schedule == sched {
				last.changes = append(last.changes, c)
				continue
			}
		}
		groups = append(groups, changeGroup{
			time:     key,
			schedule: sched,
			changes:  []Change{c},
		})
	}
	return groups
}

func DisplayChanges(changes []Change, w io.Writer) {
	if len(changes) == 0 {
		fmt.Fprintln(w, "No schedule changes detected.")
		return
	}

	groups := groupChanges(changes)

	var currentDate string
	for _, g := range groups {
		first := g.changes[0]
		date := first.DetectedAt.Local().Format("Mon, Jan 2 2006")
		if date != currentDate {
			if currentDate != "" {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "── %s ──\n", date)
			currentDate = date
		}

		timeStr := first.DetectedAt.Local().Format("3:04 PM")
		schedName := shortScheduleName(g.schedule)

		if len(g.changes) == 1 {
			fmt.Fprintf(w, "\n  %s  %s\n", timeStr, schedName)
		} else {
			fmt.Fprintf(w, "\n  %s  %s (%d changes)\n", timeStr, schedName, len(g.changes))
		}

		actor := commonActor(g.changes)
		for _, c := range g.changes {
			period := formatPeriodInline(c)
			fmt.Fprintf(w, "    %s %s%s\n", changeTypeLabel(c.ChangeType), c.Description, period)
		}

		if actor != "" {
			fmt.Fprintf(w, "    Changed by: %s\n", actor)
		}
	}
	fmt.Fprintln(w)
}

func commonActor(changes []Change) string {
	actor := changes[0].ChangedBy
	if actor == "" {
		return ""
	}
	for _, c := range changes[1:] {
		if c.ChangedBy != actor {
			return ""
		}
	}
	return actor
}

func formatPeriodInline(c Change) string {
	if c.Start.IsZero() {
		return ""
	}
	return fmt.Sprintf("  (%s – %s)",
		c.Start.Local().Format("Jan 2 3:04 PM"),
		c.End.Local().Format("Jan 2 3:04 PM"))
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
