package ui

import (
	"fmt"
	"ft_0/server"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nsf/termbox-go"
)

type Model struct {
	Mode    ModeModel
	Receive ReceiveModel
	Send    SendModel
	Relay   RelayModel
}

const (
	Accent = "#ffffaf"
	Muted  = "#4d4d4d"
	Normal = "#dddddd"
)

var (
	Container = lipgloss.NewStyle().Padding(0, 2)
)

var Error error

func (m Model) Init() tea.Cmd {
	return tea.SetWindowTitle("FT_0")
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case server.RelayLogMsg:
		if m.Mode.Choice == "Relay" {
			m.Relay, cmd = m.Relay.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			Error = nil
			if m.Mode.Choice != "" {
				switch m.Mode.Choice {
				case "Relay":
					m.Relay.Stop()
				}
				m.Mode.Choice = ""
				return m, nil
			}
			return m, tea.Quit
		}
		if m.Mode.Choice == "Relay" {
			m.Relay, cmd = m.Relay.Update(msg)
			return m, cmd
		}
		m.Mode, cmd = m.Mode.Update(msg)
	}

	return m, cmd
}

var (
	W int
	H int
)

func (m *Model) View() string {
	var s strings.Builder
	if Error != nil {
		s.WriteString(Error.Error())
	}
	if m.Mode.Choice != "" {
		switch m.Mode.Choice {
		case "Send":
			mod := InitialSendModel()
			prog := tea.NewProgram(&mod, tea.WithAltScreen())
			if _, err := prog.Run(); err != nil {
				fmt.Println("Error running program:", err)
			}
			prog.Kill()
			m.Mode.Choice = ""
		case "Receive":
			mod := InitialReceiveModel()
			prog := tea.NewProgram(&mod, tea.WithAltScreen())
			if _, err := prog.Run(); err != nil {
				fmt.Println("Error running program:", err)
			}
			prog.Kill()
			m.Mode.Choice = ""
		case "Relay":
			s.WriteString(m.Relay.View())
		}
	} else {
		s.WriteString(m.Mode.ModeList.View())
	}

	help := "j/↓: up • k/↑: down • q: quit"
	if m.Mode.Choice == "Relay" {
		help = "ctrl + c: quit"
	}

	return AppFrame(s.String(), help)
}

func AppFrame(content string, helpText string) string {
	if W == 0 || H == 0 {
		if err := termbox.Init(); err != nil {
			panic(err)
		}
		W, H = termbox.Size()
		termbox.Close()
	}

	titleStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Margin(1, 2).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color(Accent))
	title := titleStyle.Render("FT_0")

	frame := lipgloss.NewStyle().Padding(1, 0).Width(W).Height(H - 5)
	help := Container.Foreground(lipgloss.Color(Muted)).Render(helpText)

	return title + frame.Render(content) + "\n" + help
}
