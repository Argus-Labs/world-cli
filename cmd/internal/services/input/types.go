package input

import (
	"context"
	"io"
)

// ServiceInterface defines methods for handling user input.
type ServiceInterface interface {
	// Prompt displays a prompt and returns user input
	Prompt(ctx context.Context, prompt, defaultValue string) (string, error)

	// Confirm asks for Y/n confirmation with default
	Confirm(ctx context.Context, prompt, defaultValue string) (bool, error)

	// Select allows user to select from multiple options by number
	Select(ctx context.Context, title, prompt string, options []string, defaultIndex int) (int, error)

	// SelectString allows user to select from multiple options, returns the selected string
	SelectString(ctx context.Context, title, prompt string, options []string, defaultValue string) (string, error)
}

// Service implements the input interface using standard input/output.
type Service struct {
	Input  io.Reader // Allows injection of different input sources for testing
	Output io.Writer // Allows injection of different output destinations for testing
}
