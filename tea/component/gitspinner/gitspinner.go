package gitspinner

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pkg.world.dev/world-cli/common/teacmd"
)

type Model struct {
	spinner spinner.Model
}

func New() Model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	s.Spinner = spinner.Points
	return Model{spinner: s}
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case teacmd.GitCloneProgressMsg:
		return m, m.spinner.Tick
	}
	return m, nil
}

func (m Model) View() string {
	return m.spinner.View()
}
