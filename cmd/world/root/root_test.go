package root

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"

	"pkg.world.dev/world-cli/cmd/world/cardinal"
)

type healthResponse struct {
	StatusCode        int
	IsServerRunning   bool
	IsGameLoopRunning bool
}

func getHealthCheck() (*healthResponse, error) {
	var healtCheck healthResponse

	resp, err := http.Get("http://127.0.0.1:4040/health")
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(resp.Body).Decode(&healtCheck)
	if err != nil {
		return nil, err
	}

	healtCheck.StatusCode = resp.StatusCode
	return &healtCheck, nil
}

// outputFromCmd runs the rootCmd with the given cmd arguments and returns the output of the command along with
// any errors.
func outputFromCmd(cobra *cobra.Command, cmd string) ([]string, error) {
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

	if err := cobra.Execute(); err != nil {
		return nil, fmt.Errorf("root command failed with: %w", err)
	}
	lines := strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		return lines, errors.New(errorStr)
	}

	return lines, nil
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

func TestCreateStartStopRestartPurge(t *testing.T) {
	// Create Cardinal
	gameDir, err := os.MkdirTemp("", "game-template")
	assert.NilError(t, err)

	// Remove dir
	defer func() {
		err = os.RemoveAll(gameDir)
		assert.NilError(t, err)
	}()

	// Change dir
	err = os.Chdir(gameDir)
	assert.NilError(t, err)

	// set tea ouput to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)
	createCmd.SetArgs([]string{gameDir})

	err = createCmd.Execute()
	assert.NilError(t, err)

	// Start cardinal
	rootCmd.SetArgs([]string{"cardinal", "start", "--build", "--detach"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	defer func() {
		// Purge cardinal
		rootCmd.SetArgs([]string{"cardinal", "purge"})
		err = rootCmd.Execute()
		assert.NilError(t, err)
	}()

	// Check cardinal health
	resp, err := getHealthCheck()
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Assert(t, resp.IsServerRunning)
	assert.Assert(t, resp.IsGameLoopRunning)

	// Restart cardinal
	rootCmd.SetArgs([]string{"cardinal", "restart", "--detach"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check cardinal health after restart
	resp, err = getHealthCheck()
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, 200)
	assert.Assert(t, resp.IsServerRunning)
	assert.Assert(t, resp.IsGameLoopRunning)

	// Stop cardinal
	rootCmd.SetArgs([]string{"cardinal", "stop"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check cardinal health (expected error)
	_, err = getHealthCheck()
	assert.Error(t, err,
		"Get \"http://127.0.0.1:4040/health\": dial tcp 127.0.0.1:4040: connect: connection refused")
}

func TestDev(t *testing.T) {
	// Create Cardinal
	gameDir, err := os.MkdirTemp("", "game-template")
	assert.NilError(t, err)

	// Remove dir
	defer func() {
		err = os.RemoveAll(gameDir)
		assert.NilError(t, err)
	}()

	// Change dir
	err = os.Chdir(gameDir)
	assert.NilError(t, err)

	// set tea ouput to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)
	createCmd.SetArgs([]string{gameDir})

	err = createCmd.Execute()
	assert.NilError(t, err)

	// Start cardinal dev
	rootCmd.SetArgs([]string{"cardinal", "dev", "--watch"})
	go func() {
		err := rootCmd.Execute()
		assert.NilError(t, err)
	}()

	// Check cardinal health for 300 second, waiting to download dependencies and building the apps
	isServerRunning := false
	isGameLoopRunning := false
	timeout := time.Now().Add(300 * time.Second)
	for !(isServerRunning && isGameLoopRunning) && time.Now().Before(timeout) {
		resp, err := getHealthCheck()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		assert.Equal(t, resp.StatusCode, 200)
		isServerRunning = resp.IsServerRunning
		isGameLoopRunning = resp.IsGameLoopRunning
	}
	assert.Assert(t, isServerRunning)
	assert.Assert(t, isGameLoopRunning)

	// Shutdown the program
	close(cardinal.StopChan)

	// Check cardinal health (expected error), trying for 10 times
	count := 0
	for count < 10 {
		_, err = getHealthCheck()
		if err != nil {
			break
		}
		time.Sleep(1 * time.Second)
		count++
	}

	assert.Error(t, err,
		"Get \"http://127.0.0.1:4040/health\": dial tcp 127.0.0.1:4040: connect: connection refused")
}

func TestCheckLatestVersion(t *testing.T) {
	t.Run("success scenario", func(t *testing.T) {
		AppVersion = "v1.0.0"
		err := checkLatestVersion()
		assert.NilError(t, err)
	})

	t.Run("error version format", func(t *testing.T) {
		AppVersion = "wrong format"
		err := checkLatestVersion()
		assert.Error(t, err, "error parsing current version: Malformed version: wrong format")
	})
}
