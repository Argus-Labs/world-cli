package service

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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

	daNamespaceID := cfg.DockerEnv["DA_NAMESPACE_ID"]
	if daNamespaceID == "" {
		daNamespaceID = "67480c4a88c4d12935d4"
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

	routerKey := cfg.DockerEnv["ROUTER_KEY"]
	if routerKey == "" {
		routerKey = "25a0f627050d11b1461b2728ea3f704e141312b1d4f2a21edcec4eccddd940c2"
	}

	chainID := cfg.DockerEnv["CHAIN_ID"]
	if chainID == "" {
		chainID = "world-420"
	}

	chainKeyMnemonic := cfg.DockerEnv["CHAIN_KEY_MNEMONIC"]
	if chainKeyMnemonic == "" {
		chainKeyMnemonic = "enact adjust liberty squirrel bulk ticket invest tissue antique window" +
			"thank slam unknown fury script among bread social switch glide wool clog flag enroll"
	}

	evmImage := "ghcr.io/argus-labs/world-engine-evm:latest"
	if cfg.DockerEnv["EVM_IMAGE"] != "" {
		evmImage = cfg.DockerEnv["EVM_IMAGE"]
	}

	var platform ocispec.Platform
	if cfg.DockerEnv["EVM_IMAGE_PLATFORM"] != "" {
		evmImagePlatform := strings.Split(cfg.DockerEnv["EVM_IMAGE_PLATFORM"], "/")
		if len(evmImagePlatform) == 2 { //nolint:gomnd //2 is the expected length
			platform = ocispec.Platform{
				Architecture: evmImagePlatform[1],
				OS:           evmImagePlatform[0],
			}
		}
	}

	return Service{
		Name: getEVMContainerName(cfg),
		Config: container.Config{
			Image: evmImage,
			Env: []string{
				fmt.Sprintf("DA_BASE_URL=%s", daBaseURL),
				fmt.Sprintf("DA_AUTH_TOKEN=%s", cfg.DockerEnv["DA_AUTH_TOKEN"]),
				fmt.Sprintf("DA_NAMESPACE_ID=%s", daNamespaceID),
				fmt.Sprintf("FAUCET_ENABLED=%s", faucetEnabled),
				fmt.Sprintf("FAUCET_ADDRESS=%s", faucetAddress),
				fmt.Sprintf("FAUCET_AMOUNT=%s", faucetAmount),
				fmt.Sprintf("BASE_SHARD_ROUTER_KEY=%s", baseShardRouterKey),
				fmt.Sprintf("ROUTER_KEY=%s", routerKey),
				fmt.Sprintf("CHAIN_ID=%s", chainID),
				fmt.Sprintf("CHAIN_KEY_MNEMONIC=%s", chainKeyMnemonic),
			},
			ExposedPorts: getExposedPorts([]int{1317, 26657, 9090, 9601}),
		},
		HostConfig: container.HostConfig{
			PortBindings:  newPortMap([]int{1317, 26657, 9090, 9601, 8545}),
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
			NetworkMode:   container.NetworkMode(cfg.DockerEnv["CARDINAL_NAMESPACE"]),
		},
		Platform: platform,
	}
}
