package cardinal

import (
	"context"

	"pkg.world.dev/world-cli/internal/app/world-cli/common/config"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/docker"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

func (h *Handler) Restart(ctx context.Context, f models.RestartCardinalFlags) error {
	cfg, err := config.GetConfig(&f.Config)
	if err != nil {
		return err
	}
	cfg.Build = true
	cfg.Debug = f.Debug
	cfg.Detach = f.Detach

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	err = dockerClient.Restart(ctx, getServices(cfg)...)
	if err != nil {
		return err
	}

	return nil
}
