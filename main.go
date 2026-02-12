package main

import (
	"fmt"
	"os"

	"mplaya/player"
	"mplaya/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "play" {
		runSmartPlayPause()
		return
	}

	p := tea.NewProgram(tui.NewModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runSmartPlayPause() {
	_, err := player.SmartPlayPause()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
