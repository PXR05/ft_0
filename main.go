package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"ft_0/ui"
)

func main() {
	m := ui.Model{
		Mode:    ui.InitialModeModel(),
		Receive: ui.InitialReceiveModel(),
		Send:    ui.InitialSendModel(),
	}
	prog := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
