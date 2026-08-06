package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SmolNero/quack/internal/sessions"
	"github.com/SmolNero/quack/internal/ui"
	"github.com/SmolNero/quack/internal/usage"
)

func main() {
	provider := sessions.NewOpenCodeProvider("opencode")
	titleCtx, cancelTitles := context.WithTimeout(context.Background(), 3*time.Second)
	provider.SetTerminalTitles(sessions.KittyTerminalTitles(titleCtx))
	cancelTitles()

	usageProvider := usage.NewOpenAIProvider()
	program := tea.NewProgram(ui.NewModel(provider, usageProvider), tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "quack failed: %v\n", err)
		os.Exit(1)
	}
}
