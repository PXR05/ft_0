// ui/sender.go
package ui

import (
	"context"
	"fmt"
	"ft_0/server"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type senderMsg server.SendProgress

type SendModel struct {
	filepicker    filepicker.Model
	selectedFile  string
	progress      progress.Model
	quitting      bool
	err           error
	transferState server.TransferState
	sessionID     string
	speed         float64
	bytesSent     int64
	totalBytes    int64
	progressChan  chan server.SendProgress
	cancel        context.CancelFunc
}

func InitialSendModel() SendModel {
	return SendModel{
		filepicker:    CreateFilepicker(),
		progress:      progress.New(progress.WithSolidFill(Accent)),
		transferState: server.StateInitializing,
	}
}

func (m SendModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func listenForSenderProgress(sub chan server.SendProgress) tea.Cmd {
	return func() tea.Msg {
		progress := <-sub
		if progress.Error != nil {
			return senderMsg(progress)
		}
		return senderMsg(progress)
	}
}

func (m SendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 20
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter && (m.transferState == server.StateCompleted || m.transferState == server.StateError) {
			m.quitting = true
			return m, tea.Quit
		}
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			if m.cancel != nil {
				m.cancel()
				m.transferState = server.StateCancelled
			}
			m.quitting = true
			return m, tea.Quit
		}

	case senderMsg:
		m.transferState = msg.State
		m.sessionID = msg.SessionID
		m.speed = msg.Speed
		m.bytesSent = msg.BytesSent
		m.totalBytes = msg.TotalBytes

		return m, listenForSenderProgress(m.progressChan)
	}

	if m.selectedFile != "" && m.progressChan == nil {
		progressChan := make(chan server.SendProgress)
		m.progressChan = progressChan
		server.StartSender(m.selectedFile, progressChan, context.Background())
		return m, listenForSenderProgress(progressChan)
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = path
	}

	return m, cmd
}

func (m SendModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder
	emphasis := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(Accent))
	help := ""

	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.selectedFile == "" {
		help = "j/↓: up • k/↑: down • l/→: open • h/←: back • q: quit"
		s.WriteString("Pick a file")
		s.WriteString("\n\n" + m.filepicker.View())
	} else {
		s.WriteString(emphasis.Render(m.selectedFile))
		s.WriteString("\n\n")
		help = "q: quit"
		switch m.transferState {
		case server.StateInitializing:
			s.WriteString("Press any key to initialize transfer\n")

		case server.StateWaitingForReceiver:
			s.WriteString(fmt.Sprintf("Your session ID is: %s\n", emphasis.Render(m.sessionID)))
			s.WriteString("Share this ID with the receiver to start the transfer\n\n")
			s.WriteString("Waiting for receiver to join...\n")

		case server.StateTransferring:
			progress := float64(m.bytesSent) / float64(m.totalBytes)
			progressBar := m.progress.ViewAs(progress)
			s.WriteString(fmt.Sprintf("%s\n", progressBar))
			s.WriteString(fmt.Sprintf("%.2f MB/s\n", m.speed))

		case server.StateCompleted:
			s.WriteString("Transfer completed successfully\n\nPress enter to continue")

		case server.StateCancelled:
			s.WriteString("Transfer cancelled by user\n\nPress enter to continue")

		case server.StateError:
			s.WriteString(fmt.Sprintf("Error: %v\n\nPress any key to continue", m.err))
		}
	}

	return AppFrame(Container.Render(s.String()), help)
}

func CreateFilepicker() filepicker.Model {
	fp := filepicker.New()
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.Styles = filepicker.Styles{
		Cursor:         lipgloss.NewStyle().Foreground(lipgloss.Color(Accent)),
		Symlink:        lipgloss.NewStyle().Foreground(lipgloss.Color("#5fffaf")),
		Directory:      lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd787")),
		File:           lipgloss.NewStyle(),
		Permission:     lipgloss.NewStyle().Foreground(lipgloss.Color(Muted)),
		Selected:       lipgloss.NewStyle().Foreground(lipgloss.Color(Accent)).Bold(true),
		FileSize:       lipgloss.NewStyle().Foreground(lipgloss.Color(Muted)).Width(8).Align(lipgloss.Right),
		EmptyDirectory: lipgloss.NewStyle().Foreground(lipgloss.Color(Muted)).SetString("No Files Found."),
	}

	return fp
}
