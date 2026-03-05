package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"orchestra/internal/aggregator"
	"orchestra/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, "Projects")

	projectsDir := flag.String("projects-dir", defaultDir, "directory containing git projects")
	flag.Parse()

	agg := aggregator.New(*projectsDir)
	model := tui.NewModel(agg)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
