package docker

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/config"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/teacmd"
	"pkg.world.dev/world-cli/internal/pkg/logger"
)

var (
	// counter for generating unique test IDs.
	testCounter uint64
)

// getUniquePort returns a unique port number for testing.
func getUniquePort(t *testing.T) string {
	t.Helper()
	// Use atomic counter to generate unique port
	basePort := 56379
	maxAttempts := 1000 // Try up to 1000 different ports

	for i := 0; i < maxAttempts; i++ {
		port := basePort + int(atomic.AddUint64(&testCounter, 1))
		portStr := strconv.Itoa(port)

		// Check if port is available
		addr := net.JoinHostPort("localhost", portStr)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			return portStr
		}
	}

	t.Fatal("Failed to find an available port after", maxAttempts, "attempts")
	return "" // This line will never be reached due to t.Fatal above
}

// getUniqueNamespace returns a unique namespace for testing.
func getUniqueNamespace(t *testing.T) string {
	t.Helper()
	// Use atomic counter to generate unique namespace
	id := atomic.AddUint64(&testCounter, 1)
	return fmt.Sprintf("test-%d", id)
}

func TestMain(m *testing.M) {
	// Purge any existing containers
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": "test-main", // Use a fixed namespace for TestMain
		},
	}

	dockerClient, err := NewClient(cfg)
	if err != nil {
		logger.Errorf("Failed to create docker client: %v", err)
		os.Exit(1)
	}
	err = dockerClient.Purge(context.Background(), service.Nakama,
		service.Cardinal, service.Redis, service.NakamaDB, service.Jaeger, service.Prometheus)
	if err != nil {
		logger.Errorf("Failed to purge containers: %v", err)
		os.Exit(1)
	}
	// Run the tests
	code := m.Run()
	err = dockerClient.Close()
	if err != nil {
		logger.Errorf("Failed to close docker client: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func TestStart(t *testing.T) {
	t.Parallel()

	redisPort := getUniquePort(t)
	namespace := getUniqueNamespace(t)

	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": namespace,
			"REDIS_PASSWORD":     "password",
			"REDIS_PORT":         redisPort,
		},
		Detach: true,
	}
	dockerClient, err := NewClient(cfg)
	assert.NilError(t, err, "Failed to create docker client")
	ctx := t.Context()
	assert.NilError(t, dockerClient.Start(ctx, service.Redis), "failed to start container")
	cleanUp(t, dockerClient)

	// Test if the container is running
	assert.Assert(t, redisIsUp(t, redisPort))
}

func TestStop(t *testing.T) {
	t.Parallel()

	redisPort := getUniquePort(t)
	namespace := getUniqueNamespace(t)

	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": namespace,
			"REDIS_PASSWORD":     "password",
			"REDIS_PORT":         redisPort,
		},
		Detach: true,
	}
	dockerClient, err := NewClient(cfg)
	assert.NilError(t, err, "Failed to create docker client")
	ctx := t.Context()
	assert.NilError(t, dockerClient.Start(ctx, service.Redis), "failed to start container")
	cleanUp(t, dockerClient)
	assert.NilError(t, dockerClient.Stop(ctx, service.Redis), "failed to stop container")

	// Test if the container is stopped
	assert.Assert(t, redisIsDown(t, redisPort))
}

func TestRestart(t *testing.T) {
	t.Parallel()

	redisPort := getUniquePort(t)
	namespace := getUniqueNamespace(t)

	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": namespace,
			"REDIS_PASSWORD":     "password",
			"REDIS_PORT":         redisPort,
		},
		Detach: true,
	}
	dockerClient, err := NewClient(cfg)
	assert.NilError(t, err, "Failed to create docker client")
	ctx := t.Context()
	assert.NilError(t, dockerClient.Start(ctx, service.Redis), "failed to start container")
	cleanUp(t, dockerClient)
	assert.NilError(t, dockerClient.Restart(ctx, service.Redis), "failed to restart container")

	// Test if the container is running
	assert.Assert(t, redisIsUp(t, redisPort))
}

func TestPurge(t *testing.T) {
	t.Parallel()

	redisPort := getUniquePort(t)
	namespace := getUniqueNamespace(t)

	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": namespace,
			"REDIS_PASSWORD":     "password",
			"REDIS_PORT":         redisPort,
		},
		Detach: true,
	}
	dockerClient, err := NewClient(cfg)
	assert.NilError(t, err, "Failed to create docker client")
	ctx := t.Context()
	assert.NilError(t, dockerClient.Start(ctx, service.Redis), "failed to start container")
	assert.NilError(t, dockerClient.Purge(ctx, service.Redis), "failed to purge container")

	// Test if the container is stopped
	assert.Assert(t, redisIsDown(t, redisPort))
}

func TestStartUndetach(t *testing.T) {
	t.Parallel()

	redisPort := getUniquePort(t)
	namespace := getUniqueNamespace(t)

	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": namespace,
			"REDIS_PASSWORD":     "password",
			"REDIS_PORT":         redisPort,
		},
	}
	dockerClient, err := NewClient(cfg)
	assert.NilError(t, err, "Failed to create docker client")
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		assert.NilError(t, dockerClient.Start(ctx, service.Redis), "failed to start container")
		cleanUp(t, dockerClient)
	}()
	assert.Assert(t, redisIsUp(t, redisPort))

	cancel()
	assert.Assert(t, redisIsDown(t, redisPort))
}

func TestBuild(t *testing.T) {
	t.Parallel()

	// Skip test if GitHub token is not available
	if os.Getenv("ARGUS_WEV2_GITHUB_TOKEN") == "" {
		t.Skip("Skipping build test - ARGUS_WEV2_GITHUB_TOKEN not set")
	}

	namespace := getUniqueNamespace(t)

	// Create a temporary directory
	dir := t.TempDir()
	sgtDir := filepath.Join(dir, "sgt")

	// Pull the repository
	templateGitURL := "https://github.com/Argus-Labs/starter-game-template.git"
	err := teacmd.GitCloneCmd(templateGitURL, sgtDir, "Initial commit from World CLI")
	assert.NilError(t, err)
	// Preparation
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": namespace,
		},
		RootDir: sgtDir,
	}
	cardinalService := service.Cardinal(cfg)
	ctx := t.Context()
	dockerClient, err := NewClient(cfg)
	assert.NilError(t, err, "Failed to create docker client")
	// Pull prerequisite images
	assert.NilError(t, dockerClient.pullImages(ctx, cardinalService))
	// Build the image
	_, err = dockerClient.buildImage(ctx, cardinalService)
	assert.NilError(t, err, "Failed to build Docker image")
}

func redisIsUp(t *testing.T, port string) bool {
	t.Helper()
	up := false
	for i := 0; i < 60; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:"+port, time.Second)
		if err != nil {
			time.Sleep(time.Second)
			t.Logf("Failed to connect to Redis at localhost:%s, retrying...", port)
			continue
		}
		_ = conn.Close()
		up = true
		break
	}
	return up
}

func redisIsDown(t *testing.T, port string) bool {
	t.Helper()
	down := false
	for i := 0; i < 60; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:"+port, time.Second)
		if err != nil {
			down = true
			break
		}
		_ = conn.Close()
		time.Sleep(time.Second)
		t.Logf("Redis is still running at localhost:%s, retrying...", port)
		continue
	}
	return down
}
func cleanUp(t *testing.T, dockerClient *Client) {
	t.Cleanup(func() {
		assert.NilError(t, dockerClient.Purge(context.Background(), service.Nakama,
			service.Cardinal, service.Redis, service.NakamaDB, service.Jaeger, service.Prometheus),
			"Failed to purge container during cleanup")
		assert.NilError(t, dockerClient.Close())
	})
}
