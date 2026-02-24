package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	mdHeaderStyle    = lipgloss.NewStyle().Bold(true).Foreground(secondaryColor)
	mdCodeLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	mdCodeFenceStyle = lipgloss.NewStyle().Foreground(mutedColor)
	mdBulletStyle    = lipgloss.NewStyle().Foreground(accentColor)
)

// renderMarkdownLines does fast, lightweight markdown rendering.
// Handles: headers, code blocks, bullet lists, word wrap.
func renderMarkdownLines(content string, width int) []string {
	if width < 10 {
		width = 10
	}

	srcLines := strings.Split(content, "\n")
	var out []string
	inCodeBlock := false

	for _, line := range srcLines {
		trimmed := strings.TrimSpace(line)

		// Code block fences
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			out = append(out, mdCodeFenceStyle.Render(trimmed))
			continue
		}

		// Inside code block - render as-is with code style
		if inCodeBlock {
			out = append(out, mdCodeLineStyle.Render(line))
			continue
		}

		// Empty line
		if trimmed == "" {
			out = append(out, "")
			continue
		}

		// Headers
		if strings.HasPrefix(trimmed, "### ") {
			out = append(out, mdHeaderStyle.Render(trimmed[4:]))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			out = append(out, mdHeaderStyle.Render(trimmed[3:]))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			out = append(out, mdHeaderStyle.Render(trimmed[2:]))
			continue
		}

		// Bullet lists - keep prefix, wrap rest
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			bullet := mdBulletStyle.Render(string(trimmed[0])) + " "
			rest := trimmed[2:]
			wrapped := wordWrap(rest, width-2)
			for i, wl := range wrapped {
				if i == 0 {
					out = append(out, bullet+wl)
				} else {
					out = append(out, "  "+wl)
				}
			}
			continue
		}

		// Numbered lists
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			dotIdx := strings.Index(trimmed, ". ")
			if dotIdx > 0 && dotIdx <= 3 {
				prefix := trimmed[:dotIdx+2]
				rest := trimmed[dotIdx+2:]
				wrapped := wordWrap(rest, width-len(prefix))
				pad := strings.Repeat(" ", len(prefix))
				for i, wl := range wrapped {
					if i == 0 {
						out = append(out, prefix+wl)
					} else {
						out = append(out, pad+wl)
					}
				}
				continue
			}
		}

		// Horizontal rules
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			out = append(out, mdCodeFenceStyle.Render(strings.Repeat("─", width)))
			continue
		}

		// Regular text - word wrap
		wrapped := wordWrap(trimmed, width)
		out = append(out, wrapped...)
	}

	return out
}

// wordWrap splits text into lines of at most width characters, breaking on spaces.
func wordWrap(text string, width int) []string {
	if width <= 0 || len(text) <= width {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	lines = append(lines, cur)
	return lines
}
