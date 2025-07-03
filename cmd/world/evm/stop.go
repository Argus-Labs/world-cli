package evm

import (
	"context"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
)

func Stop(c *StopCmd) error {
	cfg, err := config.GetConfig(&c.Parent.Config)
	if err != nil {
		return err
	}

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	ctx := context.Background()

	err = dockerClient.Stop(ctx, service.EVM, service.CelestiaDevNet)
	if err != nil {
		return err
	}

	printer.Infoln("EVM successfully stopped")
	return nil
}
