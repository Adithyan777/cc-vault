package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"cc-vault/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// version is set by GoReleaser via ldflags at build time.
var version = "dev"

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
			if resume.ProjectDir != "" {
				if err := os.Chdir(resume.ProjectDir); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to change to project dir: %v\n", err)
				}
			}

			claudePath, err := exec.LookPath("claude")
			if err != nil {
				fmt.Fprintf(os.Stderr, "claude not found in PATH\n")
				os.Exit(1)
			}

			args := []string{"claude", "--resume", resume.SessionID}
			if err := syscall.Exec(claudePath, args, os.Environ()); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to exec claude: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
