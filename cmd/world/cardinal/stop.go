package cardinal

import (
	"context"

	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
)

func (h *Handler) Stop(ctx context.Context, f models.StopCardinalFlags) error {
	cfg, err := config.GetConfig(&f.Config)
	if err != nil {
		return err
	}

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	err = dockerClient.Stop(ctx, service.Nakama, service.Cardinal,
		service.NakamaDB, service.Redis, service.Jaeger, service.Prometheus)
	if err != nil {
		return err
	}

	printer.Successln("Cardinal successfully stopped")

	return nil
}
