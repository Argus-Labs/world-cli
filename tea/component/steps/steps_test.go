package steps

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestModel_Init(t *testing.T) {
	// Create a new model with default values
	model := New()

	// Get the init command
	cmd := model.Init()

	// Verify that the command is not nil and implements tea.Cmd
	assert.NotNil(t, cmd, "Init should return a non-nil command")
	var _ tea.Cmd = cmd

	// Execute the command and verify it produces a spinner.TickMsg
	msg := cmd()
	_, ok := msg.(spinner.TickMsg)
	assert.True(t, ok, "Init should return a command that produces spinner.TickMsg")
}
