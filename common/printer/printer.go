//nolint:forbidigo // Printer is used for customer friendly output to terminal
package printer

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guumaster/logsymbols"
	"github.com/muesli/termenv"
)

//nolint:gochecknoglobals // read only, initialize objects once for performance.
var (
	successStyle      = lipgloss.NewStyle().Bold(true)
	errorStyle        = lipgloss.NewStyle().Bold(true)
	headerStyle       = lipgloss.NewStyle().Bold(true).Underline(true)
	notificationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("178")) // Bright yellow, good for notifications
)

func Success(msg string) {
	fmt.Print(successStyle.Render(string(logsymbols.Success) + " " + msg))
}

func Successln(msg string) {
	fmt.Println(successStyle.Render(string(logsymbols.Success) + " " + msg))
}

func Successf(format string, args ...any) {
	newFormat, linesRemoved := trimAndCountTrailingNewlines(format)
	msg := successStyle.Render(string(logsymbols.Success) + " " + fmt.Sprintf(newFormat, args...))
	fmt.Print(msg)
	NewLine(linesRemoved)
}

func Error(msg string) {
	fmt.Print(errorStyle.Render(string(logsymbols.Error) + " " + msg))
}

func Errorln(msg string) {
	fmt.Println(errorStyle.Render(string(logsymbols.Error) + " " + msg))
}

func Errorf(format string, args ...any) {
	newFormat, linesRemoved := trimAndCountTrailingNewlines(format)
	msg := errorStyle.Render(string(logsymbols.Error) + " " + fmt.Sprintf(newFormat, args...))
	fmt.Print(msg)
	NewLine(linesRemoved)
}

func Info(msg string) {
	fmt.Print(msg)
}

func Infoln(msg string) {
	fmt.Println(msg)
}

func Infof(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(msg)
}

func Header(msg string) {
	fmt.Print(headerStyle.Render(msg))
}

func Headerln(msg string) {
	fmt.Println(headerStyle.Render(msg))
}

func Headerf(format string, args ...any) {
	newFormat, linesRemoved := trimAndCountTrailingNewlines(format)
	msg := headerStyle.Render(fmt.Sprintf(newFormat, args...))
	fmt.Print(msg)
	NewLine(linesRemoved)
}

func Notification(msg string) {
	fmt.Print(notificationStyle.Render(msg))
}

func Notificationf(format string, args ...any) {
	newFormat, linesRemoved := trimAndCountTrailingNewlines(format)
	msg := notificationStyle.Render(fmt.Sprintf(newFormat, args...))
	fmt.Print(msg)
	NewLine(linesRemoved)
}

func Notificationln(msg string) {
	fmt.Println(notificationStyle.Render(msg))
}

func NewLine(numberOfLines int) {
	if numberOfLines <= 0 {
		return
	}
	fmt.Print(strings.Repeat("\n", numberOfLines))
}

func MoveCursorUp(numberOfLines int) {
	output := termenv.NewOutput(os.Stdout)
	output.CursorUp(numberOfLines)
}

func MoveCursorRight(numberOfCells int) {
	output := termenv.NewOutput(os.Stdout)
	output.CursorForward(numberOfCells)
}

func ClearToEndOfLine() {
	output := termenv.NewOutput(os.Stdout)
	output.ClearLineRight()
}

// SectionDivider prints a divider line of a given symbol and length.
// Default length is 1.
func SectionDivider(symbol string, length int) {
	if length <= 0 {
		length = 1
	}
	fmt.Println(strings.Repeat(symbol, length))
}

// trimAndCountTrailingNewlines trims trailing newlines from a string and returns the count.
// Used for sylized output to ensure the cursor is reset properly.
func trimAndCountTrailingNewlines(s string) (string, int) {
	if s == "" {
		return "", 0
	}

	count := 0
	i := len(s)
	for i > 0 && s[i-1] == '\n' {
		i--
		count++
	}
	return s[:i], count
}
