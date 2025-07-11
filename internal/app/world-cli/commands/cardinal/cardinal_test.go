package cardinal_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/cardinal"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

var testCounter uint64

//nolint:unparam // port is used for testing
func getUniquePort(t *testing.T, basePort int) string {
	t.Helper()
	port := basePort + int(atomic.AddUint64(&testCounter, 1))
	return strconv.Itoa(port)
}

func getUniqueNamespace(t *testing.T) string {
	t.Helper()
	id := atomic.AddUint64(&testCounter, 1)
	return fmt.Sprintf("test-%d", id)
}

func cardinalIsUp(t *testing.T) bool {
	return ServiceIsUp("Cardinal", "localhost:4040", t)
}

func cardinalIsDown(t *testing.T) bool {
	return ServiceIsDown("Cardinal", "localhost:4040", t)
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

// copyStarterGameTemplate copies the starter game template to the target directory.
func copyStarterGameTemplate(t *testing.T, targetDir string) error {
	t.Helper()

	// Get the project root directory by finding the go.mod file
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Walk up the directory tree to find the project root (where go.mod exists)
	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			return errors.New("could not find project root (no go.mod found)")
		}
		projectRoot = parent
	}

	starterTemplateDir := filepath.Join(projectRoot, "testdata", "starter-game-template")

	// Check if the source directory exists
	if _, err := os.Stat(starterTemplateDir); os.IsNotExist(err) {
		return fmt.Errorf("starter game template not found at: %s", starterTemplateDir)
	}

	// Copy the directory
	err = copyDir(starterTemplateDir, targetDir)
	if err != nil {
		return fmt.Errorf("failed to copy starter game template: %w", err)
	}

	return nil
}

// copyDir copies a directory recursively.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

func TestCardinalStartStopRestartPurge(t *testing.T) {
	t.Skip("Skipping cardinal start stop restart purge test")

	// Unique namespace and port
	namespace := getUniqueNamespace(t)
	port := getUniquePort(t, 4040)

	// Set environment variables using t.Setenv (auto-cleanup)
	t.Setenv("CARDINAL_NAMESPACE", namespace)
	t.Setenv("CARDINAL_PORT", port)

	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Copy starter game template
	templateDir := filepath.Join(gameDir, "sgt")
	err := copyStarterGameTemplate(t, templateDir)
	assert.NilError(t, err)

	// Change to the template directory (auto-resets after test)
	t.Chdir(templateDir)

	cardinalHandler := &cardinal.Handler{}
	startFlags := models.StartCardinalFlags{
		Editor: false,
		Detach: true,
	}

	// Start cardinal
	err = cardinalHandler.Start(t.Context(), startFlags)
	assert.NilError(t, err)

	defer func() {
		// Purge cardinal
		purgeFlags := models.PurgeCardinalFlags{}
		err = cardinalHandler.Purge(t.Context(), purgeFlags)
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Restart cardinal
	restartFlags := models.RestartCardinalFlags{
		Detach: true,
	}
	err = cardinalHandler.Restart(t.Context(), restartFlags)
	assert.NilError(t, err)

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Stop cardinal
	stopFlags := models.StopCardinalFlags{}
	err = cardinalHandler.Stop(t.Context(), stopFlags)
	assert.NilError(t, err)

	// Check and wait until cardinal shutdowns
	assert.Assert(t, cardinalIsDown(t), "Cardinal is not successfully shutdown")
}

func TestCardinalDev(t *testing.T) {
	t.Skip("Skipping cardinal dev test")
	// Unique namespace and port
	namespace := getUniqueNamespace(t)
	port := getUniquePort(t, 4040)

	// Set environment variables using t.Setenv (auto-cleanup)
	t.Setenv("CARDINAL_NAMESPACE", namespace)
	t.Setenv("CARDINAL_PORT", port)

	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Copy starter game template
	templateDir := filepath.Join(gameDir, "sgt")
	err := copyStarterGameTemplate(t, templateDir)
	assert.NilError(t, err)

	// Change to the template directory (auto-resets after test)
	t.Chdir(templateDir)

	// Start cardinal dev with a 10-second timeout
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	cardinalHandler := &cardinal.Handler{}
	devFlags := models.DevCardinalFlags{
		Editor: false,
	}

	done := make(chan error, 1)
	go func() {
		done <- cardinalHandler.Dev(ctx, devFlags)
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

func TestCardinalStart(t *testing.T) {
	// Unique namespace and port
	t.Skip("Skipping cardinal start test")

	namespace := getUniqueNamespace(t)
	port := getUniquePort(t, 4040)

	// Set environment variables using t.Setenv (auto-cleanup)
	t.Setenv("CARDINAL_NAMESPACE", namespace)
	t.Setenv("CARDINAL_PORT", port)

	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Copy starter game template
	templateDir := filepath.Join(gameDir, "sgt")
	err := copyStarterGameTemplate(t, templateDir)
	assert.NilError(t, err)

	// Change to the template directory (auto-resets after test)
	t.Chdir(templateDir)

	cardinalHandler := &cardinal.Handler{}
	startFlags := models.StartCardinalFlags{
		Editor: false,
		Detach: true,
	}

	// Start Cardinal in background
	done := make(chan error, 1)
	go func() {
		done <- cardinalHandler.Start(t.Context(), startFlags)
	}()

	// Wait until Cardinal is up
	assert.Assert(t, cardinalIsUp(t), "Cardinal should be running")

	// Stop Cardinal
	stopFlags := models.StopCardinalFlags{}
	err = cardinalHandler.Stop(t.Context(), stopFlags)
	assert.NilError(t, err)

	// Verify Cardinal is down
	assert.Assert(t, cardinalIsDown(t), "Cardinal should be stopped")

	// Clean up
	<-done
}

func TestCardinalBuild(t *testing.T) {
	// Unique namespace and port
	t.Skip("Skipping cardinal build test")

	namespace := getUniqueNamespace(t)
	port := getUniquePort(t, 4040)

	// Set environment variables using t.Setenv (auto-cleanup)
	t.Setenv("CARDINAL_NAMESPACE", namespace)
	t.Setenv("CARDINAL_PORT", port)

	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Copy starter game template
	templateDir := filepath.Join(gameDir, "sgt")
	err := copyStarterGameTemplate(t, templateDir)
	assert.NilError(t, err)

	// Change to the template directory (auto-resets after test)
	t.Chdir(templateDir)

	cardinalHandler := &cardinal.Handler{}
	buildFlags := models.BuildCardinalFlags{
		LogLevel: "info",
	}

	// Build cardinal
	err = cardinalHandler.Build(t.Context(), buildFlags)
	assert.NilError(t, err)

	// Cleanup - purge any containers that might have been created during build
	defer func() {
		purgeFlags := models.PurgeCardinalFlags{}
		_ = cardinalHandler.Purge(t.Context(), purgeFlags) // Ignore errors in cleanup
	}()
}
