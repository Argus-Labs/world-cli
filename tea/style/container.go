package style

import "github.com/charmbracelet/lipgloss"

var (
	Container = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("#874BFD"))
)

var cliHeaderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#874BFD")).
	Padding(0, 2).
	BorderTop(true).
	BorderLeft(true).
	BorderRight(true).
	BorderBottom(true).
	Bold(true).
	Italic(true).
	Align(lipgloss.Center).
	Width(40)

func CLIHeader(title string, description string) string {
	return cliHeaderStyle.Render(title) + "\n" + description
}
