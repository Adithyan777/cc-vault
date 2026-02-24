package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"cc-sessions/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := tui.NewModel()

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if user wants to resume a session
	if model, ok := finalModel.(tui.Model); ok {
		if resume := model.GetResumeSession(); resume != nil {
			// Launch claude --resume in the project directory
			claudePath, err := exec.LookPath("claude")
			if err != nil {
				fmt.Fprintf(os.Stderr, "claude not found in PATH\n")
				os.Exit(1)
			}

			args := []string{"claude", "--resume", resume.SessionID}

			// Change to project directory if available
			if resume.ProjectDir != "" {
				if err := os.Chdir(resume.ProjectDir); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to change to project dir: %v\n", err)
				}
			}

			// Replace current process with claude
			if err := syscall.Exec(claudePath, args, os.Environ()); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to exec claude: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
