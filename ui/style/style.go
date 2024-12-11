package style

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// TickIcon represents a success/completion indicator
	TickIcon = lipgloss.NewStyle().SetString("✓").Foreground(lipgloss.Color("2"))

	// CrossIcon represents a failure/error indicator
	CrossIcon = lipgloss.NewStyle().SetString("✗").Foreground(lipgloss.Color("1"))

	// Container style for UI containers
	Container = lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	// Icons for UI elements
	ChevronIcon      = lipgloss.NewStyle().SetString("❯").Foreground(lipgloss.Color("12"))
	DoubleRightIcon  = lipgloss.NewStyle().SetString("»").Foreground(lipgloss.Color("12"))
	QuestionIcon     = lipgloss.NewStyle().SetString("?").Foreground(lipgloss.Color("5"))

	// Text styles
	BoldText = lipgloss.NewStyle().Bold(true)
)

// CLIHeader returns a styled CLI header with title and subtitle
func CLIHeader(title, subtitle string) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		MarginBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	header := headerStyle.Render(title)
	if subtitle != "" {
		header += "\n" + subtitleStyle.Render(subtitle)
	}
	return header
}
