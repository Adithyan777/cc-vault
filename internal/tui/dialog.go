package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// DialogType represents different dialog types
type DialogType int

const (
	DialogNone DialogType = iota
	DialogConfirmDelete
	DialogConfirmBulkDelete
	DialogConfirmPrune
	DialogRename
	DialogHelp
)

// Dialog holds dialog state
type Dialog struct {
	Type       DialogType
	Message    string
	Input      string  // for rename dialog
	CursorPos  int
	SessionIDs []string // for bulk operations
}

// renderDialog renders a modal dialog overlay
func renderDialog(d *Dialog, screenWidth, screenHeight int) string {
	if d == nil || d.Type == DialogNone {
		return ""
	}

	var content string

	switch d.Type {
	case DialogConfirmDelete:
		content = lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Delete Session"),
			"",
			d.Message,
			"",
			helpKeyStyle.Render("y")+" confirm  "+helpKeyStyle.Render("n/esc")+" cancel",
		)

	case DialogConfirmBulkDelete:
		content = lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Bulk Delete"),
			"",
			d.Message,
			"",
			helpKeyStyle.Render("y")+" confirm  "+helpKeyStyle.Render("n/esc")+" cancel",
		)

	case DialogConfirmPrune:
		content = lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Prune Empty Sessions"),
			"",
			d.Message,
			"",
			helpKeyStyle.Render("y")+" confirm  "+helpKeyStyle.Render("n/esc")+" cancel",
		)

	case DialogRename:
		inputLine := d.Input + "█"
		content = lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Rename Session"),
			"",
			"Enter new name:",
			searchStyle.Render(inputLine),
			"",
			helpKeyStyle.Render("enter")+" save  "+helpKeyStyle.Render("esc")+" cancel",
		)

	case DialogHelp:
		content = lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Keyboard Shortcuts"),
			"",
			renderHelpContent(),
			"",
			helpKeyStyle.Render("?/esc")+" close help",
		)
	}

	dialog := dialogStyle.Render(content)

	// Center the dialog
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)

	padLeft := (screenWidth - dialogWidth) / 2
	padTop := (screenHeight - dialogHeight) / 2

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	positioned := lipgloss.NewStyle().
		MarginLeft(padLeft).
		MarginTop(padTop).
		Render(dialog)

	return positioned
}

func renderHelpContent() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"↑/↓ or j/k", "Navigate within panel"},
		{"←/→ or h/l", "Switch panels"},
		{"Tab", "Cycle panels"},
		{"Enter", "Resume selected session"},
		{"r", "Rename session"},
		{"d", "Delete session"},
		{"x", "Export session to markdown"},
		{"c", "Copy resume command"},
		{"Space", "Toggle select for bulk ops"},
		{"D", "Bulk delete selected"},
		{"X", "Bulk export selected"},
		{"P", "Prune empty sessions"},
		{"/", "Search sessions"},
		{"?", "Toggle this help"},
		{"q/Ctrl+C", "Quit"},
	}

	var lines []string
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			helpKeyStyle.Width(16).Render(k.key),
			helpDescStyle.Render(k.desc),
		))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
