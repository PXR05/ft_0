package ui

import (
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

func InitialModel() Model {
	return Model{
		Mode:    InitialModeModel(),
		Send:    InitialSendModel(),
		Receive: InitialReceiveModel(),
		Relay:   NewRelayModel(),
	}
}

type ReturnToMenuMsg struct{}

const (
	Accent = "#ffffaf"
	Muted  = "#4d4d4d"
	Normal = "#dddddd"
	Err    = "#ff5f5f"
)

var (
	Container = lipgloss.NewStyle().Padding(1, 2)
)

var Error error

func (m Model) Init() tea.Cmd {
	return tea.SetWindowTitle("FT_0")
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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

	case ReturnToMenuMsg:
		switch m.Mode.Choice {
		case "Send":
			m.Send = InitialSendModel()
		case "Receive":
			m.Receive = InitialReceiveModel()
		case "Relay":
			m.Relay = NewRelayModel()
		}
		m.Mode.Choice = ""
		return m, nil
	}

	var cmd tea.Cmd
	switch m.Mode.Choice {
	case "Send":
		sendModel, cmd := m.Send.Update(msg)
		var ok bool
		m.Send, ok = sendModel.(SendModel)
		if !ok {
			panic("could not perform send model type assertion")
		}
		return m, cmd

	case "Receive":
		receiveModel, cmd := m.Receive.Update(msg)
		var ok bool
		m.Receive, ok = receiveModel.(ReceiveModel)
		if !ok {
			panic("could not perform receive model type assertion")
		}
		return m, cmd

	case "Relay":
		m.Relay, cmd = m.Relay.Update(msg)
		return m, cmd

	default:
		m.Mode, cmd = m.Mode.Update(msg)
		if m.Mode.Choice != "" {
			switch m.Mode.Choice {
			case "Send":
				m.Send = InitialSendModel()
				m.Send.width = m.width
				m.Send.height = m.height
				m.Send.filepicker.Height = m.height - 14
				m.Send.progress.Width = m.width - 20
				if cmd := m.Send.filepicker.Init(); cmd != nil {
					return m, cmd
				}

			case "Receive":
				m.Receive = InitialReceiveModel()
				m.Receive.width = m.width
				m.Receive.height = m.height
				m.Receive.progress.Width = m.width - 20
				m.Receive.sessionInput.Width = m.width - 20
				if cmd := m.Receive.Init(); cmd != nil {
					return m, cmd
				}

			case "Relay":
				m.Relay = NewRelayModel()
				if cmd := m.Relay.Init(); cmd != nil {
					return m, cmd
				}
			}
		}
		return m, cmd
	}
}

func (m Model) View() string {
	var s strings.Builder
	if Error != nil {
		s.WriteString(Error.Error())
	}

	switch m.Mode.Choice {
	case "Send":
		return m.Send.View()
	case "Receive":
		return m.Receive.View()
	case "Relay":
		return m.Relay.View()
	default:
		return m.Mode.View()
	}
}

func AppFrame(content string, helpText string, w, h int) string {
	titleStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Margin(1, 2).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color(Accent))
	title := titleStyle.Render("FT_0")

	helpStyle := Container.Foreground(lipgloss.Color(Muted))
	help := helpStyle.Render(helpText)

	contentHeight := h - lipgloss.Height(title) - lipgloss.Height(help) - 2

	frame := lipgloss.NewStyle().
		Width(w).
		Height(contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		frame.Render(content),
		help,
	)
}
