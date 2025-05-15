package root

import (
	"context"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"pkg.world.dev/world-cli/cmd/world/cardinal"
	"pkg.world.dev/world-cli/cmd/world/evm"
)

// TestMain runs before all tests and handles setup/teardown.
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Teardown if needed
	// Add any cleanup code here

	os.Exit(code)
}

//nolint:reassign // reassigning os package output is needed to capture the output
func captureOutput(f func() error) ([]string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		w.Close()
	}()

	if err := f(); err != nil { // run the command
		return nil, err
	}
	w.Close()

	out, _ := io.ReadAll(r)
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, "\r")
	}

	return lines, nil
}

func TestExecuteDoctorCommand(t *testing.T) {
	cmd := DoctorCmd{}

	lines, err := captureOutput(cmd.Run)
	assert.NilError(t, err)

	seenDependencies := map[string]int{
		"Git":                      0,
		"Go":                       0,
		"Docker":                   0,
		"Docker daemon is running": 0,
	}

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
	createCmd := CreateCmd{
		Directory: gameDir + "/sgt",
	}
	err = createCmd.Run()
	assert.NilError(t, err)

	// Change dir
	t.Chdir(gameDir + "/sgt")

	// Start cardinal
	startCmd := cardinal.StartCmd{ // cardinal start --detach --editor=false
		Editor: false,
		Detach: true,
	}
	err = startCmd.Run()
	assert.NilError(t, err)

	defer func() {
		// Purge cardinal
		purgeCmd := cardinal.PurgeCmd{}
		err = purgeCmd.Run()
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Restart cardinal
	restartCmd := cardinal.RestartCmd{ // cardinal restart --detach
		Detach: true,
	}
	err = restartCmd.Run()
	assert.NilError(t, err)

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Stop cardinal
	stopCmd := cardinal.StopCmd{}
	err = stopCmd.Run()
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
	createCmd := CreateCmd{
		Directory: gameDir + "/sgt",
	}
	err := createCmd.Run() // cardinal create {dir}/sgt
	assert.NilError(t, err)

	// Start cardinal dev with a 10-second timeout
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	devCmd := cardinal.DevCmd{ // cardinal dev --editor=false
		Editor:  false,
		Context: ctx,
	}

	done := make(chan error, 1)
	go func() {
		done <- devCmd.Run()
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Wait for either the process to finish or the timeout to occur
	select {
	case <-ctx.Done():
		// Timeout reached, process should be killed by context cancellation
	case err := <-done:
		assert.NilError(t, err)
	}

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

	createCmd := CreateCmd{
		Directory: gameDir + "/sgt",
	}
	err := createCmd.Run() // evm create {dir}/sgt
	assert.NilError(t, err)

	// Start evm dev
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(10*time.Second))
	defer cancel()

	startCmd := evm.StartCmd{
		UseDevDA: true,
		Context:  ctx,
	}
	done := make(chan error, 1)
	go func() {
		done <- startCmd.Run() // evm start --dev
	}()

	// Check and wait until evm is up
	assert.Assert(t, evmIsUp(t), "EVM is not running")

	select {
	case <-ctx.Done():
		// Timeout reached, process should be killed by context cancellation
	case err := <-done:
		assert.NilError(t, err)
	}

	// Check and wait until evm is down
	assert.Assert(t, evmIsDown(t), "EVM is not successfully shutdown")
}
