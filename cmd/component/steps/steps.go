package steps

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pkg.world.dev/world-cli/cmd/style"
)

const (
	INCOMPLETE = iota
	COMPLETE
	FAILED
)

type Model struct {
	index   int
	Steps   []Entry
	spinner spinner.Model
}

type Entry struct {
	Text   string
	Status int
	Err    error
}

type StartMsg struct {
}

// StartCmd starts the step component
func (m Model) StartCmd() tea.Cmd {
	return func() tea.Msg {
		return StartMsg{}
	}
}

type SignalFinishMsg struct {
}

func (m Model) SignalFinishCmd() tea.Cmd {
	return func() tea.Msg {
		return SignalFinishMsg{}
	}
}

type CompleteStepMsg struct {
	Err error
}

// CompleteStepCmd set the current step as completed
func (m Model) CompleteStepCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return CompleteStepMsg{Err: err}
	}
}

type SignalStepStartedMsg struct {
	Index int
}

// SignalStepStartedCmd signals the start of a step
// This is useful for when you want to trigger a certain action when a certain step starts
func (m Model) SignalStepStartedCmd(index int) tea.Cmd {
	return func() tea.Msg {
		return SignalStepStartedMsg{Index: index}
	}
}

type SignalStepCompletedMsg struct {
	Index int
}

// SignalStepCompletedCmd signals the finish of a step
// This is useful for when you want to trigger a certain action when a certain step finishes
func (m Model) SignalStepCompletedCmd(index int) tea.Cmd {
	return func() tea.Msg {
		return SignalStepCompletedMsg{Index: index}
	}
}

type SignalStepErrorMsg struct {
	Index int
	Err   error
}

// SignalStepErrorCmd signals the finish of a step with an error
// This is useful for when you want to trigger a certain action when a certain step finishes with an error
func (m Model) SignalStepErrorCmd(index int, err error) tea.Cmd {
	return func() tea.Msg {
		return SignalStepErrorMsg{Index: index, Err: err}
	}
}

func (m Model) View() string {
	output := ""
	for i, step := range m.Steps {
		icon := ""
		switch step.Status {
		case INCOMPLETE:
			// If this is the current step, show the spinner
			// Otherwise, show the to do icon.
			if i == m.index {
				icon = m.spinner.View()
			} else {
				icon = style.TodoIcon.Render()
			}
		case COMPLETE:
			icon = style.TickIcon.Render()
		case FAILED:
			icon = style.CrossIcon.Render()
		}

		output += fmt.Sprint(icon, step.Text)
		if i != len(m.Steps)-1 {
			output += "\n"
		}
	}
	return style.Container.Render(output)
}

func NewStep(text string) Entry {
	return Entry{
		Text:   text,
		Status: INCOMPLETE,
		Err:    nil,
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StartMsg:
		return m, m.spinner.Tick
	case CompleteStepMsg:
		if msg.Err != nil {
			m.Steps[m.index].Status = FAILED
			m.Steps[m.index].Err = msg.Err
			// Send a signal that the step has failed with an error
			return m, m.SignalStepErrorCmd(m.index, msg.Err)
		}

		m.Steps[m.index].Status = COMPLETE
		// If this is the last step, then we're done
		// Otherwise, we move on to the next step
		if m.index == len(m.Steps)-1 {
			// Send a signal that all the steps have been finished
			return m, tea.Sequence(m.SignalStepCompletedCmd(m.index), m.SignalFinishCmd())
		} else {
			m.index++
			// Send a signal that the current step has been completed and the next step has started
			return m, tea.Sequence(m.SignalStepCompletedCmd(m.index-1), m.SignalStepStartedCmd(m.index))
		}
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		index:   0,
		Steps:   []Entry{},
		spinner: s,
	}
}
