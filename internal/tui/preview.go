package tui

import (
	"cc-vault/internal/claude"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// msgBlock holds rendered lines for a single message.
type msgBlock struct {
	lines []string
	role  string // "user" or "assistant"
}

// PreviewCache holds rendered preview data.
type PreviewCache struct {
	AllLines   []string   // all messages, used when preview panel is active
	Blocks     []msgBlock // individual message blocks for building summary
	StatsLines []string   // footer stats lines
	TotalMsgs  int        // total number of message blocks
}

// buildPreviewCache renders all messages into styled lines.
// Called once per session change, cached on the model.
func buildPreviewCache(preview *claude.PreviewData, session *claude.Session, projectConfig *claude.ProjectConfig, contentWidth int) *PreviewCache {
	empty := &PreviewCache{
		AllLines: []string{mutedStyle.Render("  Select a session to preview")},
	}
	if preview == nil || session == nil {
		return empty
	}

	if contentWidth < 10 {
		contentWidth = 10
	}

	renderWidth := contentWidth - 4 // room for "   " indent
	if renderWidth < 10 {
		renderWidth = 10
	}

	// Render each message into blocks of lines
	var blocks []msgBlock

	for _, msg := range preview.Messages {
		label := userLabelStyle.Render("User")
		if msg.Role == "assistant" {
			label = assistantLabelStyle.Render("Agent")
		}

		var block []string
		block = append(block, label+" "+mutedStyle.Render("───"))

		rendered := renderMarkdownLines(msg.Content, renderWidth)
		for _, line := range rendered {
			block = append(block, "   "+line)
		}
		block = append(block, "")
		blocks = append(blocks, msgBlock{lines: block, role: msg.Role})
	}

	// Build stats footer
	var statsLines []string
	statsLines = append(statsLines, strings.Repeat("─", contentWidth))

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
		statsLines = append(statsLines, statStyle.Render(strings.Join(stats, "  ")))
	}

	// All lines: every message block + stats
	var allLines []string
	for _, b := range blocks {
		allLines = append(allLines, b.lines...)
	}
	allLines = append(allLines, statsLines...)

	return &PreviewCache{
		AllLines:   allLines,
		Blocks:     blocks,
		StatsLines: statsLines,
		TotalMsgs:  len(blocks),
	}
}

// BuildSummaryLines creates a truncated summary showing first and last user+assistant pairs.
// Called from View() with the actual panel height so sizing is always correct.
func BuildSummaryLines(cache *PreviewCache, panelHeight int) []string {
	if cache == nil || len(cache.Blocks) == 0 {
		if cache != nil {
			return cache.StatsLines
		}
		return nil
	}
	blocks := cache.Blocks
	statsLines := cache.StatsLines
	if len(blocks) == 0 {
		return statsLines
	}

	// Find first user+assistant pair and last user+assistant pair
	var firstPair, lastPair []msgBlock

	// First pair: first user message + the assistant response after it
	for i, b := range blocks {
		if b.role == "user" {
			firstPair = append(firstPair, b)
			if i+1 < len(blocks) && blocks[i+1].role == "assistant" {
				firstPair = append(firstPair, blocks[i+1])
			}
			break
		}
	}

	// Last pair: last user message + the assistant response after it
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].role == "user" {
			lastPair = append(lastPair, blocks[i])
			if i+1 < len(blocks) && blocks[i+1].role == "assistant" {
				lastPair = append(lastPair, blocks[i+1])
			}
			break
		}
	}

	// Build summary blocks, avoiding duplicates if first==last pair
	var summaryBlocks []msgBlock
	summaryBlocks = append(summaryBlocks, firstPair...)

	// Only add last pair if it's different from first pair (check by index)
	firstUserIdx := -1
	lastUserIdx := -1
	for i, b := range blocks {
		if b.role == "user" {
			if firstUserIdx == -1 {
				firstUserIdx = i
			}
			lastUserIdx = i
		}
	}
	if lastUserIdx > firstUserIdx && len(lastPair) > 0 {
		summaryBlocks = append(summaryBlocks, lastPair...)
	}

	hasSeparator := len(blocks) > len(summaryBlocks)
	contentHeight := panelHeight - 4
	overhead := len(statsLines)
	if hasSeparator {
		overhead += 2 // separator + blank line
	}
	available := contentHeight - overhead

	// Calculate total lines needed at full size
	totalNeeded := 0
	for _, b := range summaryBlocks {
		totalNeeded += len(b.lines)
	}

	// Calculate per-block line budgets
	// Short blocks keep their natural size, savings redistribute to longer blocks
	budgets := make([]int, len(summaryBlocks))
	if totalNeeded <= available {
		for i, b := range summaryBlocks {
			budgets[i] = len(b.lines)
		}
	} else {
		remaining := available
		blocksLeft := len(summaryBlocks)
		assigned := make([]bool, len(summaryBlocks))

		// Multi-pass: assign blocks that fit within fair share, redistribute leftovers
		for pass := 0; pass < len(summaryBlocks); pass++ {
			fairShare := remaining / blocksLeft
			if fairShare < 3 {
				fairShare = 3
			}
			changed := false
			for i, b := range summaryBlocks {
				if assigned[i] {
					continue
				}
				if len(b.lines) <= fairShare {
					budgets[i] = len(b.lines)
					remaining -= len(b.lines)
					blocksLeft--
					assigned[i] = true
					changed = true
				}
			}
			if !changed {
				break
			}
		}
		// Remaining space goes to unassigned (long) blocks
		for i := range summaryBlocks {
			if !assigned[i] {
				share := remaining / blocksLeft
				if share < 3 {
					share = 3
				}
				budgets[i] = share
				remaining -= share
				blocksLeft--
			}
		}
	}

	// Build summary with per-block budgets
	var result []string

	firstPairLen := len(firstPair)
	for i, b := range summaryBlocks[:firstPairLen] {
		result = append(result, truncateBlock(b.lines, budgets[i])...)
	}

	if hasSeparator {
		hidden := len(blocks) - len(summaryBlocks)
		separator := mutedStyle.Render(fmt.Sprintf("  ··· %d more messages ···", hidden))
		result = append(result, separator, "")
	}

	if firstPairLen < len(summaryBlocks) {
		for i, b := range summaryBlocks[firstPairLen:] {
			result = append(result, truncateBlock(b.lines, budgets[firstPairLen+i])...)
		}
	}

	result = append(result, statsLines...)
	return result
}

// truncateBlock truncates a message block to maxLines, adding a "..." indicator if truncated.
func truncateBlock(lines []string, maxLines int) []string {
	if len(lines) <= maxLines {
		return lines
	}
	truncated := make([]string, 0, maxLines)
	truncated = append(truncated, lines[:maxLines-1]...)
	truncated = append(truncated, mutedStyle.Render("   ..."))
	return truncated
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
