package service

import (
	"fmt"

	"github.com/docker/docker/api/types/container"

	"pkg.world.dev/world-cli/common/config"
)

func getEVMContainerName(cfg *config.Config) string {
	return fmt.Sprintf("%s-evm", cfg.DockerEnv["CARDINAL_NAMESPACE"])
}

func EVM(cfg *config.Config) Service {
	// Check cardinal namespace
	checkCardinalNamespace(cfg)

	daBaseURL := cfg.DockerEnv["DA_BASE_URL"]
	if daBaseURL == "" || cfg.DevDA {
		daBaseURL = fmt.Sprintf("http://%s", getCelestiaDevNetContainerName(cfg))
	}

	faucetEnabled := cfg.DockerEnv["FAUCET_ENABLED"]
	if faucetEnabled == "" {
		faucetEnabled = "false"
	}

	faucetAddress := cfg.DockerEnv["FAUCET_ADDRESS"]
	if faucetAddress == "" {
		faucetAddress = "aa9288F88233Eb887d194fF2215Cf1776a6FEE41"
	}

	faucetAmount := cfg.DockerEnv["FAUCET_AMOUNT"]
	if faucetAmount == "" {
		faucetAmount = "0x56BC75E2D63100000"
	}

	baseShardRouterKey := cfg.DockerEnv["BASE_SHARD_ROUTER_KEY"]
	if baseShardRouterKey == "" {
		baseShardRouterKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01"
	}

	return Service{
		Name: getEVMContainerName(cfg),
		Config: container.Config{
			Image: "ghcr.io/argus-labs/world-engine-evm:1.4.1",
			Env: []string{
				fmt.Sprintf("DA_BASE_URL=%s", daBaseURL),
				fmt.Sprintf("DA_AUTH_TOKEN=%s", cfg.DockerEnv["DA_AUTH_TOKEN"]),
				fmt.Sprintf("FAUCET_ENABLED=%s", faucetEnabled),
				fmt.Sprintf("FAUCET_ADDRESS=%s", faucetAddress),
				fmt.Sprintf("FAUCET_AMOUNT=%s", faucetAmount),
				fmt.Sprintf("BASE_SHARD_ROUTER_KEY=%s", baseShardRouterKey),
			},
			ExposedPorts: getExposedPorts([]int{1317, 26657, 9090, 9601}),
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap([]int{1317, 26657, 9090, 9601, 8545}),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
	}
}
