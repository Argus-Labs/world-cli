package root

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

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
	rootCmd.SetArgs([]string{"cardinal", "start", "--build", "--detach", "--editor=false"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	defer func() {
		// Purge cardinal
		rootCmd.SetArgs([]string{"cardinal", "purge"})
		err = rootCmd.Execute()
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Restart cardinal
	rootCmd.SetArgs([]string{"cardinal", "restart", "--detach"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Stop cardinal
	rootCmd.SetArgs([]string{"cardinal", "stop"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until cardinal shutdowns
	assert.Assert(t, cardinalIsDown(t), "Cardinal is not successfully shutdown")
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
	ctx, cancel := context.WithCancel(context.Background())
	rootCmd.SetArgs([]string{"cardinal", "dev", "--editor=false"})
	go func() {
		err := rootCmd.ExecuteContext(ctx)
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Shutdown the program
	cancel()

	// Check and wait until cardinal shutdowns
	assert.Assert(t, cardinalIsDown(t), "Cardinal is not successfully shutdown")
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

func cardinalIsUp(t *testing.T) bool {
	up := false
	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:4040", time.Second)
		if err != nil {
			time.Sleep(time.Second)
			t.Log("Failed to connect to Cardinal, retrying...")
			continue
		}
		_ = conn.Close()
		up = true
		break
	}
	return up
}

func cardinalIsDown(t *testing.T) bool {
	down := false
	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:4040", time.Second)
		if err != nil {
			down = true
			break
		}
		_ = conn.Close()
		time.Sleep(time.Second)
		t.Log("Cardinal is still running, retrying...")
		continue
	}
	return down
}
