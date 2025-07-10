package multispinner

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pkg.world.dev/world-cli/internal/pkg/tea/style"
)

// Spinner is a component that displays a spinner while updating the logs.
type MultiSpinner struct {
	processMap  *ProcessStateMap // need to be pointer because of the mutex
	processList []string         // list of process names

	spinner spinner.Model // spinner model
	cancel  func()        // cancel function for context cancellation
	allDone bool
}

type ProcessStateMap struct {
	sync.Mutex
	value map[string]ProcessState
}

type ProcessState struct {
	Icon   string
	State  string
	Type   string
	Name   string
	Detail string
	Done   bool
}

func CreateSpinner(processList []string, cancel func()) MultiSpinner {
	// Initialize the spinner
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	s.Spinner = spinner.Points

	// Initialize the process map
	processMap := &ProcessStateMap{
		value: make(map[string]ProcessState),
	}

	// put all processes in the map
	for _, process := range processList {
		processMap.value[process] = ProcessState{
			Name: process,
		}
	}

	return MultiSpinner{
		spinner:     s,
		processList: processList,
		processMap:  processMap,
		cancel:      cancel,
	}
}

// Init is called when the program starts and returns the initial command.
func (s MultiSpinner) Init() tea.Cmd {
	// Start the spinner
	return s.spinner.Tick
}

// Update handles incoming messages.
func (s MultiSpinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// case ctrl + c
		if msg.String() == "ctrl+c" {
			if s.cancel != nil {
				s.cancel()
			}
			return s, tea.Quit
		}
	case spinner.TickMsg:
		// Update the spinner
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		// If all processes are done, quit after the view is updated
		if s.allDone {
			return s, tea.Batch(cmd, tea.Quit)
		}
		return s, cmd
	case ProcessState:
		s.setState(msg.Name, msg)

		// check if all processes are done
		allDone := true
		for _, state := range s.getStates() {
			if !state.Done {
				allDone = false
				break
			}
		}

		if allDone {
			// Set the flag to indicate all processes are done
			s.allDone = true
			// Return a spinner tick to update the view one last time before quitting
			return s, s.spinner.Tick
		}
	}

	return s, nil
}

// View renders the UI.
func (s MultiSpinner) View() string {
	text := ""

	processStates := s.getStates()

	for _, state := range processStates {
		icon := state.Icon
		if !state.Done {
			icon = s.spinner.View()
		}

		text += fmt.Sprintf("%s %s %s %s %s", icon,
			style.ForegroundPrint(state.State, "12"),
			style.ForegroundPrint(state.Type, "13"),
			style.ForegroundPrint(state.Name, "2"),
			state.Detail)

		text += "\n"
	}

	return text
}

func (s MultiSpinner) setState(process string, state ProcessState) {
	s.processMap.Lock()
	defer s.processMap.Unlock()

	s.processMap.value[process] = state
}

func (s MultiSpinner) getStates() []ProcessState {
	s.processMap.Lock()
	defer s.processMap.Unlock()

	states := make([]ProcessState, 0, len(s.processMap.value))
	for _, process := range s.processList {
		states = append(states, s.processMap.value[process])
	}

	return states
}
