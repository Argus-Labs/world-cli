package cardinal

import (
	"context"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
)

type StopCmd struct {
}

func (c *StopCmd) Run() error {
	cfg, err := config.GetConfig()
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
	err = dockerClient.Stop(ctx, service.Nakama, service.Cardinal,
		service.NakamaDB, service.Redis, service.Jaeger, service.Prometheus)
	if err != nil {
		return err
	}

	printer.Successln("Cardinal successfully stopped")

	return nil
}
