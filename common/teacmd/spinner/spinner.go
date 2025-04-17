package spinner

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error
type updateMsg string

type Model struct {
	spinner  spinner.Model
	quitting bool
	err      error
	message  string
	msgChan  chan string
}

func New(message string, msgChan chan string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return Model{
		spinner: s,
		message: message,
		msgChan: msgChan,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			if m.msgChan == nil {
				return nil
			}
			select {
			case msg := <-m.msgChan:
				return updateMsg(msg)
			default:
				return nil
			}
		},
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, nil

	case updateMsg:
		m.message = string(msg)
		return m, func() tea.Msg {
			if m.msgChan == nil {
				return nil
			}
			select {
			case msg := <-m.msgChan:
				return updateMsg(msg)
			default:
				return nil
			}
		}

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m Model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	str := fmt.Sprintf("\r%s %s", m.spinner.View(), m.message)
	if m.quitting {
		return str + "\n"
	}
	return str
}

// RunWithContext runs the spinner with a context that can be cancelled
func RunWithContext(ctx context.Context, message string, msgChan chan string) error {
	p := tea.NewProgram(New(message, msgChan), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run spinner: %w", err)
	}
	return nil
}

// Run runs the spinner without a context
func Run(message string, msgChan chan string) error {
	return RunWithContext(context.Background(), message, msgChan)
}
