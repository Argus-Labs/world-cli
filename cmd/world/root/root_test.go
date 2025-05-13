package root

import (
	"bytes"
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rotisserie/eris"
	"gotest.tools/v3/assert"
	"pkg.world.dev/world-cli/cmd/world/cardinal"
	"pkg.world.dev/world-cli/cmd/world/evm"
)

// setupTestEnv initializes the test environment.
func setupTestEnv() {
	// Initialize all commands
	cardinal.Init()
	evm.Init()
}

// TestMain runs before all tests and handles setup/teardown.
func TestMain(m *testing.M) {
	// Setup
	setupTestEnv()

	// Run tests
	code := m.Run()

	// Teardown if needed
	// Add any cleanup code here

	os.Exit(code)
}

// outputFromCmd runs the command with the given arguments and returns the output of the command along with
// any errors.
func outputFromCmd(parser *kong.Kong, args []string) ([]string, error) {
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}

	ctx, err := parser.Parse(args)
	if err != nil {
		return nil, eris.Wrap(err, "failed to parse command")
	}

	// Capture stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = os.NewFile(1, "stdout")
	os.Stderr = os.NewFile(2, "stderr")
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	if err := ctx.Run(); err != nil {
		return nil, eris.Wrap(err, "command failed")
	}

	lines := strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		return lines, eris.New(errorStr)
	}

	return lines, nil
}

func TestSubcommandsHaveHelpText(t *testing.T) {
	lines, err := outputFromCmd(testEnv.cli, []string{"--help"})
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

	// Create a new Kong parser with our CLI struct
	cli := &cli{
		Doctor: &DoctorCmd{},
	}

	// Parse the command
	parser, err := kong.New(cli)
	assert.NilError(t, err)

	// Execute the doctor command
	ctx, err := parser.Parse([]string{"doctor"})
	assert.NilError(t, err)

	// Run the command
	err = ctx.Run()
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
	var err error

	// Create Cardinal
	gameDir := t.TempDir()
	t.Chdir(gameDir)

	// Create a new Kong parser with our CLI struct
	cli := &cli{
		Create: &CreateCmd{},
	}

	// Parse and execute the create command
	parser, err := kong.New(cli)
	assert.NilError(t, err)

	ctx, err := parser.Parse([]string{"create", gameDir + "/sgt"})
	assert.NilError(t, err)

	err = ctx.Run()
	assert.NilError(t, err)

	// Change dir
	t.Chdir(gameDir + "/sgt")

	// Start cardinal
	cli = &cli{}
	parser, err = kong.New(cli)
	assert.NilError(t, err)

	ctx, err = parser.Parse([]string{"cardinal", "start", "--detach", "--editor=false"})
	assert.NilError(t, err)

	err = ctx.Run()
	assert.NilError(t, err)

	defer func() {
		// Purge cardinal
		ctx, err = parser.Parse([]string{"cardinal", "purge"})
		assert.NilError(t, err)

		err = ctx.Run()
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Restart cardinal
	ctx, err = parser.Parse([]string{"cardinal", "restart", "--detach"})
	assert.NilError(t, err)

	err = ctx.Run()
	assert.NilError(t, err)

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Stop cardinal
	ctx, err = parser.Parse([]string{"cardinal", "stop"})
	assert.NilError(t, err)

	err = ctx.Run()
	assert.NilError(t, err)

	// Check and wait until cardinal shutdowns
	assert.Assert(t, cardinalIsDown(t), "Cardinal is not successfully shutdown")
}

func TestDev(t *testing.T) {
	// Use t.TempDir(), which also auto-cleans the dir
	gameDir := t.TempDir()

	// Change working directory
	t.Chdir(gameDir)

	// Create a new Kong parser with our CLI struct
	cli := &cli{
		Create: &CreateCmd{},
	}

	// Parse and execute the create command
	parser, err := kong.New(cli)
	assert.NilError(t, err)

	ctx, err := parser.Parse([]string{"create", gameDir + "/sgt"})
	assert.NilError(t, err)

	err = ctx.Run()
	assert.NilError(t, err)

	// Start cardinal dev
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cli = &cli{}
	parser, err = kong.New(cli)
	assert.NilError(t, err)

	ctx, err = parser.Parse([]string{"cardinal", "dev", "--editor=false"})
	assert.NilError(t, err)

	go func() {
		err := ctx.Run()
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
	for range 120 {
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
	for range 120 {
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
	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Change to temp dir (auto-resets after test)
	t.Chdir(gameDir)

	// Create a new Kong parser with our CLI struct
	cli := &cli{
		Create: &CreateCmd{},
	}

	// Parse and execute the create command
	parser, err := kong.New(cli)
	assert.NilError(t, err)

	ctx, err := parser.Parse([]string{"create", gameDir + "/sgt"})
	assert.NilError(t, err)

	err = ctx.Run()
	assert.NilError(t, err)

	// Start evm dev
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cli = &cli{}
	parser, err = kong.New(cli)
	assert.NilError(t, err)

	ctx, err = parser.Parse([]string{"evm", "start", "--dev"})
	assert.NilError(t, err)

	go func() {
		err := ctx.Run()
		assert.NilError(t, err)
	}()

	// Check and wait until evm is up
	assert.Assert(t, evmIsUp(t), "EVM is not running")

	// Shutdown the program
	cancel()

	// Check and wait until evm is down
	assert.Assert(t, evmIsDown(t), "EVM is not successfully shutdown")
}
