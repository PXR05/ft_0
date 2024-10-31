package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nsf/termbox-go"
)

func ModeItemStyles() (s list.DefaultItemStyles) {
	s.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(Normal)).
		Padding(0, 0, 0, 2)

	s.NormalDesc = s.NormalTitle.
		Foreground(lipgloss.Color(Muted))

	s.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color(Accent)).
		Foreground(lipgloss.Color(Accent)).
		Padding(0, 0, 0, 1).
		Bold(true)

	s.SelectedDesc = s.SelectedTitle.
		Foreground(lipgloss.Color(Accent)).
		Bold(false)

	return s
}

type ModeItem struct {
	ModeName, ModeDesc string
}

func (i ModeItem) Title() string       { return i.ModeName }
func (i ModeItem) Description() string { return i.ModeDesc }
func (i ModeItem) FilterValue() string { return "" }

type ModeModel struct {
	ModeList list.Model
	Choice   string
	width    int
	height   int
}

func InitialModeModel() ModeModel {
	return ModeModel{
		ModeList: CreateModeList(),
	}
}

func (m ModeModel) Init() tea.Cmd {
	return nil
}

func (m ModeModel) Update(msg tea.Msg) (ModeModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ModeList.SetWidth(msg.Width)
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			i, ok := m.ModeList.SelectedItem().(ModeItem)
			if ok {
				m.Choice = i.ModeName
			}
			return m, nil
		}
		m.ModeList, cmd = m.ModeList.Update(msg)
	}

	return m, cmd
}

func (m ModeModel) View() string {
	return AppFrame("\n"+m.ModeList.View(), "j/↓: up • k/↑: down • q: quit", m.width, m.height)
}

func CreateModeList() list.Model {
	items := []list.Item{
		ModeItem{ModeName: "Send", ModeDesc: "Send a file to a receiver"},
		ModeItem{ModeName: "Receive", ModeDesc: "Receive a file from a sender"},
		ModeItem{ModeName: "Relay", ModeDesc: "Start a relay server"},
	}

	if err := termbox.Init(); err != nil {
		panic(err)
	}
	w, _ := termbox.Size()
	termbox.Close()

	delegate := list.NewDefaultDelegate()
	delegate.Styles = ModeItemStyles()

	l := list.New(items, delegate, w, len(items)*3+1)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)

	return l
}
