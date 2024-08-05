package docker

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"pkg.world.dev/world-cli/common/config"
)

type GetDockerConfig func(cfg *config.Config) Config

// DockerConfig is a configuration for a docker container
// It contains the name of the container and a function to get the container and host config
type Config struct {
	Name string
	*container.Config
	*container.HostConfig
	*network.NetworkingConfig
	*ocispec.Platform
	*Dockerfile
}

type Dockerfile struct {
	Script string
	Target string
}

func getRedisContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-redis", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func getCardinalContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-cardinal", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func getNakamaContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-nakama", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func getNakamaDBContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-nakama-db", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func getCelestiaDevNetContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-celestia-devnet", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func getEVMContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-evm", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func Redis(cfg *config.Config) Config {
	return Config{
		Name: getRedisContainerName(cfg),
		Config: &container.Config{
			Image: "redis:latest",
			Env: []string{
				fmt.Sprintf("REDIS_PASSWORD=%s", cfg.DockerEnv["REDIS_PASSWORD"]),
			},
			ExposedPorts: nat.PortSet{
				"6379/tcp": struct{}{},
			},
		},
		HostConfig: &container.HostConfig{
			PortBindings: nat.PortMap{
				"6379/tcp": []nat.PortBinding{{HostPort: "6379"}},
			},
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			Mounts:        []mount.Mount{{Type: mount.TypeVolume, Source: "data", Target: "/redis"}},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}

func Cardinal(cfg *config.Config) Config {
	imageName := cfg.DockerEnv["CARDINAL_NAMESPACE"]
	containerConfig := container.Config{
		Image: imageName,
		Env: []string{
			fmt.Sprintf("REDIS_ADDRESS=%s:6379", getRedisContainerName(cfg)),
			fmt.Sprintf("BASE_SHARD_SEQUENCER_ADDRESS=%s:9601", getEVMContainerName(cfg)),
		},
		ExposedPorts: nat.PortSet{
			"4040/tcp": struct{}{},
		},
	}

	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{
			"4040/tcp": []nat.PortBinding{{HostPort: "4040"}},
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
	}

	dockerfile := Dockerfile{Script: cardinalDockerfile, Target: "runtime"}

	// Add debug options
	debug := cfg.Debug
	if debug {
		containerConfig.ExposedPorts["40000/tcp"] = struct{}{}
		hostConfig.PortBindings["40000/tcp"] = []nat.PortBinding{{HostPort: "40000"}}
		hostConfig.CapAdd = []string{"SYS_PTRACE"}
		hostConfig.SecurityOpt = []string{"seccomp:unconfined"}
		dockerfile.Target = "runtime-debug"
	}

	return Config{
		Name:       getCardinalContainerName(cfg),
		Config:     &containerConfig,
		HostConfig: &hostConfig,
		Dockerfile: &dockerfile,
	}
}

func Nakama(cfg *config.Config) Config {
	return Config{
		Name: getNakamaContainerName(cfg),
		Config: &container.Config{
			Image: "ghcr.io/argus-labs/world-engine-nakama:1.2.7",
			Env: []string{
				fmt.Sprintf("CARDINAL_CONTAINER=%s", getCardinalContainerName(cfg)),
				fmt.Sprintf("CARDINAL_ADDR=%s:4040", getCardinalContainerName(cfg)),
				fmt.Sprintf("CARDINAL_NAMESPACE=%s", cfg.DockerEnv["CARDINAL_NAMESPACE"]),
				fmt.Sprintf("DB_PASSWORD=%s", cfg.DockerEnv["DB_PASSWORD"]),
				"ENABLE_ALLOWLIST=false",
				"OUTGOING_QUEUE_SIZE=64",
			},
			Entrypoint: []string{
				"/bin/sh",
				"-ec",
				fmt.Sprintf("/nakama/nakama migrate up --database.address root:%s@%s:26257/nakama && /nakama/nakama --config /nakama/data/local.yml --database.address root:%s@%s:26257/nakama --socket.outgoing_queue_size=64 --logger.level INFO", //nolint:lll
					cfg.DockerEnv["DB_PASSWORD"],
					getNakamaDBContainerName(cfg),
					cfg.DockerEnv["DB_PASSWORD"],
					getNakamaDBContainerName(cfg)),
			},
			ExposedPorts: nat.PortSet{
				"7349/tcp": struct{}{},
				"7350/tcp": struct{}{},
				"7351/tcp": struct{}{},
			},
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "/nakama/nakama", "healthcheck"},
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
				Retries:  20, //nolint:gomnd
			},
		},
		HostConfig: &container.HostConfig{
			PortBindings: nat.PortMap{
				"7349/tcp": []nat.PortBinding{{HostPort: "7349"}},
				"7350/tcp": []nat.PortBinding{{HostPort: "7350"}},
				"7351/tcp": []nat.PortBinding{{HostPort: "7351"}},
			},
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
		Platform: &ocispec.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
	}
}

func NakamaDB(cfg *config.Config) Config {
	return Config{
		Name: getNakamaDBContainerName(cfg),
		Config: &container.Config{
			Image: "cockroachdb/cockroach:latest-v23.1",
			Cmd:   []string{"start-single-node", "--insecure", "--store=attrs=ssd,path=/var/lib/cockroach/,size=20%"},
			Env: []string{
				"COCKROACH_DATABASE=nakama",
				"COCKROACH_USER=root",
				fmt.Sprintf("COCKROACH_PASSWORD=%s", cfg.DockerEnv["DB_PASSWORD"]),
			},
			ExposedPorts: nat.PortSet{
				"26257/tcp": struct{}{},
				"8080/tcp":  struct{}{},
			},
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "curl", "-f", "http://localhost:8080/health?ready=1"},
				Interval: 3 * time.Second, //nolint:gomnd
				Timeout:  3 * time.Second, //nolint:gomnd
				Retries:  5,               //nolint:gomnd
			},
		},
		HostConfig: &container.HostConfig{
			PortBindings: nat.PortMap{
				"26257/tcp": []nat.PortBinding{{HostPort: "26257"}},
				"8080/tcp":  []nat.PortBinding{{HostPort: "8080"}},
			},
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			Mounts: []mount.Mount{{Type: mount.TypeVolume, Source: cfg.DockerEnv["CARDINAL_NAMESPACE"],
				Target: "/var/lib/cockroach"}},
			NetworkMode: container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}

func CelestiaDevNet(cfg *config.Config) Config {
	return Config{
		Name: getCelestiaDevNetContainerName(cfg),
		Config: &container.Config{
			Image: "ghcr.io/rollkit/local-celestia-devnet:latest",
			ExposedPorts: nat.PortSet{
				"26657/tcp": struct{}{},
				"26658/tcp": struct{}{},
				"26659/tcp": struct{}{},
				"9090/tcp":  struct{}{},
			},
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD", "curl", "-f", "http://127.0.0.1:26659/head"},
				Interval: 1 * time.Second,
				Timeout:  1 * time.Second,
				Retries:  20, //nolint:gomnd
			},
		},
		HostConfig: &container.HostConfig{
			PortBindings: nat.PortMap{
				"26657/tcp": []nat.PortBinding{{HostPort: "26657"}},
				"26658/tcp": []nat.PortBinding{{HostPort: "26658"}},
				"26659/tcp": []nat.PortBinding{{HostPort: "26659"}},
				"9090/tcp":  []nat.PortBinding{{HostPort: "9090"}},
			},
			RestartPolicy: container.RestartPolicy{Name: "on-failure"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}

func EVM(cfg *config.Config) Config {
	return Config{
		Name: getEVMContainerName(cfg),
		Config: &container.Config{
			Image: "ghcr.io/argus-labs/world-engine-evm:1.4.1",
			Env: []string{
				"DA_BASE_URL=http://celestia-devnet",
				"DA_AUTH_TOKEN=",
				"FAUCET_ENABLED=false",
				"FAUCET_ADDRESS=aa9288F88233Eb887d194fF2215Cf1776a6FEE41",
				"FAUCET_AMOUNT=0x56BC75E2D63100000",
				"BASE_SHARD_ROUTER_KEY=abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01",
			},
			ExposedPorts: nat.PortSet{
				"1317/tcp":  struct{}{},
				"26657/tcp": struct{}{},
				"9090/tcp":  struct{}{},
				"9601/tcp":  struct{}{},
				"8545/tcp":  struct{}{},
			},
		},
		HostConfig: &container.HostConfig{
			PortBindings: nat.PortMap{
				"1317/tcp":  []nat.PortBinding{{HostPort: "1317"}},
				"26657/tcp": []nat.PortBinding{{HostPort: "26657"}},
				"9090/tcp":  []nat.PortBinding{{HostPort: "9090"}},
				"9601/tcp":  []nat.PortBinding{{HostPort: "9601"}},
				"8545/tcp":  []nat.PortBinding{{HostPort: "8545"}},
			},
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
