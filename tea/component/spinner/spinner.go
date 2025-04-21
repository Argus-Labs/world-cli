package teaspinner

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Spinner is a component that displays a spinner while updating the logs.
type Spinner struct {
	Spinner spinner.Model
	Cancel  func()

	text string
	done bool
}

type LogMsg string

// Init is called when the program starts and returns the initial command.
func (s Spinner) Init() tea.Cmd {
	// Start the spinner
	return s.Spinner.Tick
}

// Update handles incoming messages.
func (s Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// case ctrl + c
		if msg.String() == "ctrl+c" {
			s.Cancel()
			return s, tea.Quit
		}
	case spinner.TickMsg:
		// Update the spinner
		var cmd tea.Cmd
		s.Spinner, cmd = s.Spinner.Update(msg)
		return s, cmd
	case LogMsg:
		// Add the log message to the list of logs and return a spinner tick
		s.text = string(msg)
		if string(msg) == "spin: completed" {
			s.done = true
			return s, tea.Quit
		}
		return s, s.Spinner.Tick
	}

	return s, nil
}

// View renders the UI.
func (s Spinner) View() string {
	if s.done {
		return "Build completed!"
	}

	return fmt.Sprintf("%s %s", s.Spinner.View(), s.text)
}
