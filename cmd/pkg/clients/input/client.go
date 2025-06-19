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

// NewClient creates a new input client with standard stdin/stdout.
func NewClient() Client {
	return Client{
		Input:  nil, // Will use os.Stdin if nil
		Output: nil, // Will use os.Stdout if nil
	}
}

// NewTestClient creates a new input client for testing with custom input/output.
func NewTestClient(input io.Reader, output io.Writer) Client {
	return Client{
		Input:  input,
		Output: output,
	}
}

// Prompt displays a prompt and returns user input.
func (c *Client) Prompt(ctx context.Context, prompt, defaultValue string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Display prompt
		if prompt != "" {
			c.printf("%s", prompt)
		}
		if defaultValue != "" {
			c.printf(" [%s]: ", defaultValue)
		} else {
			c.printf(": ")
		}

		// Read input
		input, err := c.readLine()
		if err != nil {
			return "", eris.Wrap(err, "failed to read input")
		}

		input = strings.TrimSpace(input)
		if input == "" && defaultValue != "" {
			// Display the default value as if they typed it in
			c.moveCursorUp(1)
			c.moveCursorRight(len(defaultValue) + 4 + len(prompt))
			c.println(defaultValue)
			return defaultValue, nil
		}
		return input, nil
	}
}

// Confirm asks for Y/n confirmation with default.
func (c *Client) Confirm(ctx context.Context, prompt, defaultValue string) (bool, error) {
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			input, err := c.Prompt(ctx, prompt, defaultValue)
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
					c.println("Invalid input. Please enter 'y' or 'n'")
					continue
				}
			default:
				c.println("Invalid input. Please enter 'y' or 'n'")
				continue
			}
		}
	}
}

// Select allows user to select from multiple options by number.
//
//nolint:gocognit // This function is complex but seperated into smaller functions would make it harder to read.
func (c *Client) Select(ctx context.Context, prompt string, options []string, defaultIndex int) (int, error) {
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			// Display options
			c.println("")
			for i, option := range options {
				c.printf("  %d. %s\n", i+1, option)
			}

			defaultStr := ""
			if defaultIndex >= 0 && defaultIndex < len(options) {
				defaultStr = strconv.Itoa(defaultIndex + 1)
			}

			input, err := c.Prompt(ctx, prompt, defaultStr)
			if err != nil {
				return 0, err
			}

			if input == "q" || input == "quit" {
				return -1, ErrInputCanceled
			}

			// Parse selection
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(options) {
				c.printf("Please enter a number between 1 and %d\n", len(options))
				continue
			}

			return num - 1, nil // Convert to 0-based index
		}
	}
}

// SelectString allows user to select from multiple options, returns the selected string.
func (c *Client) SelectString(
	ctx context.Context,
	prompt string,
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

	selectedIndex, err := c.Select(ctx, prompt, options, defaultIndex)
	if err != nil {
		return "", err
	}
	if selectedIndex == -1 {
		return "", ErrInputCanceled
	}

	return options[selectedIndex], nil
}

// Helper methods for I/O operations

func (c *Client) readLine() (string, error) {
	input := c.Input
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

func (c *Client) printf(format string, args ...interface{}) {
	output := c.Output
	if output == nil {
		printer.Info(fmt.Sprintf(format, args...))
		return
	}
	fmt.Fprintf(output, format, args...)
}

func (c *Client) println(text string) {
	output := c.Output
	if output == nil {
		printer.Infoln(text)
		return
	}
	fmt.Fprintln(output, text)
}

func (c *Client) moveCursorUp(lines int) {
	if c.Output == nil {
		printer.MoveCursorUp(lines)
		return
	}
	// If using custom output, skip cursor movements
}

func (c *Client) moveCursorRight(chars int) {
	if c.Output == nil {
		printer.MoveCursorRight(chars)
		return
	}
	// If using custom output, skip cursor movements
}
