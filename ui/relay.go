package ui

import (
	"ft_0/server"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type RelayModel struct {
	relay  *server.RelayServer
	width  int
	height int
}

func NewRelayModel() RelayModel {
	return RelayModel{}
}

func (m RelayModel) Init() tea.Cmd {
	return nil
}

func (m RelayModel) Update(msg tea.Msg) (RelayModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case server.RelayLogMsg:
		m.relay.Messages = append(m.relay.Messages, string(msg))
		return m, server.CheckRelayLogs(m.relay)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			if m.relay != nil {
				m.relay.Stop()
				m.relay.Messages = nil
			}
			return m, func() tea.Msg {
				return ReturnToMenuMsg{}
			}
		}
		if msg.Type == tea.KeyEnter {
			if m.relay == nil {
				s := server.NewRelayServer()
				go s.Start()
				m.relay = s
				return m, server.CheckRelayLogs(m.relay)
			}
			if !m.relay.IsRunning {
				go m.relay.Start()
				return m, server.CheckRelayLogs(m.relay)
			}
		}
	}

	return m, nil
}

func (m RelayModel) View() string {
	var s strings.Builder

	contentHeight := m.height - 6

	if m.relay != nil && m.relay.IsRunning {
		messageStyle := lipgloss.NewStyle().
			MaxHeight(contentHeight).
			MaxWidth(m.width - 4)
		messages := strings.Join(m.relay.Messages, "\n")
		s.WriteString(Container.Render(messageStyle.Render(messages)))
	} else {
		s.WriteString(Container.Render("Press Enter to start relay server"))
	}

	return AppFrame(s.String(), "q: quit", m.width, m.height)
}

func (m RelayModel) Stop() {
	if m.relay != nil {
		m.relay.Stop()
		m.relay.Messages = nil
	}
}

func (m *RelayModel) IsRunning() bool {
	return m.relay != nil && m.relay.IsRunning
}
