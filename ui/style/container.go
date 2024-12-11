package style

import (
	"github.com/charmbracelet/lipgloss"
	"pkg.world.dev/world-cli/ui"
)

var (
	Container = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1,
		ui.ContainerPadding).BorderForeground(lipgloss.Color("#874BFD"))
	cliHeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(0, ui.ContainerPadding).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			Bold(true).
			Italic(true).
			Align(lipgloss.Center).
			Width(ui.HeaderWidth)
)

func CLIHeader(title string, description string) string {
	return cliHeaderStyle.Render(title) + "\n" + description
}
