package style

import "github.com/charmbracelet/lipgloss"

var (
	QuestionIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("251")).SetString("? ").Bold(true)
	CrossIcon       = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).SetString("(FAIL) ").Bold(true)
	TickIcon        = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).SetString("(OK)   ").Bold(true)
	TodoIcon        = lipgloss.NewStyle().SetString("- ").Bold(true)
	DoubleRightIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("251")).SetString(">> ").Bold(true)
	ChevronIcon     = lipgloss.NewStyle().Foreground(lipgloss.Color("251")).SetString("> ").Bold(true)
)
