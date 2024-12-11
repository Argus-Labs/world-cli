package root

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	"pkg.world.dev/world-cli/config"
)

var (
	testBaseDir string
	origWorkDir string
)

func TestMain(m *testing.M) {
	var err error
	// Save original working directory
	origWorkDir, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	// Create base test directory
	testBaseDir, err = os.MkdirTemp("", "world-cli-test")
	if err != nil {
		panic(err)
	}

	// Initialize config with Docker environment variables
	cfg, err := config.GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to get config: %v", err))
	}

	// Set up Docker environment variables
	if cfg.DockerEnv == nil {
		cfg.DockerEnv = make(map[string]string)
	}

	// Set required environment variables
	cfg.DockerEnv["CARDINAL_NAMESPACE"] = "test-cardinal"
	cfg.DockerEnv["DA_AUTH_TOKEN"] = "test-token"
	cfg.DockerEnv["DA_BASE_URL"] = "http://localhost:26657"
	cfg.DockerEnv["DA_NAMESPACE_ID"] = "test-namespace"

	// Set environment variables for backward compatibility
	//nolint:tenv // testing.Setenv is not available in current Go version
	if err := os.Setenv("CARDINAL_NAMESPACE", cfg.DockerEnv["CARDINAL_NAMESPACE"]); err != nil {
		panic(err)
	}
	if err := os.Setenv("DA_AUTH_TOKEN", cfg.DockerEnv["DA_AUTH_TOKEN"]); err != nil {
		panic(err)
	}
	if err := os.Setenv("DA_BASE_URL", cfg.DockerEnv["DA_BASE_URL"]); err != nil {
		panic(err)
	}
	if err := os.Setenv("DA_NAMESPACE_ID", cfg.DockerEnv["DA_NAMESPACE_ID"]); err != nil {
		panic(err)
	}

	// Run tests
	code := m.Run()

	// Cleanup with error checking
	if err := os.Chdir(origWorkDir); err != nil {
		panic(fmt.Sprintf("failed to change directory in cleanup: %v", err))
	}
	if err := os.RemoveAll(testBaseDir); err != nil {
		panic(fmt.Sprintf("failed to remove test directory: %v", err))
	}
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
	lines, err := outputFromCmd(rootCmd, "help")
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
	_, err := outputFromCmd(getDoctorCmd(teaOut), "")
	assert.NilError(t, err)

	seenDependencies := map[string]int{
		"Git":                      0,
		"Go":                       0,
		"Docker":                   0,
		"Docker daemon is running": 0,
	}

	lines := strings.Split(teaOut.String(), "\r\n")
	for _, line := range lines {
		// Check each dependency by looking for its name in the line
		for dep := range seenDependencies {
			if strings.Contains(line, dep) {
				seenDependencies[dep]++
			}
		}
	}

	for dep, count := range seenDependencies {
		assert.Check(t, count > 0, "dependencies %q is not listed in dependencies checking", dep)
	}
}

func TestCreateStartStopRestartPurge(t *testing.T) {
	// Create test directory within base test directory
	testDir := filepath.Join(testBaseDir, "test-create-start-stop")
	err := os.MkdirAll(testDir, 0755)
	assert.NilError(t, err)

	// Change to test directory
	err = os.Chdir(testDir)
	assert.NilError(t, err)

	// Ensure we return to original directory after test
	defer func() {
		err = os.Chdir(origWorkDir)
		assert.NilError(t, err)
	}()

	// set tea output to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)

	// checkout the repo
	sgtDir := filepath.Join(testDir, "sgt")
	createCmd.SetArgs([]string{sgtDir})
	err = createCmd.Execute()
	assert.NilError(t, err)

	// Change dir to project root and verify cardinal directory exists
	err = os.Chdir(sgtDir)
	assert.NilError(t, err)

	// Verify cardinal directory exists and is accessible
	cardinalDir := filepath.Join(sgtDir, "cardinal")
	_, err = os.Stat(cardinalDir)
	assert.NilError(t, err, "cardinal directory not found in project root")

	// Start cardinal
	rootCmd.SetArgs([]string{"cardinal", "start", "--detach", "--editor=false"})
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
	// Create test directory within base test directory
	testDir := filepath.Join(testBaseDir, "test-dev")
	err := os.MkdirAll(testDir, 0755)
	assert.NilError(t, err)

	// Change to test directory
	err = os.Chdir(testDir)
	assert.NilError(t, err)

	// Ensure we return to original directory after test
	defer func() {
		err = os.Chdir(origWorkDir)
		assert.NilError(t, err)
	}()

	// Get config and update paths for this test
	cfg, err := config.GetConfig()
	assert.NilError(t, err)

	// set tea output to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)

	// checkout the repo
	sgtDir := filepath.Join(testDir, "sgt")
	createCmd.SetArgs([]string{sgtDir})
	err = createCmd.Execute()
	assert.NilError(t, err)

	// Change dir to project root and verify cardinal directory exists
	err = os.Chdir(sgtDir)
	assert.NilError(t, err)

	// Update config with correct paths for this test
	cfg.RootDir = sgtDir
	cfg.GameDir = "cardinal"

	// Verify cardinal directory exists and is accessible
	cardinalDir := filepath.Join(sgtDir, "cardinal")
	_, err = os.Stat(cardinalDir)
	assert.NilError(t, err, "cardinal directory not found in project root")

	// Ensure environment variables are set for this test
	err = os.Setenv("CARDINAL_NAMESPACE", cfg.DockerEnv["CARDINAL_NAMESPACE"])
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
	t.Cleanup(func() {
		AppVersion = ""
	})

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
	maxAttempts := 30                       // Reduce max attempts to 30
	retryInterval := 200 * time.Millisecond // Shorter retry interval

	for i := 0; i < maxAttempts; i++ {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			time.Sleep(retryInterval)
			t.Logf("%s is not running, retrying... (attempt %d/%d)\n", name, i+1, maxAttempts)
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
	maxAttempts := 30                       // Reduce max attempts to 30
	retryInterval := 200 * time.Millisecond // Shorter retry interval

	for i := 0; i < maxAttempts; i++ {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			down = true
			break
		}
		_ = conn.Close()
		time.Sleep(retryInterval)
		t.Logf("%s is still running, retrying... (attempt %d/%d)\n", name, i+1, maxAttempts)
		continue
	}
	return down
}

func TestEVMStart(t *testing.T) {
	// Set required environment variables for Docker before any operations
	t.Setenv("CARDINAL_NAMESPACE", "test-cardinal-evm")

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

	// set tea ouput to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)

	// checkout the repo
	sgtDir := filepath.Join(gameDir, "sgt")
	createCmd.SetArgs([]string{sgtDir})
	err = createCmd.Execute()
	assert.NilError(t, err)

	// Change dir to project root and verify cardinal directory exists
	err = os.Chdir(sgtDir)
	assert.NilError(t, err)

	// Verify cardinal directory exists and is accessible
	cardinalDir := filepath.Join(sgtDir, "cardinal")
	_, err = os.Stat(cardinalDir)
	assert.NilError(t, err, "cardinal directory not found in project root")

	// Start EVM without detach flag
	rootCmd.SetArgs([]string{"evm", "start"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until EVM is healthy
	assert.Assert(t, evmIsUp(t), "EVM is not running")

	// Stop EVM
	rootCmd.SetArgs([]string{"evm", "stop"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until EVM shutdowns
	assert.Assert(t, evmIsDown(t), "EVM is not successfully shutdown")
}
