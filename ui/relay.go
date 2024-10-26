package ui

import (
	"ft_0/server"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type RelayModel struct {
	relay *server.RelayServer
}

func NewRelayModel() RelayModel {
	return RelayModel{}
}

func (m *RelayModel) Init() tea.Cmd {
	return nil
}

func (m *RelayModel) Update(msg tea.Msg) (RelayModel, tea.Cmd) {
	switch msg := msg.(type) {
	case server.RelayLogMsg:
		m.relay.Messages = append(m.relay.Messages, string(msg))
		return *m, server.CheckRelayLogs(m.relay)

	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			if m.relay == nil {
				s := server.NewRelayServer()
				go s.Start()
				m.relay = s
				return *m, server.CheckRelayLogs(m.relay)
			}
			if !m.relay.IsRunning {
				go m.relay.Start()
				return *m, server.CheckRelayLogs(m.relay)
			}
		}
	}

	return *m, nil
}

func (m *RelayModel) View() string {
	var s strings.Builder

	if m.relay != nil && m.relay.IsRunning {
		for _, msg := range m.relay.Messages {
			s.WriteString(Container.Render(msg + "\n"))
		}
	} else {
		s.WriteString(Container.Render("Press Enter to start relay server"))
	}

	return s.String()
}

func (m *RelayModel) Stop() {
	if m.relay != nil {
		m.relay.Stop()
		m.relay.Messages = nil
	}
}

func (m *RelayModel) IsRunning() bool {
	return m.relay != nil && m.relay.IsRunning
}
