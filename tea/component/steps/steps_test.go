package steps

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestModel_Init(t *testing.T) {
	// Create a new model with default values
	model := New()

	// Get the init command
	cmd := model.Init()

	// Verify that the command is not nil (it should be the spinner's Tick command)
	assert.NotNil(t, cmd, "Init should return the spinner's Tick command")

	// Verify that the returned command is a tea.Cmd
	var _ tea.Cmd = cmd // Type assertion to ensure cmd implements tea.Cmd
}
