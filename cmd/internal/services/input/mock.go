package input

import (
	"context"
	"errors"

	"github.com/stretchr/testify/mock"
)

// MockService provides a mock implementation of ServiceInterface for testing.
type MockService struct {
	mock.Mock
}

// Ensure MockService implements ServiceInterface.
var _ ServiceInterface = (*MockService)(nil)

// Prompt mocks the Prompt method.
func (m *MockService) Prompt(ctx context.Context, prompt, defaultValue string) (string, error) {
	args := m.Called(ctx, prompt, defaultValue)
	return args.String(0), args.Error(1)
}

// Confirm mocks the Confirm method.
func (m *MockService) Confirm(ctx context.Context, prompt, defaultValue string) (bool, error) {
	args := m.Called(ctx, prompt, defaultValue)
	return args.Bool(0), args.Error(1)
}

// Select mocks the Select method.
func (m *MockService) Select(ctx context.Context, prompt string, options []string, defaultIndex int) (int, error) {
	args := m.Called(ctx, prompt, options, defaultIndex)
	return args.Int(0), args.Error(1)
}

// SelectString mocks the SelectString method.
func (m *MockService) SelectString(
	ctx context.Context,
	prompt string,
	options []string,
	defaultValue string,
) (string, error) {
	args := m.Called(ctx, prompt, options, defaultValue)
	return args.String(0), args.Error(1)
}

// TestInputService provides a simple test implementation with predefined responses.
type TestInputService struct {
	Responses       []string // Predefined responses in order
	ResponseIndex   int      // Current index in responses
	ConfirmResponse bool     // Default confirm response
}

// NewTestInputService creates a new test service with predefined responses.
func NewTestInputService(responses []string) *TestInputService {
	return &TestInputService{
		Responses:       responses,
		ResponseIndex:   0,
		ConfirmResponse: true,
	}
}

// Prompt returns the next predefined response.
func (t *TestInputService) Prompt(_ context.Context, _, defaultValue string) (string, error) {
	if t.ResponseIndex >= len(t.Responses) {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", errors.New("no more test responses available")
	}

	response := t.Responses[t.ResponseIndex]
	t.ResponseIndex++

	if response == "" && defaultValue != "" {
		return defaultValue, nil
	}

	return response, nil
}

// Confirm returns the predefined confirm response.
func (t *TestInputService) Confirm(_ context.Context, _, _ string) (bool, error) {
	return t.ConfirmResponse, nil
}

// Select returns the first option by default.
func (t *TestInputService) Select(_ context.Context, _ string, options []string, defaultIndex int) (int, error) {
	if defaultIndex >= 0 && defaultIndex < len(options) {
		return defaultIndex, nil
	}
	if len(options) > 0 {
		return 0, nil
	}
	return -1, errors.New("no options available")
}

// SelectString returns the default value or first option.
func (t *TestInputService) SelectString(
	_ context.Context,
	_ string,
	options []string,
	defaultValue string,
) (string, error) {
	if defaultValue != "" {
		for _, option := range options {
			if option == defaultValue {
				return defaultValue, nil
			}
		}
	}
	if len(options) > 0 {
		return options[0], nil
	}
	return "", errors.New("no options available")
}
