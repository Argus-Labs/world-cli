package style

import (
	"github.com/charmbracelet/lipgloss"
	"pkg.world.dev/world-cli/common/printer"
)

func ContextPrint(title, titleColor, subject, object string) {
	titleStr := ForegroundPrint(title, titleColor)
	arrowStr := ForegroundPrint("→", "241")
	subjectStr := ForegroundPrint(subject, "4")

	printer.Infof("%s %s %s %s ", titleStr, arrowStr, subjectStr, object)
}

func ForegroundPrint(text string, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(text)
}
