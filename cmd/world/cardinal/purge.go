package cardinal

import (
	"context"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/printer"
)

func Purge(c *PurgeCmd) error {
	cfg, err := config.GetConfig(&c.Parent.Config)
	if err != nil {
		return err
	}

	// Create a new Docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	ctx := context.Background()
	err = dockerClient.Purge(ctx, service.Nakama, service.Cardinal,
		service.NakamaDB, service.Redis, service.Jaeger, service.Prometheus)
	if err != nil {
		return err
	}
	printer.Infoln("Cardinal successfully purged")

	return nil
}
