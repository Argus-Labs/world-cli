package cardinal

import (
	"context"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
)

func Restart(c *RestartCmd) error {
	cfg, err := config.GetConfig(&c.Parent.Config)
	if err != nil {
		return err
	}
	cfg.Build = true
	cfg.Debug = c.Debug
	cfg.Detach = c.Detach

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	ctx := context.Background()
	err = dockerClient.Restart(ctx, getServices(cfg)...)
	if err != nil {
		return err
	}

	return nil
}
