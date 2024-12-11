package steps

import tea "github.com/charmbracelet/bubbletea"

/////////////////////////
// Bubble Tea Commands //
/////////////////////////

type StartMsg struct{}
type CompleteStepMsg struct {
	Err error
}
type SignalStepStartedMsg struct {
	Index int
}
type SignalStepCompletedMsg struct {
	Index int
}
type SignalStepErrorMsg struct {
	Index int
	Err   error
}
type SignalAllStepCompletedMsg struct{}

// StartCommand starts the step component
func (m Model) StartCommand() tea.Cmd {
	return func() tea.Msg {
		return StartMsg{}
	}
}

// CompleteStepCommand sets the current step as completed
func (m Model) CompleteStepCommand(err error) tea.Cmd {
	return func() tea.Msg {
		return CompleteStepMsg{Err: err}
	}
}

// SignalStepStartedCommand signals the start of a step
// This is useful for when you want to trigger a certain action when a certain step starts
func (m Model) SignalStepStartedCommand(index int) tea.Cmd {
	return func() tea.Msg {
		return SignalStepStartedMsg{Index: index}
	}
}

// SignalStepCompletedCommand signals the finish of a step
// This is useful for when you want to trigger a certain action when a certain step finishes
func (m Model) SignalStepCompletedCommand(index int) tea.Cmd {
	return func() tea.Msg {
		return SignalStepCompletedMsg{Index: index}
	}
}

// SignalStepErrorCommand signals the finish of a step with an error
// This is useful for when you want to trigger a certain action when a certain step finishes with an error
func (m Model) SignalStepErrorCommand(index int, err error) tea.Cmd {
	return func() tea.Msg {
		return SignalStepErrorMsg{Index: index, Err: err}
	}
}

// SignalAllStepCompletedCommand signals the finish of all steps
func (m Model) SignalAllStepCompletedCommand() tea.Cmd {
	return func() tea.Msg {
		return SignalAllStepCompletedMsg{}
	}
}
