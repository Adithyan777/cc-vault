package tui

import (
	"cc-vault/internal/claude"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderProjectsPanel renders the left panel with project list
func renderProjectsPanel(projects []claude.Project, selected int, active bool, width, height int) string {
	var title string
	if active {
		title = activeTitleStyle.Render(fmt.Sprintf(" Projects (%d) ", len(projects)))
	} else {
		title = inactiveTitleStyle.Render(fmt.Sprintf(" Projects (%d) ", len(projects)))
	}

	contentHeight := height - 4 // account for border + title

	var lines []string
	// Calculate scroll offset
	scrollOffset := 0
	if selected >= contentHeight {
		scrollOffset = selected - contentHeight + 1
	}

	for i := scrollOffset; i < len(projects) && i < scrollOffset+contentHeight; i++ {
		p := projects[i]
		cursor := "  "
		if i == selected {
			cursor = "▸ "
		}

		display := p.DisplayPath
		maxLen := width - 6
		if maxLen > 0 && len(display) > maxLen {
			display = "..." + display[len(display)-maxLen+3:]
		}

		line := cursor + display
		if i == selected {
			line = selectedItemStyle.Width(width - 4).Render(line)
		} else {
			line = normalItemStyle.Render(line)
		}
		lines = append(lines, line)
	}

	// Pad remaining height
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	var panel lipgloss.Style
	if active {
		panel = activePanelStyle
	} else {
		panel = inactivePanelStyle
	}

	return panel.Width(width).Height(height).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, strings.Repeat("─", width-4), content),
	)
}
