package root

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// outputFromCmd runs the rootCmd with the given cmd arguments and returns the output of the command along with
// any errors.
func outputFromCmd(cobra *cobra.Command, cmd string) (lines []string, err error) {
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	cobra.SetOut(stdOut)
	defer func() {
		cobra.SetOut(nil)
	}()
	cobra.SetErr(stdErr)
	defer func() {
		cobra.SetErr(nil)
	}()
	cobra.SetArgs(strings.Split(cmd, " "))
	defer func() {
		cobra.SetArgs(nil)
	}()

	if err = cobra.Execute(); err != nil {
		return nil, fmt.Errorf("root command failed with: %w", err)
	}
	lines = strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		err = errors.New(errorStr)
	}
	return lines, err
}

func TestSubcommandsHaveHelpText(t *testing.T) {
	lines, err := outputFromCmd(rootCmd, "help")
	assert.NilError(t, err)
	seenSubcommands := map[string]int{
		"cardinal":   0,
		"completion": 0,
		"doctor":     0,
		"help":       0,
		"version":    0,
	}

	for _, line := range lines {
		for subcommand := range seenSubcommands {
			if strings.HasPrefix(line, "  "+subcommand) {
				seenSubcommands[subcommand]++
			}
		}
	}

	for subcommand, count := range seenSubcommands {
		assert.Check(t, count > 0, "subcommand %q is not listed in the help command", subcommand)
	}
}

func TestExecuteDoctorCommand(t *testing.T) {
	teaOut := &bytes.Buffer{}
	_, err := outputFromCmd(getDoctorCmd(teaOut), "")
	assert.NilError(t, err)

	seenDependencies := map[string]int{
		"Git":                      0,
		"Go":                       0,
		"Docker":                   0,
		"Docker Compose":           0,
		"Docker daemon is running": 0,
	}

	lines := strings.Split(teaOut.String(), "\r\n")
	for _, line := range lines {
		// Remove the first three characters for the example(✓  Git)
		resultString := ""
		if len(line) > 5 {
			resultString = line[5:]
		}
		for dep := range seenDependencies {
			if resultString != "" && resultString == dep {
				seenDependencies[dep]++
			}
		}
	}

	for dep, count := range seenDependencies {
		assert.Check(t, count > 0, "dependencies %q is not listed in dependencies checking", dep)
	}
}

func TestCreateStartStopPurge(t *testing.T) {
	// Create Cardinal
	gameDir, err := os.MkdirTemp("", "game-template")
	assert.NilError(t, err)

	// Remove dir
	defer os.RemoveAll(gameDir)

	// set tea ouput to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)
	createCmd.SetArgs([]string{gameDir})

	err = createCmd.Execute()
	assert.NilError(t, err)

	// Start cardinal
	os.Chdir(gameDir)
	rootCmd.SetArgs([]string{"cardinal", "start", "--build", "--detach"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check cardinal health
	resp, err := http.Get("http://127.0.0.1:3333/health")
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	var healthResponse struct {
		IsServerRunning   bool
		IsGameLoopRunning bool
	}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	assert.NilError(t, err)
	assert.Assert(t, healthResponse.IsServerRunning)
	assert.Assert(t, healthResponse.IsGameLoopRunning)

	// Stop cardinal
	rootCmd.SetArgs([]string{"cardinal", "stop"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check cardinal health
	_, err = http.Get("http://127.0.0.1:3333/health")
	assert.Error(t, err,
		"Get \"http://127.0.0.1:3333/health\": dial tcp 127.0.0.1:3333: connect: connection refused")

	// Purge cardinal
	rootCmd.SetArgs([]string{"cardinal", "purge"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

}
