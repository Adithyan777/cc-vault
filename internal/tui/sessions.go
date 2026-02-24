package tui

import (
	"cc-vault/internal/claude"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderSessionsPanel renders the middle panel with session list
func renderSessionsPanel(sessions []claude.Session, selected int, active bool, width, height int, searchActive bool, searchQuery string) string {
	var title string
	if searchActive {
		queryDisplay := searchQuery
		maxQueryLen := width - 22
		if maxQueryLen > 0 && len(queryDisplay) > maxQueryLen {
			queryDisplay = queryDisplay[:maxQueryLen-3] + "..."
		}
		var titleStyle lipgloss.Style
		if active {
			titleStyle = activeTitleStyle
		} else {
			titleStyle = inactiveTitleStyle
		}
		title = titleStyle.Render(" Search: ") +
			searchStyle.Render("\""+queryDisplay+"\"") +
			titleStyle.Render(fmt.Sprintf(" (%d results) ", len(sessions)))
	} else if active {
		title = activeTitleStyle.Render(fmt.Sprintf(" Sessions (%d) ", len(sessions)))
	} else {
		title = inactiveTitleStyle.Render(fmt.Sprintf(" Sessions (%d) ", len(sessions)))
	}

	contentHeight := height - 4

	var lines []string

	if len(sessions) == 0 {
		lines = append(lines, mutedStyle.Render("  No sessions found"))
	} else {
		scrollOffset := 0
		if selected >= contentHeight {
			scrollOffset = selected - contentHeight + 1
		}

		for i := scrollOffset; i < len(sessions) && i < scrollOffset+contentHeight; i++ {
			s := sessions[i]

			cursor := "  "
			if i == selected {
				cursor = "▸ "
			}

			// Selection marker for bulk operations
			selectMark := " "
			if s.Selected {
				selectMark = "●"
			}

			// Pin indicator
			pin := " "
			if s.IsPinned {
				pin = "📌"
			}

			date := s.Date.Format("02/01")
			name := s.DisplayName()

			maxNameLen := width - 16
			if maxNameLen > 0 && len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			line := fmt.Sprintf("%s%s%s %s %s", cursor, selectMark, pin, date, name)

			if i == selected {
				line = selectedItemStyle.Width(width - 4).Render(line)
			} else if s.IsPinned {
				line = pinnedStyle.Render(line)
			} else {
				line = normalItemStyle.Render(line)
			}

			lines = append(lines, line)
		}
	}

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
