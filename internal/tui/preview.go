package tui

import (
	"cc-sessions/internal/claude"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// buildPreviewLines renders all messages into styled lines.
// Uses lightweight markdown rendering instead of glamour.
// Called once per session change, cached on the model.
func buildPreviewLines(preview *claude.PreviewData, session *claude.Session, projectConfig *claude.ProjectConfig, contentWidth int) []string {
	if preview == nil || session == nil {
		return []string{mutedStyle.Render("  Select a session to preview")}
	}

	if contentWidth < 10 {
		contentWidth = 10
	}

	renderWidth := contentWidth - 4 // room for "   " indent
	if renderWidth < 10 {
		renderWidth = 10
	}

	var lines []string

	for _, msg := range preview.Messages {
		label := userLabelStyle.Render("User")
		if msg.Role == "assistant" {
			label = assistantLabelStyle.Render("Agent")
		}

		lines = append(lines, label+" "+mutedStyle.Render("───"))

		// Lightweight markdown rendering
		rendered := renderMarkdownLines(msg.Content, renderWidth)
		for _, line := range rendered {
			lines = append(lines, "   "+line)
		}
		lines = append(lines, "")
	}

	// Stats footer
	lines = append(lines, strings.Repeat("─", contentWidth))

	var stats []string
	if session.GitBranch != "" {
		stats = append(stats, fmt.Sprintf("🌿 %s", session.GitBranch))
	}
	stats = append(stats, fmt.Sprintf("💬 %d msgs", preview.TotalMessages))

	if projectConfig != nil && session.IsPinned {
		if projectConfig.LastCost > 0 {
			stats = append(stats, fmt.Sprintf("💰$%.2f", projectConfig.LastCost))
		}
		totalTokens := projectConfig.LastInputTokens + projectConfig.LastOutputTokens
		if totalTokens > 0 {
			if totalTokens > 1000 {
				stats = append(stats, fmt.Sprintf("🔤%dk", totalTokens/1000))
			} else {
				stats = append(stats, fmt.Sprintf("🔤%d", totalTokens))
			}
		}
		if projectConfig.LastDuration > 0 {
			d := time.Duration(projectConfig.LastDuration) * time.Millisecond
			if d.Minutes() >= 1 {
				stats = append(stats, fmt.Sprintf("⏱ %.0fm", d.Minutes()))
			} else {
				stats = append(stats, fmt.Sprintf("⏱ %.0fs", d.Seconds()))
			}
		}
	}

	if len(stats) > 0 {
		lines = append(lines, statStyle.Render(strings.Join(stats, "  ")))
	}

	return lines
}

// renderPreviewPanel takes pre-built cached lines and handles scrolling + framing.
// Called from View() every frame - must be fast (no rendering, just slicing).
func renderPreviewPanel(cachedLines []string, active bool, width, height int, scroll int, branch string, msgCount int) (string, int) {
	titleStyle := inactiveTitleStyle
	if active {
		titleStyle = activeTitleStyle
	}
	title := titleStyle.Render(" Preview ")

	contentHeight := height - 4
	totalLines := len(cachedLines)

	maxScroll := totalLines - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	var visible []string
	end := scroll + contentHeight
	if end > totalLines {
		end = totalLines
	}
	if scroll < totalLines {
		visible = cachedLines[scroll:end]
	}

	for len(visible) < contentHeight {
		visible = append(visible, "")
	}

	// Right-side info: branch + msg count + scroll %
	var rightParts []string
	if branch != "" {
		rightParts = append(rightParts, mutedStyle.Render(branch))
	}
	if msgCount > 0 {
		rightParts = append(rightParts, mutedStyle.Render(fmt.Sprintf("%d msgs", msgCount)))
	}
	if active && maxScroll > 0 {
		pct := scroll * 100 / maxScroll
		rightParts = append(rightParts, mutedStyle.Render(fmt.Sprintf("%d%%", pct)))
	}
	if len(rightParts) > 0 {
		rightInfo := strings.Join(rightParts, " | ")
		gap := width - lipgloss.Width(title) - lipgloss.Width(rightInfo) - 6
		if gap < 1 {
			gap = 1
		}
		title += strings.Repeat(" ", gap) + rightInfo
	}

	content := lipgloss.JoinVertical(lipgloss.Left, visible...)

	var panel lipgloss.Style
	if active {
		panel = activePanelStyle
	} else {
		panel = inactivePanelStyle
	}

	rendered := panel.Width(width).Height(height).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, strings.Repeat("─", width-4), content),
	)

	return rendered, totalLines
}
