package ui

import (
	"fmt"
	"ft_0/server"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Mode    ModeModel
	Receive ReceiveModel
	Send    SendModel
	Relay   RelayModel
	width   int
	height  int
}

const (
	Accent = "#ffffaf"
	Muted  = "#4d4d4d"
	Normal = "#dddddd"
	Err    = "#ff5f5f"
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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

	return AppFrame(s.String(), help, m.width, m.height)
}

func AppFrame(content string, helpText string, w, h int) string {
	titleStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Margin(1, 2).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color(Accent))
	title := titleStyle.Render("FT_0")

	contentHeight := h - lipgloss.Height(title) - lipgloss.Height(helpText)

	frame := lipgloss.NewStyle().
		Padding(1, 0).
		Width(w).
		Height(contentHeight)

	help := Container.Foreground(lipgloss.Color(Muted)).Render(helpText)

	return title + frame.Render(content) + "\n" + help
}
