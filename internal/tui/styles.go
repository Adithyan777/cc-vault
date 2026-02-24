package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // purple
	secondaryColor = lipgloss.Color("#06B6D4") // cyan
	accentColor    = lipgloss.Color("#F59E0B") // amber
	mutedColor     = lipgloss.Color("#6B7280") // gray
	textColor      = lipgloss.Color("#E5E7EB") // light gray
	bgColor        = lipgloss.Color("#1F2937") // dark bg
	selectedBg     = lipgloss.Color("#374151") // selected bg
	errorColor     = lipgloss.Color("#EF4444") // red
	successColor   = lipgloss.Color("#10B981") // green
	pinnedColor    = lipgloss.Color("#F59E0B") // amber for pinned

	// Panel styles
	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	inactivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(mutedColor).
				Padding(0, 1)

	// Title styles
	activeTitleStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	inactiveTitleStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Bold(true)

	// Item styles
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Background(selectedBg).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(textColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	pinnedStyle = lipgloss.NewStyle().
			Foreground(pinnedColor).
			Bold(true)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(lipgloss.Color("#111827")).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Dialog styles
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Width(50)

	dialogTitleStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	// Preview styles
	userMsgStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	assistantMsgStyle = lipgloss.NewStyle().
				Foreground(successColor)

	userLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EC4899")).
			Bold(true)

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	statStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// Search
	searchStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
)
