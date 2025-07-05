package evm

import (
	"context"

	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
)

func (h *Handler) Stop(ctx context.Context, flags models.StopEVMFlags) error {
	cfg, err := config.GetConfig(&flags.Config)
	if err != nil {
		return err
	}

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	err = dockerClient.Stop(ctx, service.EVM, service.CelestiaDevNet)
	if err != nil {
		return err
	}

	printer.Infoln("EVM successfully stopped")
	return nil
}
