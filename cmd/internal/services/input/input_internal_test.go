package input

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	client := NewService()
	require.NotNil(t, client)
	require.Nil(t, client.Input)  // Should be nil to use os.Stdin
	require.Nil(t, client.Output) // Should be nil to use os.Stdout
}

func TestNewTestClient(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("test input")
	output := &bytes.Buffer{}

	client := NewTestService(input, output)
	require.Equal(t, input, client.Input)
	require.Equal(t, output, client.Output)
}

func TestPrompt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		prompt       string
		defaultValue string
		input        string
		expected     string
		shouldError  bool
	}{
		{
			name:     "simple prompt with input",
			prompt:   "Enter name",
			input:    "John\n",
			expected: "John",
		},
		{
			name:         "prompt with default value used",
			prompt:       "Enter name",
			defaultValue: "DefaultName",
			input:        "\n",
			expected:     "DefaultName",
		},
		{
			name:     "prompt with whitespace input",
			prompt:   "Enter value",
			input:    "  test value  \n",
			expected: "test value",
		},
		{
			name:     "empty prompt",
			prompt:   "",
			input:    "test\n",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}
			client := NewTestService(input, output)

			ctx := t.Context()
			result, err := client.Prompt(ctx, tt.prompt, tt.defaultValue)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPromptWithContextCancellation(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("test\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	_, err := client.Prompt(ctx, "Test prompt", "")
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestPromptWithTimeout(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("") // Empty input to simulate hanging
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	_, err := client.Prompt(ctx, "Test prompt", "")
	require.Error(t, err)
	// The actual error might be EOF when reader is exhausted, not necessarily timeout
}

func TestConfirm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		prompt       string
		defaultValue string
		inputs       []string // Multiple inputs to test retry logic
		expected     bool
		shouldError  bool
	}{
		{
			name:     "confirm with y",
			prompt:   "Continue?",
			inputs:   []string{"y\n"},
			expected: true,
		},
		{
			name:     "confirm with yes",
			prompt:   "Continue?",
			inputs:   []string{"yes\n"},
			expected: true,
		},
		{
			name:     "confirm with Y (uppercase)",
			prompt:   "Continue?",
			inputs:   []string{"Y\n"},
			expected: true,
		},
		{
			name:     "confirm with n",
			prompt:   "Continue?",
			inputs:   []string{"n\n"},
			expected: false,
		},
		{
			name:     "confirm with no",
			prompt:   "Continue?",
			inputs:   []string{"no\n"},
			expected: false,
		},
		{
			name:         "confirm with default yes",
			prompt:       "Continue?",
			defaultValue: "y",
			inputs:       []string{"\n"},
			expected:     true,
		},
		{
			name:         "confirm with default no",
			prompt:       "Continue?",
			defaultValue: "n",
			inputs:       []string{"\n"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := strings.NewReader(strings.Join(tt.inputs, ""))
			output := &bytes.Buffer{}
			client := NewTestService(input, output)

			ctx := t.Context()
			result, err := client.Confirm(ctx, tt.prompt, tt.defaultValue)

			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConfirmWithContextCancellation(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("y\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	_, err := client.Confirm(ctx, "Continue?", "")
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestSelect(t *testing.T) {
	t.Parallel()
	options := []string{"Option 1", "Option 2", "Option 3"}

	tests := []struct {
		name         string
		prompt       string
		options      []string
		defaultIndex int
		inputs       []string
		expected     int
		shouldError  bool
	}{
		{
			name:     "select first option",
			prompt:   "Choose option",
			options:  options,
			inputs:   []string{"1\n"},
			expected: 0,
		},
		{
			name:     "select second option",
			prompt:   "Choose option",
			options:  options,
			inputs:   []string{"2\n"},
			expected: 1,
		},
		{
			name:     "select third option",
			prompt:   "Choose option",
			options:  options,
			inputs:   []string{"3\n"},
			expected: 2,
		},
		{
			name:         "select with default",
			prompt:       "Choose option",
			options:      options,
			defaultIndex: 1,
			inputs:       []string{"\n"},
			expected:     1,
		},

		{
			name:        "select with quit",
			prompt:      "Choose option",
			options:     options,
			inputs:      []string{"q\n"},
			expected:    -1,
			shouldError: true,
		},
		{
			name:        "select with quit (full word)",
			prompt:      "Choose option",
			options:     options,
			inputs:      []string{"quit\n"},
			expected:    -1,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := strings.NewReader(strings.Join(tt.inputs, ""))
			output := &bytes.Buffer{}
			client := NewTestService(input, output)

			ctx := t.Context()
			result, err := client.Select(ctx, tt.prompt, tt.options, tt.defaultIndex)

			if tt.shouldError {
				require.Error(t, err)
				if tt.expected == -1 {
					require.Equal(t, ErrInputCanceled, err)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSelectWithContextCancellation(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	options := []string{"Option 1", "Option 2"}
	_, err := client.Select(ctx, "Choose", options, -1)
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestSelectString(t *testing.T) {
	t.Parallel()
	options := []string{"apple", "banana", "cherry"}

	tests := []struct {
		name         string
		prompt       string
		options      []string
		defaultValue string
		inputs       []string
		expected     string
		shouldError  bool
	}{
		{
			name:     "select string first option",
			prompt:   "Choose fruit",
			options:  options,
			inputs:   []string{"1\n"},
			expected: "apple",
		},
		{
			name:     "select string second option",
			prompt:   "Choose fruit",
			options:  options,
			inputs:   []string{"2\n"},
			expected: "banana",
		},
		{
			name:         "select string with default",
			prompt:       "Choose fruit",
			options:      options,
			defaultValue: "banana",
			inputs:       []string{"\n"},
			expected:     "banana",
		},
		{
			name:         "select string with invalid default",
			prompt:       "Choose fruit",
			options:      options,
			defaultValue: "orange",        // Not in options
			inputs:       []string{"1\n"}, // Should select first option manually
			expected:     "apple",
		},
		{
			name:        "select string with quit",
			prompt:      "Choose fruit",
			options:     options,
			inputs:      []string{"q\n"},
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := strings.NewReader(strings.Join(tt.inputs, ""))
			output := &bytes.Buffer{}
			client := NewTestService(input, output)

			ctx := t.Context()
			result, err := client.SelectString(ctx, tt.prompt, tt.options, tt.defaultValue)

			if tt.shouldError {
				require.Error(t, err)
				if strings.Contains(tt.inputs[0], "q") {
					require.Equal(t, ErrInputCanceled, err)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSelectStringWithContextCancellation(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	options := []string{"Option 1", "Option 2"}
	_, err := client.SelectString(ctx, "Choose", options, "")
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestServiceImplementsInterface(t *testing.T) {
	t.Parallel()
	// Ensure Client implements ClientInterface
	var _ ServiceInterface = (*Service)(nil)

	// Test that we can create and use the client
	client := NewService()
	assert.NotNil(t, client)
}

func TestErrorVariables(t *testing.T) {
	t.Parallel()
	// Test that our error variables are defined
	require.Error(t, ErrInputCanceled)
	require.Error(t, ErrInvalidInput)

	// Test error messages
	require.Contains(t, ErrInputCanceled.Error(), "input canceled")
	require.Contains(t, ErrInvalidInput.Error(), "invalid input")
}

// Integration-style tests

func TestPromptIntegration(t *testing.T) {
	t.Parallel()
	// Test the actual output formatting
	input := strings.NewReader("test input\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx := t.Context()
	result, err := client.Prompt(ctx, "Enter value", "default")

	require.NoError(t, err)
	require.Equal(t, "test input", result)

	// Check that prompt was written to output
	outputStr := output.String()
	assert.Contains(t, outputStr, "Enter value")
	assert.Contains(t, outputStr, "[default]")
}

func TestSelectIntegration(t *testing.T) {
	t.Parallel()
	// Test that options are displayed correctly
	input := strings.NewReader("2\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	options := []string{"First", "Second", "Third"}
	ctx := t.Context()
	result, err := client.Select(ctx, "Choose", options, -1)

	require.NoError(t, err)
	assert.Equal(t, 1, result)

	// Check that options were displayed
	outputStr := output.String()
	assert.Contains(t, outputStr, "1. First")
	assert.Contains(t, outputStr, "2. Second")
	assert.Contains(t, outputStr, "3. Third")
}

func TestConfirmEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		defaultValue string
		input        string
		expected     bool
	}{
		{
			name:         "empty input with y default",
			defaultValue: "y",
			input:        "\n",
			expected:     true,
		},
		{
			name:         "empty input with yes default",
			defaultValue: "yes",
			input:        "\n",
			expected:     true,
		},
		{
			name:         "empty input with n default",
			defaultValue: "n",
			input:        "\n",
			expected:     false,
		},
		{
			name:         "empty input with no default",
			defaultValue: "no",
			input:        "\n",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}
			client := NewTestService(input, output)

			ctx := t.Context()
			result, err := client.Confirm(ctx, "Continue?", tt.defaultValue)

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSelectWithEmptyOptions(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx := t.Context()
	_, err := client.Select(ctx, "Choose", []string{}, -1)

	// Should get an error when trying to parse "1" with no options
	require.Error(t, err)
}

func TestSelectStringWithEmptyOptions(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("1\n")
	output := &bytes.Buffer{}
	client := NewTestService(input, output)

	ctx := t.Context()
	_, err := client.SelectString(ctx, "Choose", []string{}, "")

	// Should get an error when trying to parse "1" with no options
	require.Error(t, err)
}
