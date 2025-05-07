package docker

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/teacmd"
)

const (
	redisPort         = "56379"
	redisPassword     = "password"
	cardinalNamespace = "test"
)

func TestMain(m *testing.M) {
	// Purge any existing containers
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": cardinalNamespace,
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
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": cardinalNamespace,
			"REDIS_PASSWORD":     redisPassword,
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
	assert.Assert(t, redislIsUp(t))
}

func TestStop(t *testing.T) {
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": cardinalNamespace,
			"REDIS_PASSWORD":     redisPassword,
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
	assert.Assert(t, redisIsDown(t))
}

func TestRestart(t *testing.T) {
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": cardinalNamespace,
			"REDIS_PASSWORD":     redisPassword,
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
	assert.Assert(t, redislIsUp(t))
}

func TestPurge(t *testing.T) {
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": cardinalNamespace,
			"REDIS_PASSWORD":     redisPassword,
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
	assert.Assert(t, redisIsDown(t))
}

func TestStartUndetach(t *testing.T) {
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": cardinalNamespace,
			"REDIS_PASSWORD":     redisPassword,
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
	assert.Assert(t, redislIsUp(t))

	cancel()
	assert.Assert(t, redisIsDown(t))
}

func TestBuild(t *testing.T) {
	// Create a temporary directory
	dir := t.TempDir()

	// Change to the temporary directory
	t.Chdir(dir)

	sgtDir := dir + "/sgt"

	// Pull the repository
	templateGitURL := "https://github.com/Argus-Labs/starter-game-template.git"
	err := teacmd.GitCloneCmd(templateGitURL, sgtDir, "Initial commit from World CLI")
	assert.NilError(t, err)

	// Preparation
	cfg := &config.Config{
		DockerEnv: map[string]string{
			"CARDINAL_NAMESPACE": cardinalNamespace,
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

func redislIsUp(t *testing.T) bool {
	up := false
	for i := 0; i < 60; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:"+redisPort, time.Second)
		if err != nil {
			time.Sleep(time.Second)
			t.Logf("Failed to connect to Redis at localhost:%s, retrying...", redisPort)
			continue
		}
		_ = conn.Close()
		up = true
		break
	}
	return up
}

func redisIsDown(t *testing.T) bool {
	down := false
	for i := 0; i < 60; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:"+redisPort, time.Second)
		if err != nil {
			down = true
			break
		}
		_ = conn.Close()
		time.Sleep(time.Second)
		t.Logf("Redis is still running at localhost:%s, retrying...", redisPort)
		continue
	}
	return down
}

func cleanUp(t *testing.T, dockerClient *Client) {
	t.Cleanup(func() {
		//nolint:usetesting // don't use t.Context() here; it's canceled during cleanup
		assert.NilError(t, dockerClient.Purge(context.Background(), service.Nakama,
			service.Cardinal, service.Redis, service.NakamaDB, service.Jaeger, service.Prometheus),
			"Failed to purge container during cleanup")

		assert.NilError(t, dockerClient.Close())
	})
}
