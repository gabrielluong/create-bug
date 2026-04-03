package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabrielluong/create-bug/internal/config"
	"github.com/gabrielluong/create-bug/internal/tui"
)

func main() {
	cfg := config.Load()

	app := tui.NewApp(cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
