package multiselect

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the UI state for the region selection menu
type Model struct {
	Items    []string
	Cursor   int
	Selected map[int]bool
	Ctx      context.Context
}

// InitialMultiselectModel creates a new Model with the given items and context
func InitialMultiselectModel(ctx context.Context, items []string) Model {
	return Model{
		Items:    items,
		Selected: make(map[int]bool),
		Ctx:      ctx,
	}
}

// Init initializes the bubbletea model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the model state accordingly
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	select {
	case <-m.Ctx.Done():
		return m, tea.Quit
	default:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up", "k":
				if m.Cursor > 0 {
					m.Cursor--
				}
			case "down", "j":
				if m.Cursor < len(m.Items)-1 {
					m.Cursor++
				}
			case " ":
				m.Selected[m.Cursor] = !m.Selected[m.Cursor]
			case "enter":
				return m, tea.Quit
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}
		return m, nil
	}
}

// View renders the current state of the region selection menu
func (m Model) View() string {
	s := "Choose regions (space to select/unselect, enter when done):\n\n"

	for i, item := range m.Items {
		cursor := " "
		if m.Cursor == i {
			cursor = ">"
		}

		checked := " "
		if m.Selected[i] {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, item)
	}

	s += "\n(press q to quit)\n"

	return s
}
