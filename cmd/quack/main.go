package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"quack/internal/sessions"
	"quack/internal/ui"
)

func main() {
	provider := sessions.NewOpenCodeProvider("opencode")
	program := tea.NewProgram(ui.NewModel(provider), tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "quack failed: %v\n", err)
		os.Exit(1)
	}
}
