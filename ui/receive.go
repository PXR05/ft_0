package ui

import (
	"fmt"
	"ft_0/server"
	"net"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ReceiveModel struct {
	sessionInput  textinput.Model
	sessionId     string
	err           error
	transferState TransferStatus
	progressChan  chan server.TransferProgress
	cancelChan    chan struct{}
	progress      progress.Model
	width         int
}

type TransferStatus struct {
	Progress float64
	Speed    float64
	State    server.TransferState
	Error    error
}

type transferMsg server.TransferProgress

var (
	conn      *net.Conn
	metadata  server.FileMetadata
	selected  string
	confirmed string
)

func resetState() {
	conn = nil
	metadata = server.FileMetadata{}
	selected = ""
	confirmed = ""
}
func InitialReceiveModel() ReceiveModel {
	resetState()
	return ReceiveModel{
		sessionInput: CreateSessionInput(),
		transferState: TransferStatus{
			State: server.StateInitializing,
		},
		progress:   progress.New(progress.WithSolidFill(Accent)),
		cancelChan: make(chan struct{}),
	}
}

func (m ReceiveModel) Init() tea.Cmd {
	return textinput.Blink
}

func listenForTransferProgress(sub chan server.TransferProgress) tea.Cmd {
	return func() tea.Msg {
		progress := <-sub
		if progress.Error != nil {
			return transferMsg(progress)
		}
		return transferMsg(progress)
	}
}

func (m ReceiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 20
		return m, nil

	case transferMsg:
		m.transferState.Progress = float64(msg.BytesReceived) / float64(metadata.Size)
		m.transferState.Speed = msg.Speed
		m.transferState.Error = msg.Error
		m.transferState.State = msg.State

		return m, listenForTransferProgress(m.progressChan)

	case tea.KeyMsg:
		if m.transferState.State == server.StateCompleted || m.transferState.State == server.StateCancelled || m.transferState.State == server.StateError {
			err := server.LeaveSession(m.sessionId)
			if err != nil {
				m.err = err
			}
			resetState()
			return m, tea.Quit
		}
		if msg.Type == tea.KeyCtrlC {
			if m.transferState.State == server.StateReceiving {
				close(m.cancelChan)
				return m, nil
			}
			resetState()
			return m, tea.Quit
		}
		if m.transferState.State == server.StateCancelled {
			resetState()
			return m, tea.Quit
		}
		if metadata != (server.FileMetadata{}) {
			if msg.Type == tea.KeyEnter {
				if selected == "n" || selected == "N" {
					confirmed = "n"
					m.transferState.State = server.StateCancelled
					return m, nil
				} else if selected == "y" || selected == "Y" || selected == "" {
					confirmed = "y"
					m.progressChan = make(chan server.TransferProgress)
					server.ReceiveFile(*conn, metadata, m.progressChan, m.cancelChan)
					return m, listenForTransferProgress(m.progressChan)
				}
			}
			selected = msg.String()
			return m, nil
		}
		if msg.Type == tea.KeyEnter {
			m.sessionId = m.sessionInput.Value()
		}
	}

	m.sessionInput, cmd = m.sessionInput.Update(msg)
	return m, cmd
}

func createView(m ReceiveModel) string {
	textHighlight := lipgloss.NewStyle().Foreground(lipgloss.Color(Accent))
	metaString := fmt.Sprintf(
		("Filename : %s\n" +
			"Size     : %s\n" +
			"From     : %s\n\n"),
		textHighlight.Render(metadata.Name),
		textHighlight.Render(fmt.Sprintf("%d bytes", metadata.Size)),
		textHighlight.Render(metadata.SenderIP),
	)

	switch m.transferState.State {
	case server.StateError:
		return fmt.Sprintf("Error: %v\n\nPress any key to continue\n", m.transferState.Error)

	case server.StateCancelled:
		return metaString + "Transfer cancelled\n\nPress any key to continue\n"

	case server.StateCompleted:
		return metaString + "File received\n\nPress any key to continue\n"

	case server.StateReceiving:
		progressBar := m.progress.ViewAs(m.transferState.Progress)
		return metaString + fmt.Sprintf(
			"%s\n(%.2f MB/s)\n",
			progressBar,
			m.transferState.Speed,
		)
	}

	if m.sessionId != "" {
		if metadata == (server.FileMetadata{}) || conn == nil {
			cn, err := server.StartReceiver(m.sessionId)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}

			conn = &cn
			if metadata == (server.FileMetadata{}) {
				meta, err := server.ReceiveMetadata(cn)
				if err != nil {
					return fmt.Sprintf("Error: %v", err)
				}
				metadata = meta
			}
		}

		metaString := fmt.Sprintf(
			("Filename : %s\n" +
				"Size     : %s\n" +
				"From     : %s\n\n"),
			textHighlight.Render(metadata.Name),
			textHighlight.Render(fmt.Sprintf("%d bytes", metadata.Size)),
			textHighlight.Render(metadata.SenderIP),
		)
		if confirmed == "" {
			return metaString + fmt.Sprintf(
				"Accept file? (Y/n): %s\n",
				textHighlight.Render(selected),
			)
		}
	}
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(Accent))
	return fmt.Sprintf(
		"Input sender session token\n\n%s\n",
		inputStyle.Render(m.sessionInput.View()),
	)
}

func (m ReceiveModel) View() string {
	m.sessionInput.Focus()
	return AppFrame(Container.Render(createView(m)), "ctrl + c: quit")
}

func CreateSessionInput() textinput.Model {
	input := textinput.New()
	input.Focus()
	input.Placeholder = "Session ID"
	input.CharLimit = 6
	input.Width = 10

	return input
}
