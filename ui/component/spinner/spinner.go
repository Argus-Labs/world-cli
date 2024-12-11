package spinner

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner is a component that displays a loading spinner
type Spinner struct {
	frames []string
	speed  time.Duration
	index  int
	style  lipgloss.Style
}

// New creates a new spinner instance
func New() *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		speed:  time.Millisecond * 80,
		style:  lipgloss.NewStyle(),
	}
}

// View returns the current frame of the spinner
func (s *Spinner) View() string {
	return s.style.Render(s.frames[s.index])
}

// Update advances the spinner animation
func (s *Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return s, nil
	default:
		s.index = (s.index + 1) % len(s.frames)
		return s, tea.Tick(s.speed, func(_ time.Time) tea.Msg {
			return nil
		})
	}
}

// Init initializes the spinner
func (s *Spinner) Init() tea.Cmd {
	return tea.Tick(s.speed, func(_ time.Time) tea.Msg {
		return nil
	})
}

// WithStyle sets the style for the spinner
func (s *Spinner) WithStyle(style lipgloss.Style) *Spinner {
	s.style = style
	return s
}
