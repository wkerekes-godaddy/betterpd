package ui

import (
	"fmt"
	"strings"
	"time"

	"betterpd/internal/pd"
	"github.com/charmbracelet/lipgloss"
)

type statusBar struct {
	orgName     string
	scope       pd.Scope
	lastRefresh time.Time
	count       int
	width       int
	message     string
	messageTime time.Time
}

func (s *statusBar) setMessage(msg string) {
	s.message = msg
	s.messageTime = time.Now()
}

func (s statusBar) View() string {
	left := fmt.Sprintf("[%s] %d incidents", s.scope, s.count)

	var middle string
	if s.message != "" && time.Since(s.messageTime) < 5*time.Second {
		middle = s.message
	} else if s.orgName != "" {
		middle = styleOrgName.Render(s.orgName)
	}

	var right string
	if !s.lastRefresh.IsZero() {
		right = fmt.Sprintf("refreshed %s ago", timeAgo(s.lastRefresh))
	}

	// styleStatusBar has its own Padding(0, 1), which adds 2 columns on
	// top of whatever width we render, so target 2 fewer columns here.
	innerWidth := s.width - 2
	if innerWidth < 0 {
		innerWidth = 0
	}

	bar := layoutStatusBar(left, middle, right, innerWidth)

	// bar is built to exactly innerWidth, so Width() here only pads
	// (never wraps) and MaxHeight guards against any rounding slack.
	return styleStatusBar.Width(innerWidth).MaxHeight(1).Render(bar)
}

// layoutStatusBar places left flush left, right flush right, and middle
// centered across the full innerWidth, dropping/truncating pieces (in that
// priority order: middle, then right, then left) if they don't all fit.
func layoutStatusBar(left, middle, right string, innerWidth int) string {
	lw, mw, rw := lipgloss.Width(left), lipgloss.Width(middle), lipgloss.Width(right)

	for lw+mw+rw > innerWidth {
		switch {
		case mw > 0:
			middle, mw = "", 0
		case rw > 0:
			right, rw = "", 0
		default:
			left = truncate(left, innerWidth)
			lw = lipgloss.Width(left)
		}
	}

	if mw == 0 {
		gap := innerWidth - lw - rw
		if gap < 0 {
			gap = 0
		}
		return left + strings.Repeat(" ", gap) + right
	}

	midStart := (innerWidth - mw) / 2
	if midStart < lw {
		midStart = lw
	}
	leftGap := midStart - lw

	rightGap := innerWidth - midStart - mw - rw
	if rightGap < 0 {
		rightGap = 0
	}

	return left + strings.Repeat(" ", leftGap) + middle + strings.Repeat(" ", rightGap) + right
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}
