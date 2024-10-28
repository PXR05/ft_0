package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"ft_0/ui"
)

func main() {
	model := ui.InitialModel()
	p := tea.NewProgram(&model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
