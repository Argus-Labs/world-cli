package style

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func ContextPrint(title, titleColor, subject, object string) {
	titleStr := ForegroundPrint(title, titleColor)
	arrowStr := ForegroundPrint("→", "241")
	subjectStr := ForegroundPrint(subject, "4")

	fmt.Printf("%s %s %s %s ", titleStr, arrowStr, subjectStr, object)
}

func ForegroundPrint(text string, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(text)
}