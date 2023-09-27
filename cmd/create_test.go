package cmd_test

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"

	"pkg.world.dev/world-cli/cmd"
)

func TestNewProjectCreation(t *testing.T) {
	projectName := "test"
	err := cmd.CreateNewProject(projectName)
	assert.NilError(t, err)
	fileInfo, err := os.Stat(projectName)
	assert.NilError(t, err)
	assert.Assert(t, fileInfo.IsDir())
	err = os.RemoveAll(projectName)
	assert.NilError(t, err)
	_, err = os.Stat(projectName)
	assert.Assert(t, err != nil)
}
