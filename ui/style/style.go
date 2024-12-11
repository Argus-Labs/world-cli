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
)
