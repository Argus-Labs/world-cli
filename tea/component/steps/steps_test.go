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

func TestModel_Update(t *testing.T) {
	// Create a new model
	model := New()

	// Test spinner tick message
	spinnerMsg := spinner.TickMsg{}
	newModel, cmd := model.Update(spinnerMsg)
	assert.NotNil(t, cmd, "Update should return a command for spinner tick")
	assert.Equal(t, model.index, newModel.index, "Index should not change on spinner tick")

	// Test start message
	startMsg := StartMsg{}
	newModel, cmd = model.Update(startMsg)
	assert.NotNil(t, cmd, "Update should return a command for start message")
	assert.Equal(t, 0, newModel.index, "Index should be 0 after start")

	// Add some test steps
	model.Steps = []Entry{
		{Text: "Step 1", Status: INCOMPLETE},
		{Text: "Step 2", Status: INCOMPLETE},
	}

	// Test complete step message without error
	completeMsg := CompleteStepMsg{Err: nil}
	newModel, cmd = model.Update(completeMsg)
	assert.NotNil(t, cmd, "Update should return a command for complete message")
	assert.Equal(t, COMPLETE, newModel.Steps[0].Status, "First step should be marked as complete")
	assert.Equal(t, 1, newModel.index, "Index should increment after completion")

	// Test complete step message with error
	errorMsg := CompleteStepMsg{Err: assert.AnError}
	newModel, _ := model.Update(errorMsg)
	assert.Equal(t, FAILED, newModel.Steps[1].Status, "Step should be marked as failed on error")
	assert.Equal(t, assert.AnError, newModel.Steps[1].Err, "Error should be stored in the step")
}

func TestModel_View(t *testing.T) {
	// Create a new model
	model := New()
	model.Steps = []Entry{
		{Text: "Step 1", Status: INCOMPLETE},
		{Text: "Step 2", Status: INCOMPLETE},
	}

	// Test initial view
	view := model.View()
	assert.Contains(t, view, "Step 1", "View should contain first step")
	assert.Contains(t, view, "Step 2", "View should contain second step")

	// Test view with completed step
	model.Steps[0].Status = COMPLETE
	model.index = 1
	view = model.View()
	assert.Contains(t, view, "Step 1", "View should contain completed step")
	assert.Contains(t, view, "Step 2", "View should contain current step")

	// Test view with failed step
	model.Steps[1].Status = FAILED
	view = model.View()
	assert.Contains(t, view, "Step 1", "View should contain completed step")
	assert.Contains(t, view, "Step 2", "View should contain failed step")
}
