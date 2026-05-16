package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SmolNero/quack/internal/sessions"
	"github.com/SmolNero/quack/internal/ui"
)

func main() {
	provider := sessions.NewOpenCodeProvider("opencode")
	program := tea.NewProgram(ui.NewModel(provider), tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "quack failed: %v\n", err)
		os.Exit(1)
	}
}
