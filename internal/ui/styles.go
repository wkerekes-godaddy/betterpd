package ui

import "github.com/charmbracelet/lipgloss"

var (
	styleTriggered = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleAcked     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleResolved  = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	styleHigh      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleLow       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	styleSelected = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Bold(true)

	styleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	styleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("40"))

	styleOverlay = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	styleKeyBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	styleKeyBarKey = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("86")).
			Bold(true)

	styleKeyBarLabel = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Foreground(lipgloss.Color("245"))

	styleOrgName = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("86")).
			Bold(true)

	styleSnoozed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))
)
