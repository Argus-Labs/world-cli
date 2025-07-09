package evm

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
	"pkg.world.dev/world-cli/cmd/world/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/world/internal/clients/browser"
	"pkg.world.dev/world-cli/cmd/world/internal/clients/repo"
	cmdsetup "pkg.world.dev/world-cli/cmd/world/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/world/internal/services/config"
	"pkg.world.dev/world-cli/cmd/world/internal/services/input"
	"pkg.world.dev/world-cli/cmd/world/cloud"
	"pkg.world.dev/world-cli/cmd/world/organization"
	"pkg.world.dev/world-cli/cmd/world/project"
	"pkg.world.dev/world-cli/cmd/world/root"
	"pkg.world.dev/world-cli/cmd/world/user"
)

var testCounter uint64

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

// createTestDependencies creates a minimal set of dependencies for testing.
func createTestDependencies() cmdsetup.Dependencies {
	configService, _ := config.NewService("LOCAL")
	inputService := input.NewService()
	repoClient := repo.NewClient()
	apiClient := api.NewClient("", "", "")

	projectHandler := project.NewHandler(
		repoClient,
		configService,
		apiClient,
		&inputService,
	)

	orgHandler := organization.NewHandler(
		projectHandler,
		&inputService,
		apiClient,
		configService,
	)

	userHandler := user.NewHandler(
		apiClient,
		&inputService,
	)

	cloudHandler := cloud.NewHandler(
		apiClient,
		configService,
		projectHandler,
		&inputService,
	)

	setupController := cmdsetup.NewController(
		configService,
		repoClient,
		orgHandler,
		projectHandler,
		apiClient,
		&inputService,
	)

	browserClient := browser.NewClient()
	rootHandler := root.NewHandler("test-version", configService, apiClient, setupController, browserClient)

	// Create EVM handler
	evmHandler := &Handler{}

	return cmdsetup.Dependencies{
		ConfigService:       configService,
		InputService:        &inputService,
		APIClient:           apiClient,
		RepoClient:          repoClient,
		OrganizationHandler: orgHandler,
		ProjectHandler:      projectHandler,
		UserHandler:         userHandler,
		CloudHandler:        cloudHandler,
		SetupController:     setupController,
		RootHandler:         rootHandler,
		EVMHandler:          evmHandler,
	}
}

func TestEVMStart(t *testing.T) {
	t.Skip("Skipping evm start test")

	// Unique namespace and port
	namespace := getUniqueNamespace(t)
	port := getUniquePort(t, 9601)

	// Set environment variables using t.Setenv (auto-cleanup)
	t.Setenv("CARDINAL_NAMESPACE", namespace)
	t.Setenv("EVM_PORT", port)

	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Copy starter game template
	templateDir := filepath.Join(gameDir, "sgt")
	err := copyStarterGameTemplate(t, templateDir)
	assert.NilError(t, err)

	// Change to the template directory (auto-resets after test)
	t.Chdir(templateDir)

	// Start evm dev with longer timeout
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(20*time.Second))
	defer cancel()

	// Create dependencies and CLI commands
	dependencies := createTestDependencies()

	startCmd := StartCmd{
		Parent:       &EvmCmd{},
		UseDevDA:     true,
		Context:      ctx,
		Dependencies: dependencies,
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

func TestEVMStop(t *testing.T) {
	t.Skip("Skipping evm stop test")

	// Unique namespace and port
	namespace := getUniqueNamespace(t)
	port := getUniquePort(t, 9601)

	// Set environment variables using t.Setenv (auto-cleanup)
	t.Setenv("CARDINAL_NAMESPACE", namespace)
	t.Setenv("EVM_PORT", port)

	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Copy starter game template
	templateDir := filepath.Join(gameDir, "sgt")
	err := copyStarterGameTemplate(t, templateDir)
	assert.NilError(t, err)

	// Change to the template directory (auto-resets after test)
	t.Chdir(templateDir)

	// Start EVM first so we have something to stop with longer timeout
	ctx, startCancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer startCancel()

	// Create dependencies and CLI commands
	dependencies := createTestDependencies()

	startCmd := StartCmd{
		Parent:       &EvmCmd{},
		UseDevDA:     true,
		Context:      ctx,
		Dependencies: dependencies,
	}

	// Start EVM in background
	done := make(chan error, 1)
	go func() {
		done <- startCmd.Run()
	}()

	// Wait until EVM is up
	assert.Assert(t, evmIsUp(t), "EVM should be running before we test stop")

	// Now test the Stop command
	stopCmd := StopCmd{
		Parent:       &EvmCmd{},
		Context:      ctx,
		Dependencies: dependencies,
	}

	err = stopCmd.Run()
	assert.NilError(t, err)

	// Verify EVM is down
	assert.Assert(t, evmIsDown(t), "EVM should be stopped after running stop command")

	// Clean up the start goroutine
	startCancel()
	<-done // Wait for start command to finish
}

func TestEVMStartAndStop(t *testing.T) {
	t.Skip("Skipping evm start and stop test")

	// Unique namespace and port
	namespace := getUniqueNamespace(t)
	port := getUniquePort(t, 9601)

	// Set environment variables using t.Setenv (auto-cleanup)
	t.Setenv("CARDINAL_NAMESPACE", namespace)
	t.Setenv("EVM_PORT", port)

	// Create temp working dir (auto-cleans up)
	gameDir := t.TempDir()

	// Copy starter game template
	templateDir := filepath.Join(gameDir, "sgt")
	err := copyStarterGameTemplate(t, templateDir)
	assert.NilError(t, err)

	// Change to the template directory (auto-resets after test)
	t.Chdir(templateDir)

	// Start EVM with longer timeout for container cleanup
	ctx, startCancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer startCancel()

	// Create dependencies and CLI commands
	dependencies := createTestDependencies()

	startCmd := StartCmd{
		Parent:       &EvmCmd{},
		UseDevDA:     true,
		Context:      ctx,
		Dependencies: dependencies,
	}

	// Start EVM in background
	done := make(chan error, 1)
	go func() {
		done <- startCmd.Run()
	}()

	// Wait until EVM is up
	assert.Assert(t, evmIsUp(t), "EVM should be running")

	// Test Stop command
	stopCmd := StopCmd{
		Parent:       &EvmCmd{},
		Context:      ctx,
		Dependencies: dependencies,
	}

	err = stopCmd.Run()
	assert.NilError(t, err)

	// Verify EVM is down
	assert.Assert(t, evmIsDown(t), "EVM should be stopped")

	// Clean up
	startCancel()
	<-done
}
