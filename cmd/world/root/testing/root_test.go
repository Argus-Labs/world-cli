package root

import (
	"bytes"
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"pkg.world.dev/world-cli/cmd/world/cardinal"
	"pkg.world.dev/world-cli/cmd/world/evm"
	"pkg.world.dev/world-cli/cmd/world/root"
)

var (
	// testEnv holds the global test environment.
	testEnv *testEnvironment
)

// testEnvironment holds the test environment setup.
type testEnvironment struct {
	rootCmd *cobra.Command
}

// setupTestEnv initializes the test environment.
func setupTestEnv() *testEnvironment {
	// Initialize all commands
	cardinal.Init()
	evm.EvmInit()
	root.RootCmdInit()

	return &testEnvironment{
		rootCmd: root.RootCmdTesting,
	}
}

// TestMain runs before all tests and handles setup/teardown.
func TestMain(m *testing.M) {
	// Setup
	testEnv = setupTestEnv()

	// Run tests
	code := m.Run()

	// Teardown if needed
	// Add any cleanup code here

	os.Exit(code)
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
		return nil, eris.Errorf("root command failed with: %v", err)
	}
	lines := strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		return lines, eris.New(errorStr)
	}

	return lines, nil
}

func TestSubcommandsHaveHelpText(t *testing.T) {
	lines, err := outputFromCmd(testEnv.rootCmd, "help")
	assert.NilError(t, err)
	seenSubcommands := map[string]int{
		"cardinal": 0,
		"doctor":   0,
		"help":     0,
		"version":  0,
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
	_, err := outputFromCmd(root.GetDoctorCmdTesting(teaOut), "")
	assert.NilError(t, err)

	seenDependencies := map[string]int{
		"Git":                      0,
		"Go":                       0,
		"Docker":                   0,
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
	gameDir, err := os.MkdirTemp("", "game-template-start")
	assert.NilError(t, err)

	// Remove dir
	defer func() {
		err = os.RemoveAll(gameDir)
		assert.NilError(t, err)
	}()

	// Change dir
	err = os.Chdir(gameDir)
	assert.NilError(t, err)

	// set tea output to variable
	teaOut := &bytes.Buffer{}
	createCmd := root.GetCreateCmdTesting(teaOut)

	// checkout the repo
	sgtDir := gameDir + "/sgt"
	createCmd.SetArgs([]string{sgtDir})
	err = createCmd.Execute()
	assert.NilError(t, err)

	// Change dir
	err = os.Chdir(sgtDir)
	assert.NilError(t, err)

	// Start cardinal
	testEnv.rootCmd.SetArgs([]string{"cardinal", "start", "--detach", "--editor=false"})
	err = testEnv.rootCmd.Execute()
	assert.NilError(t, err)

	defer func() {
		// Purge cardinal
		testEnv.rootCmd.SetArgs([]string{"cardinal", "purge"})
		err = testEnv.rootCmd.Execute()
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Restart cardinal
	testEnv.rootCmd.SetArgs([]string{"cardinal", "restart", "--detach"})
	err = testEnv.rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Stop cardinal
	testEnv.rootCmd.SetArgs([]string{"cardinal", "stop"})
	err = testEnv.rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until cardinal shutdowns
	assert.Assert(t, cardinalIsDown(t), "Cardinal is not successfully shutdown")
}

func TestDev(t *testing.T) {
	// Create Cardinal
	gameDir, err := os.MkdirTemp("", "game-template-dev")
	assert.NilError(t, err)

	// Remove dir
	defer func() {
		err = os.RemoveAll(gameDir)
		assert.NilError(t, err)
	}()

	// Change dir
	err = os.Chdir(gameDir)
	assert.NilError(t, err)

	// set tea output to variable
	teaOut := &bytes.Buffer{}
	createCmd := root.GetCreateCmdTesting(teaOut)
	createCmd.SetArgs([]string{gameDir})

	// checkout the repo
	sgtDir := gameDir + "/sgt"
	createCmd.SetArgs([]string{sgtDir})
	err = createCmd.Execute()
	assert.NilError(t, err)

	// Start cardinal dev
	ctx, cancel := context.WithCancel(context.Background())
	testEnv.rootCmd.SetArgs([]string{"cardinal", "dev", "--editor=false"})
	go func() {
		err := testEnv.rootCmd.ExecuteContext(ctx)
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
	t.Cleanup(func() {
		root.AppVersion = ""
	})

	t.Run("success scenario", func(t *testing.T) {
		root.AppVersion = "v1.0.0"
		err := root.CheckLatestVersionTesting()
		assert.NilError(t, err)
	})

	t.Run("error version format", func(t *testing.T) {
		root.AppVersion = "wrong format"
		err := root.CheckLatestVersionTesting()
		assert.Error(t, err, "error parsing current version: Malformed version: wrong format")
	})
}

func cardinalIsUp(t *testing.T) bool {
	return ServiceIsUp("Cardinal", "localhost:4040", t)
}

func cardinalIsDown(t *testing.T) bool {
	return ServiceIsDown("Cardinal", "localhost:4040", t)
}

func evmIsUp(t *testing.T) bool {
	return ServiceIsUp("EVM", "localhost:9601", t)
}

func evmIsDown(t *testing.T) bool {
	return ServiceIsDown("EVM", "localhost:9601", t)
}

func ServiceIsUp(name, address string, t *testing.T) bool {
	up := false
	for i := 0; i < 120; i++ {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			time.Sleep(time.Second)
			t.Logf("%s is not running, retrying...\n", name)
			continue
		}
		_ = conn.Close()
		up = true
		break
	}
	return up
}

func ServiceIsDown(name, address string, t *testing.T) bool {
	down := false
	for i := 0; i < 120; i++ {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			down = true
			break
		}
		_ = conn.Close()
		time.Sleep(time.Second)
		t.Logf("%s is still running, retrying...\n", name)
		continue
	}
	return down
}

func TestEVMStart(t *testing.T) {
	// Create Cardinal
	gameDir, err := os.MkdirTemp("", "game-template-dev")
	assert.NilError(t, err)

	// Remove dir
	defer func() {
		err = os.RemoveAll(gameDir)
		assert.NilError(t, err)
	}()

	// Change dir
	err = os.Chdir(gameDir)
	assert.NilError(t, err)

	// set tea output to variable
	teaOut := &bytes.Buffer{}
	createCmd := root.GetCreateCmdTesting(teaOut)
	createCmd.SetArgs([]string{gameDir})

	// checkout the repo
	sgtDir := gameDir + "/sgt"
	createCmd.SetArgs([]string{sgtDir})
	err = createCmd.Execute()
	assert.NilError(t, err)

	// Start evn dev
	ctx, cancel := context.WithCancel(context.Background())
	testEnv.rootCmd.SetArgs([]string{"evm", "start", "--dev"})
	go func() {
		err := testEnv.rootCmd.ExecuteContext(ctx)
		assert.NilError(t, err)
	}()

	// Check and wait until evm is up
	assert.Assert(t, evmIsUp(t), "EVM is not running")

	// Shutdown the program
	cancel()

	// Check and wait until evm is down
	assert.Assert(t, evmIsDown(t), "EVM is not successfully shutdown")
}
