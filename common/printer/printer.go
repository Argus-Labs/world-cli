package printer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guumaster/logsymbols"
)

var (
	headerStyle       = lipgloss.NewStyle().Bold(true).Underline(true)
	successStyle      = lipgloss.NewStyle().Bold(true)
	errorStyle        = lipgloss.NewStyle().Bold(true)
	notificationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("178")) // Bright yellow, good for notifications
)

func Success(msg string) {
	//nolint:forbidigo // Need customer friendly output
	fmt.Print(successStyle.Render(string(logsymbols.Success) + " " + msg))
}

func Successln(msg string) {
	//nolint:forbidigo // Need customer friendly output
	fmt.Println(successStyle.Render(string(logsymbols.Success) + " " + msg))
}

func Successf(format string, args ...any) {
	msg := successStyle.Render(string(logsymbols.Success) + " " + fmt.Sprintf(format, args...))
	fmt.Print(msg) //nolint:forbidigo // Need customer friendly output
}

func Error(msg string) {
	//nolint:forbidigo // Need customer friendly output
	fmt.Print(errorStyle.Render(string(logsymbols.Error) + " " + msg))
}

func Errorln(msg string) {
	//nolint:forbidigo // Need customer friendly output
	fmt.Println(errorStyle.Render(string(logsymbols.Error) + " " + msg))
}

func Errorf(format string, args ...any) {
	msg := errorStyle.Render(string(logsymbols.Error) + " " + fmt.Sprintf(format, args...))
	fmt.Print(msg) //nolint:forbidigo // Need customer friendly output
}

// Info prints a message with newline.
func Info(msg string) {
	fmt.Print(msg) //nolint:forbidigo // Need customer friendly output
}

func Infoln(msg string) {
	fmt.Println(msg) //nolint:forbidigo // Need customer friendly output
}

func Infof(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(msg) //nolint:forbidigo // Need customer friendly output
}

func Header(msg string) {
	fmt.Print(headerStyle.Render(msg)) //nolint:forbidigo // Need customer friendly output
}

func Headerln(msg string) {
	fmt.Println(headerStyle.Render(msg)) //nolint:forbidigo // Need customer friendly output
}

func Headerf(format string, args ...any) {
	msg := headerStyle.Render(fmt.Sprintf(format, args...))
	fmt.Print(msg) //nolint:forbidigo // Need customer friendly output
}

func Notification(msg string) {
	fmt.Print(notificationStyle.Render(msg)) //nolint:forbidigo // Need customer friendly output
}

func Notificationf(format string, args ...any) {
	msg := notificationStyle.Render(fmt.Sprintf(format, args...))
	fmt.Print(msg) //nolint:forbidigo // Need customer friendly output
}

func Notificationln(msg string) {
	fmt.Println(notificationStyle.Render(msg)) //nolint:forbidigo // Need customer friendly output
}

func NewLine(numberOfLines int) {
	if numberOfLines <= 0 {
		numberOfLines = 1
	}
	fmt.Print(strings.Repeat("\n", numberOfLines)) //nolint:forbidigo // Need customer friendly output
}

// SectionDivider prints a divider line of a given symbol and length.
// Default length is 1.
func SectionDivider(symbol string, length int) {
	if length <= 0 {
		length = 1
	}
	fmt.Println(strings.Repeat(symbol, length)) //nolint:forbidigo // Need customer friendly output
}
