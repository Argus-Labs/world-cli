package input

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrInputCanceled = eris.New("input canceled")
	ErrInvalidInput  = eris.New("invalid input")
)

// NewService creates a new input service with standard stdin/stdout.
func NewService() Service {
	return Service{
		Input:  nil, // Will use os.Stdin if nil
		Output: nil, // Will use os.Stdout if nil
	}
}

// NewTestService creates a new input service for testing with custom input/output.
func NewTestService(input io.Reader, output io.Writer) Service {
	return Service{
		Input:  input,
		Output: output,
	}
}

// Prompt displays a prompt and returns user input.
func (s *Service) Prompt(ctx context.Context, prompt, defaultValue string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Display prompt
		if prompt != "" {
			s.printf("%s", prompt)
		}
		if defaultValue != "" {
			s.printf(" [%s]: ", defaultValue)
		} else {
			s.printf(": ")
		}

		// Read input
		input, err := s.readLine()
		if err != nil {
			return "", eris.Wrap(err, "failed to read input")
		}

		input = strings.TrimSpace(input)
		if input == "" && defaultValue != "" {
			// Display the default value as if they typed it in
			s.moveCursorUp(1)
			s.moveCursorRight(len(defaultValue) + 4 + len(prompt))
			s.println(defaultValue)
			return defaultValue, nil
		}
		return input, nil
	}
}

// Confirm asks for Y/n confirmation with default.
func (s *Service) Confirm(ctx context.Context, prompt, defaultValue string) (bool, error) {
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			input, err := s.Prompt(ctx, prompt, defaultValue)
			if err != nil {
				return false, err
			}

			switch strings.ToLower(input) {
			case "y", "yes":
				return true, nil
			case "n", "no":
				return false, nil
			case "":
				// Use default value logic
				switch strings.ToLower(defaultValue) {
				case "y", "yes":
					return true, nil
				case "n", "no":
					return false, nil
				default:
					s.println("Invalid input. Please enter 'y' or 'n'")
					continue
				}
			default:
				s.println("Invalid input. Please enter 'y' or 'n'")
				continue
			}
		}
	}
}

// Select allows user to select from multiple options by number.
//
//nolint:gocognit // This function is complex but separated into smaller functions would make it harder to read.
func (s *Service) Select(ctx context.Context, title, prompt string, options []string, defaultIndex int) (int, error) {
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			// Display options
			s.println("")
			if title != "" {
				s.println(" " + title)
			}
			for i, option := range options {
				s.printf("%d. %s\n", i+1, option)
			}

			defaultStr := ""
			if defaultIndex >= 0 && defaultIndex < len(options) {
				defaultStr = strconv.Itoa(defaultIndex + 1)
			}

			input, err := s.Prompt(ctx, prompt, defaultStr)
			if err != nil {
				return 0, err
			}

			if input == "q" || input == "quit" {
				return -1, ErrInputCanceled
			}

			// Parse selection
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(options) {
				s.printf("Please enter a number between 1 and %d\n", len(options))
				continue
			}

			return num - 1, nil // Convert to 0-based index
		}
	}
}

// SelectString allows user to select from multiple options, returns the selected string.
func (s *Service) SelectString(
	ctx context.Context,
	title, prompt string,
	options []string,
	defaultValue string,
) (string, error) {
	defaultIndex := -1
	for i, option := range options {
		if option == defaultValue {
			defaultIndex = i
			break
		}
	}

	selectedIndex, err := s.Select(ctx, title, prompt, options, defaultIndex)
	if err != nil {
		return "", err
	}
	if selectedIndex == -1 {
		return "", ErrInputCanceled
	}

	return options[selectedIndex], nil
}

// Helper methods for I/O operations

func (s *Service) readLine() (string, error) {
	input := s.Input
	if input == nil {
		input = os.Stdin
	}

	reader := bufio.NewReader(input)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line, nil
}

func (s *Service) printf(format string, args ...interface{}) {
	output := s.Output
	if output == nil {
		printer.Info(fmt.Sprintf(format, args...))
		return
	}
	fmt.Fprintf(output, format, args...)
}

func (s *Service) println(text string) {
	output := s.Output
	if output == nil {
		printer.Infoln(text)
		return
	}
	fmt.Fprintln(output, text)
}

func (s *Service) moveCursorUp(lines int) {
	if s.Output == nil {
		printer.MoveCursorUp(lines)
		return
	}
	// If using custom output, skip cursor movements
}

func (s *Service) moveCursorRight(chars int) {
	if s.Output == nil {
		printer.MoveCursorRight(chars)
		return
	}
	// If using custom output, skip cursor movements
}
