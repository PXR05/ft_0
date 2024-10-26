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

	// if len(os.Args) < 2 {
	// 	fmt.Println("Usage:")
	// 	fmt.Println("Start relay server: go run main.go relay")
	// 	fmt.Println("To receive files: go run main.go receive")
	// 	fmt.Println("To send a file: go run main.go send <filepath>")
	// 	os.Exit(1)
	// }

	// mode := os.Args[1]

	// switch mode {
	// case "relay":
	// 	startRelayServer()
	// case "receive":
	// 	startReceiver()
	// case "send":
	// 	if len(os.Args) != 3 {
	// 		fmt.Println("Send mode requires filepath")
	// 		fmt.Println("Usage: go run main.go send <filepath>")
	// 		os.Exit(1)
	// 	}
	// 	startSender(os.Args[2])
	// default:
	// 	fmt.Println("Invalid mode. Use 'relay', 'send', or 'receive'")
	// 	os.Exit(1)
	// }
}
