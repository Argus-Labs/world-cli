package service

import (
	_ "embed"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/config"
	"pkg.world.dev/world-cli/internal/pkg/logger"
)

const (
// mountCache is the Docker mount command to cache the go build cache.
// mountCacheScript = `--mount=type=cache,target="/root/.cache/go-build"`
// mountSecret is the Docker mount command for GitHub token secret.
// mountSecretScript = `--mount=type=secret,id=github_token`
// gitConfigWithSecret is the git config command that uses the secret mount.
//
//	gitConfigWithSecret = `RUN --mount=type=secret,id=github_token \
//	  git config --global \
//	  url."https://$(cat /run/secrets/github_token):x-oauth-basic@github.com/".insteadOf "https://github.com/"`
//
// gitConfigWithEnv is the git config command that uses environment variable.
//
//	gitConfigWithEnv = `RUN git config --global \
//	  url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"`
)

//nolint:gochecknoglobals // dockerfileContent is embedded at compile time and is read-only
//go:embed cardinal.Dockerfile
var dockerfileContent string

func getCardinalContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-cardinal", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Cardinal(cfg *config.Config) Service { //nolint:funlen // it does what it needs to do
	// Check cardinal namespace
	checkCardinalNamespace(cfg)

	exposedPorts := []int{4040}

	runtime := "runtime"
	if cfg.Debug {
		runtime = "runtime-debug"
	}

	dockerfile := dockerfileContent
	if !BuildkitSupport {
		// When BuildKit is disabled, we use the standard Dockerfile without BuildKit-specific features
		// This is less secure as it embeds the GitHub token in image layers, but allows for debugging
		if logger.VerboseMode {
			logger.Printf("BuildKit disabled - using standard Docker build for Cardinal\r\n")
		}
	}

	// Set env variables
	const falseValue = "false"

	// Set Base Shard Router Key
	baseShardRouterKey := cfg.DockerEnv["BASE_SHARD_ROUTER_KEY"]
	if baseShardRouterKey == "" {
		baseShardRouterKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01"
	}

	// Set Cardinal Log Level
	cardinalLogLevel := cfg.DockerEnv["CARDINAL_LOG_LEVEL"]
	if cardinalLogLevel == "" {
		cardinalLogLevel = "info"
	}

	// Set Cardinal Log Pretty
	cardinalLogPretty := cfg.DockerEnv["CARDINAL_LOG_PRETTY"]
	if cardinalLogPretty == "" {
		cardinalLogPretty = "true"
	}

	// Set Cardinal Rollup Enabled
	cardinalRollupEnabled := cfg.DockerEnv["CARDINAL_ROLLUP_ENABLED"]
	if cardinalRollupEnabled == "" {
		cardinalRollupEnabled = falseValue
	}

	// Set Telemetry Profiler Enabled
	telemetryProfilerEnabled := cfg.DockerEnv["TELEMETRY_PROFILER_ENABLED"]
	if telemetryProfilerEnabled == "" {
		telemetryProfilerEnabled = falseValue
	}

	// Set telemetry trace enabled
	telemetryTraceEnabled := cfg.DockerEnv["TELEMETRY_TRACE_ENABLED"]
	if telemetryTraceEnabled == "" {
		telemetryTraceEnabled = falseValue
	}

	// Set router key
	routerKey := cfg.DockerEnv["ROUTER_KEY"]
	if routerKey == "" {
		routerKey = "25a0f627050d11b1461b2728ea3f704e141312b1d4f2a21edcec4eccddd940c2"
	}

	service := Service{
		Name: getCardinalContainerName(cfg),
		Config: container.Config{
			Image: cfg.DockerEnv["CARDINAL_NAMESPACE"],
			Env: []string{
				fmt.Sprintf("REDIS_ADDRESS=%s:6379", getRedisContainerName(cfg)),
				fmt.Sprintf("BASE_SHARD_SEQUENCER_ADDRESS=%s:9601", getEVMContainerName(cfg)),
				fmt.Sprintf("BASE_SHARD_ROUTER_KEY=%s", baseShardRouterKey),
				fmt.Sprintf("CARDINAL_LOG_LEVEL=%s", cardinalLogLevel),
				fmt.Sprintf("CARDINAL_LOG_PRETTY=%s", cardinalLogPretty),
				fmt.Sprintf("CARDINAL_ROLLUP_ENABLED=%s", cardinalRollupEnabled),
				fmt.Sprintf("TELEMETRY_PROFILER_ENABLED=%s", telemetryProfilerEnabled),
				fmt.Sprintf("TELEMETRY_TRACE_ENABLED=%s", telemetryTraceEnabled),
				fmt.Sprintf("ROUTER_KEY=%s", routerKey),
			},
			ExposedPorts: getExposedPorts(exposedPorts),
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap(exposedPorts),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
		Dockerfile:  dockerfile,
		BuildTarget: runtime,
		Dependencies: []Service{
			{
				Name: "golang:1.23-bookworm",
				Config: container.Config{
					Image: "golang:1.23-bookworm",
				},
			},
			{
				Name: "gcr.io/distroless/base-debian12",
				Config: container.Config{
					Image: "gcr.io/distroless/base-debian12",
				},
			},
		},
	}

	// Add debug options
	debug := cfg.Debug
	if debug {
		service.ExposedPorts["40000/tcp"] = struct{}{}
		service.PortBindings["40000/tcp"] = []nat.PortBinding{{HostPort: "40000"}}
		service.CapAdd = []string{"SYS_PTRACE"}
		service.SecurityOpt = []string{"seccomp:unconfined"}
	}

	return service
}
